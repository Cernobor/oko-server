package server

import (
	"embed"
	"fmt"
	"os"
	"path"
	"regexp"
	"sort"
	"strconv"
	"time"

	"zombiezen.com/go/sqlite"
	"zombiezen.com/go/sqlite/sqlitemigration"
	"zombiezen.com/go/sqlite/sqlitex"
)

func (s *Server) cleanupDb() {
	close(s.checkpointNotice)
	s.log.Info("Closing db connection pool...")
	s.dbpool.Close()

	// manually force truncate checkpoint
	conn, err := sqlite.OpenConn(fmt.Sprintf("file:%s", s.config.DbPath), 0)
	if err != nil {
		s.log.WithError(err).Error("Failed to open connection for final checkpoint.")
		return
	}
	err = sqlitex.Execute(conn, "vacuum", nil)
	if err != nil {
		s.log.WithError(err).Error("Failed to vacuum db.")
	}
	s.checkpointDb(conn, true)
	conn.Close()
}

func (s *Server) getDbConn() *sqlite.Conn {
	conn, err := s.dbpool.Get(s.ctx)
	if err != nil {
		panic(err)
	}
	return conn
}

func (s *Server) returnDbConn(conn *sqlite.Conn) {
	s.dbpool.Put(conn)
}

func withDbConn[T any](s *Server, f func(conn *sqlite.Conn) T) T {
	conn := s.getDbConn()
	defer s.returnDbConn(conn)
	return f(conn)
}

func (s *Server) checkpointDb(conn *sqlite.Conn, truncate bool) {
	var query string
	if truncate {
		query = "PRAGMA wal_checkpoint(TRUNCATE)"
	} else {
		query = "PRAGMA wal_checkpoint(RESTART)"
	}
	stmt, _, err := conn.PrepareTransient(query)
	if err != nil {
		s.log.WithError(err).Error("Failed to prepare checkpoint query.")
		return
	}
	defer stmt.Finalize()

	has, err := stmt.Step()
	if err != nil {
		s.log.WithError(err).Error("Failed to step through checkpoint query.")
		return
	}
	if !has {
		s.log.Error("Checkpoint query returned no rows.")
		return
	}

	blocked := stmt.ColumnInt(0)
	noWalPages := stmt.ColumnInt(1)
	noReclaimedPages := stmt.ColumnInt(2)
	if blocked == 1 {
		s.log.Warn("Checkpoint query was blocked.")
	}
	s.log.Debugf("Checkpoint complete. %d pages written to WAL, %d pages written back to DB.", noWalPages, noReclaimedPages)
}

func (s *Server) setupDB() error {
	s.dbAvailable.Store(false)
	s.log.Debugf("Using db %s", s.config.DbPath)

	ready := make(chan struct{})
	migErr := make(chan error)
	s.dbpool = sqlitemigration.NewPool(fmt.Sprintf("file:%s", s.config.DbPath), sqlSchema, sqlitemigration.Options{
		PoolSize: 10,
		PrepareConn: func(conn *sqlite.Conn) error {
			return sqlitex.ExecuteTransient(conn, "PRAGMA foreign_keys = ON;", nil)
		},
		OnReady: func() {
			close(ready)
		},
		OnError: func(err error) {
			migErr <- err
		},
	})
	select {
	case <-ready:
	case err := <-migErr:
		return fmt.Errorf("error during db migration: %w", err)
	}
	s.checkpointNotice = make(chan struct{})

	// aggressively checkpoint the database on idle times
	go func() {
		s.log.Debug("Starting manual restart checkpointing.")
		defer s.log.Debug("Manual restart checkpointing stopped.")
		delay := time.Minute * 15
		var (
			timer <-chan time.Time
			ok    bool
		)
		for {
			select {
			case _, ok = <-s.checkpointNotice:
				if !ok {
					return
				}
				timer = time.After(delay)
			case <-timer:
				withDbConn(s, func(conn *sqlite.Conn) any {
					s.checkpointDb(conn, false)
					return nil
				})
				timer = nil
			}
		}
	}()
	s.dbAvailable.Store(true)
	return nil
}

func (s *Server) requestCheckpoint() {
	go func() {
		s.checkpointNotice <- struct{}{}
	}()
}

func (s *Server) reinitDb() error {
	s.log.Debug("Reinitializing db.")
	s.dbAvailable.Store(false)
	defer s.dbAvailable.Store(true)
	close(s.checkpointNotice)
	err := s.dbpool.Close()
	if err != nil {
		return fmt.Errorf("failed to close db: %w", err)
	}
	s.log.Debug("Removing main db file.")
	err = os.Remove(s.config.DbPath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove db file %s: %w", s.config.DbPath, err)
	}
	s.log.Debug("Removing WAL file.")
	err = os.Remove(s.config.DbPath + "-wal")
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove db-wal file %s-wal: %w", s.config.DbPath, err)
	}
	s.log.Debug("Initializing db.")
	err = s.setupDB()
	if err != nil {
		return fmt.Errorf("failed to setup db during reinit")
	}
	s.log.Debug("DB reinitialized.")
	return nil
}

// SQL schema

//go:embed sql_schema/V*.sql
var sqlSchemaFiles embed.FS
var sqlSchema sqlitemigration.Schema = func() sqlitemigration.Schema {
	type migration struct {
		content string
		version int
		name    string
	}

	entries, err := sqlSchemaFiles.ReadDir("sql_schema")
	if err != nil {
		panic(fmt.Errorf("failed to read sql_schema migrations: %w", err))
	}

	pattern := regexp.MustCompile("^V([0-9]+)_(.*)[.][sS][qQ][lL]$")
	migrations := []*migration{}
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() {
			panic(fmt.Errorf("embedded sql migration '%s' is a directory", name))
		}
		matches := pattern.FindStringSubmatch(name)
		if matches == nil {
			panic(fmt.Errorf("embedded sql migration '%s' does not match the filename pattern", name))
		}
		if len(matches) != 3 {
			panic(fmt.Errorf("embedded sql migration '%s' does not have the correct number of submatches", name))
		}
		version, err := strconv.Atoi(matches[1])
		if err != nil {
			panic(fmt.Errorf("failed to parse version number of migration '%s': %w", name, err))
		}
		migName := matches[2]
		file := path.Join("sql_schema", name)
		content, err := sqlSchemaFiles.ReadFile(file)
		if err != nil {
			panic(fmt.Errorf("failed to read embedded migration %s", entry.Name()))
		}
		migrations = append(migrations, &migration{
			content: string(content),
			version: version,
			name:    migName,
		})
	}
	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].version < migrations[j].version
	})

	return sqlitemigration.Schema{
		Migrations: Map(func(m *migration) string { return m.content }, migrations),
	}
}()
