package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	_ "embed"

	"cernobor.cz/oko-server/models"
	"crawshaw.io/sqlite"
	"crawshaw.io/sqlite/sqlitex"
	mbsh "github.com/consbio/mbtileserver/handlers"
	geojson "github.com/paulmach/go.geojson"
	"github.com/sirupsen/logrus"
)

//go:embed initdb.sql
var initDB string

type Server struct {
	config          ServerConfig
	dbpool          *sqlitex.Pool
	log             *logrus.Logger
	ctx             context.Context
	tileserverSvSet *mbsh.ServiceSet
}

type ServerConfig struct {
	DbPath       string
	TilepackPath string
	ApkPath      string
}

func New(dbPath, tilepackPath, apkPath string) *Server {
	return &Server{
		config: ServerConfig{
			DbPath:       dbPath,
			TilepackPath: tilepackPath,
			ApkPath:      apkPath,
		},
	}
}

func (s *Server) Run(ctx context.Context) {
	s.log = logrus.New()
	s.log.SetLevel(logrus.DebugLevel)

	s.ctx = ctx
	s.setupDB(true)
	s.setupTiles()

	router := s.setupRouter()

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", 8080),
		Handler: router,
	}
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.log.Infof("listen: %s\n", err)
		}
	}()

	<-s.ctx.Done()
	s.log.Info("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		s.log.WithError(err).Fatal("Server forced to shutdown.")
	}
	s.dbpool.Close()

	s.log.Info("Server exitting.")
}

func (s *Server) getDbConn() *sqlite.Conn {
	conn := s.dbpool.Get(s.ctx)
	_, err := conn.Prep("PRAGMA foreign_keys = ON").Step()
	if err != nil {
		panic(err)
	}
	return conn
}

func (s *Server) setupDB(reinit bool) {
	dbpool, err := sqlitex.Open(fmt.Sprintf("file:%s", s.config.DbPath), 0, 10)
	if err != nil {
		s.log.WithError(err).Fatal("Failed to open/create DB.")
	}
	s.dbpool = dbpool

	if reinit {
		conn := s.getDbConn()
		defer s.dbpool.Put(conn)

		err = sqlitex.ExecScript(conn, initDB)
		if err != nil {
			s.log.WithError(err).Fatal("init DB transaction failed")
		}
	}
}

func (s *Server) setupTiles() {
	tsRootURL, err := url.Parse(URITileserverRoot)
	if err != nil {
		s.log.WithError(err).Fatal("Failed to parse tileserver root URL.")
	}
	svs, err := mbsh.New(&mbsh.ServiceSetConfig{
		EnableServiceList: true,
		EnableTileJSON:    true,
		EnablePreview:     false,
		EnableArcGIS:      false,
		RootURL:           tsRootURL,
	})
	if err != nil {
		s.log.WithError(err).Fatal("Failed to create tileserver service set.")
	}
	err = svs.AddTileset(s.config.TilepackPath, "main")
	if err != nil {
		s.log.WithError(err).Fatal("Failed to register main tileset.")
	}
	s.tileserverSvSet = svs
}

func (s *Server) handshake(hc models.HandshakeChallenge) (models.UserID, error) {
	conn := s.getDbConn()
	defer s.dbpool.Put(conn)

	userID, err := func() (uid int, err error) {
		defer sqlitex.Save(conn)(&err)

		var id *int
		if hc.Exists {
			err = sqlitex.Exec(conn, "select id from users where name = ?", func(stmt *sqlite.Stmt) error {
				id = ptrInt(stmt.ColumnInt(0))
				return nil
			}, hc.Name)
			if sqlite.ErrCode(err) != sqlite.SQLITE_OK {
				return 0, err
			}
			if id == nil {
				return 0, ErrUserNotExists
			}
			if *id == 0 {
				return 0, ErrAttemptedSystemUser
			}
		} else {
			err = sqlitex.Exec(conn, "insert into users(name) values(?) returning id", func(stmt *sqlite.Stmt) error {
				id = ptrInt(stmt.ColumnInt(0))
				return nil
			}, hc.Name)
			if sqlite.ErrCode(err) == sqlite.SQLITE_CONSTRAINT_UNIQUE {
				return 0, ErrUserAlreadyExists
			}
			if sqlite.ErrCode(err) != sqlite.SQLITE_OK {
				return 0, err
			}
		}
		return *id, nil
	}()

	if err != nil {
		return 0, fmt.Errorf("failed to insert/retrieve user from db: %w", err)
	}
	return models.UserID(userID), nil
}

func (s *Server) getData() (models.Data, error) {
	conn := s.getDbConn()
	defer s.dbpool.Put(conn)

	return func() (data models.Data, err error) {
		defer sqlitex.Save(conn)(&err)

		people, err := s.getPeople(conn)
		if err != nil {
			return models.Data{}, fmt.Errorf("failed to retreive people: %w", err)
		}
		features, err := s.getFeatures(conn)
		if err != nil {
			return models.Data{}, fmt.Errorf("failed to retreive features: %w", err)
		}

		return models.Data{
			Users:    people,
			Features: features,
		}, nil
	}()
}

func (s *Server) update(data models.Update) error {
	conn := s.getDbConn()
	defer s.dbpool.Put(conn)

	return func() (err error) {
		defer sqlitex.Save(conn)(&err)

		if data.Create != nil {
			err = s.addFeatures(conn, data.Create)
			if err != nil {
				err = fmt.Errorf("failed to add features: %w", err)
				return
			}
		}
		if data.Update != nil {
			err = s.updateFeatures(conn, data.Update)
			if err != nil {
				err = fmt.Errorf("failed to update features: %w", err)
				return
			}
		}
		if data.Delete != nil {
			err = s.deleteFeatures(conn, data.Delete)
			if err != nil {
				err = fmt.Errorf("failed to delete features: %w", err)
				return
			}
		}

		return nil
	}()
}

func (s *Server) getPeople(conn *sqlite.Conn) ([]models.User, error) {
	if conn == nil {
		conn = s.getDbConn()
		defer s.dbpool.Put(conn)
	}

	users := make([]models.User, 0, 50)
	err := sqlitex.Exec(conn, "select id, name from users", func(stmt *sqlite.Stmt) error {
		users = append(users, models.User{
			ID:   models.UserID(stmt.ColumnInt(0)),
			Name: stmt.ColumnText(1),
		})
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get users from db: %w", err)
	}
	return users, nil
}

func (s *Server) getFeatures(conn *sqlite.Conn) ([]models.Feature, error) {
	if conn == nil {
		conn = s.getDbConn()
		defer s.dbpool.Put(conn)
	}

	features := make([]models.Feature, 0, 100)
	err := sqlitex.Exec(conn, "select id, owner_id, name, description, category, geom from features", func(stmt *sqlite.Stmt) error {
		id := stmt.ColumnInt(0)
		var ownerID *int
		if stmt.ColumnType(1) != sqlite.SQLITE_NULL {
			ownerID = ptrInt(stmt.ColumnInt(1))
		}
		name := stmt.ColumnText(2)
		var description *string
		if stmt.ColumnType(3) != sqlite.SQLITE_NULL {
			description = ptrString(stmt.ColumnText(3))
		}
		var category *string
		if stmt.ColumnType(4) != sqlite.SQLITE_NULL {
			category = ptrString(stmt.ColumnText(4))
		}
		geomRaw := stmt.ColumnText(5)

		var geom geojson.Geometry
		err := json.Unmarshal([]byte(geomRaw), &geom)
		if err != nil {
			return fmt.Errorf("failed to parse geometry for point id=%d: %w", id, err)
		}

		feature := models.Feature{
			ID:          models.FeatureID(id),
			OwnerID:     (*models.UserID)(ownerID),
			Name:        name,
			Description: description,
			Category:    category,
			Geometry:    geom,
		}

		features = append(features, feature)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get users from db: %w", err)
	}
	return features, nil
}

func (s *Server) addFeatures(conn *sqlite.Conn, features []models.Feature) error {
	stmt, err := conn.Prepare("insert into features(owner_id, name, description, category, geom) values(?, ?, ?, ?, ?)")
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Finalize()

	for _, feature := range features {
		err := stmt.Reset()
		if err != nil {
			return fmt.Errorf("failed to reset prepared statement: %w", err)
		}
		err = stmt.ClearBindings()
		if err != nil {
			return fmt.Errorf("failed to clear bindings of prepared statement: %w", err)
		}

		if feature.OwnerID == nil {
			stmt.BindNull(1)
		} else {
			stmt.BindInt64(1, int64(*feature.OwnerID))
		}
		stmt.BindText(2, feature.Name)
		if feature.Description == nil {
			stmt.BindNull(3)
		} else {
			stmt.BindText(3, *feature.Description)
		}
		if feature.Category == nil {
			stmt.BindNull(4)
		} else {
			stmt.BindText(4, *feature.Category)
		}
		geomBytes, err := json.Marshal(feature.Geometry)
		if err != nil {
			return fmt.Errorf("failed to marshal geometry of feature ID %d: %w", feature.ID, err)
		}
		stmt.BindText(5, string(geomBytes))

		_, err = stmt.Step()
		if err != nil {
			return fmt.Errorf("failed to evaluate prepared statement: %w", err)
		}
	}
	return nil
}

func (s *Server) updateFeatures(conn *sqlite.Conn, features []models.Feature) error {
	stmt, err := conn.Prepare("update features set owner_id = ?, name = ?, description = ?, category = ?, geom = ? where id = ?")
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Finalize()

	for _, feature := range features {
		err := stmt.Reset()
		if err != nil {
			return fmt.Errorf("failed to reset prepared statement: %w", err)
		}
		err = stmt.ClearBindings()
		if err != nil {
			return fmt.Errorf("failed to clear bindings of prepared statement: %w", err)
		}

		if feature.OwnerID == nil {
			stmt.BindNull(1)
		} else {
			stmt.BindInt64(1, int64(*feature.OwnerID))
		}
		stmt.BindText(2, feature.Name)
		if feature.Description == nil {
			stmt.BindNull(3)
		} else {
			stmt.BindText(3, *feature.Description)
		}
		if feature.Category == nil {
			stmt.BindNull(4)
		} else {
			stmt.BindText(4, *feature.Category)
		}
		geomBytes, err := json.Marshal(feature.Geometry)
		if err != nil {
			return fmt.Errorf("failed to marshal geometry of feature ID %d: %w", feature.ID, err)
		}
		stmt.BindText(5, string(geomBytes))
		stmt.BindInt64(6, int64(feature.ID))

		_, err = stmt.Step()
		if err != nil {
			return fmt.Errorf("failed to evaluate prepared statement: %w", err)
		}
	}
	return nil
}

func (s *Server) deleteFeatures(conn *sqlite.Conn, featureIDs []models.FeatureID) error {
	stmt, err := conn.Prepare("delete from features where id = ?")
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Finalize()

	for _, featureID := range featureIDs {
		err := stmt.Reset()
		if err != nil {
			return fmt.Errorf("failed to reset prepared statement: %w", err)
		}
		err = stmt.ClearBindings()
		if err != nil {
			return fmt.Errorf("failed to clear bindings of prepared statement: %w", err)
		}

		stmt.BindInt64(1, int64(featureID))

		_, err = stmt.Step()
		if err != nil {
			return fmt.Errorf("failed to evaluate prepared statement: %w", err)
		}
	}
	return nil
}
