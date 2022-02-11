package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"cernobor.cz/oko-server/models"
	"cernobor.cz/oko-server/server"
	"github.com/sirupsen/logrus"
)

func main() {
	tilepackFileArg := flag.String("tilepack", "", "File that will be sent to clients when they request a tile pack, also used to serve tiles in online mode. Required.")
	portArg := flag.Int("port", 8080, "Port where the server will listen to. Default is 8080.")
	dbFileArg := flag.String("dbfile", "./data.sqlite3", "File that holds the server's sqlite3 database. Will be created if it does not exist. Default is \"./data.sqlite3\".")
	apkFileArg := flag.String("apk", "", "APK file with the client app. If not specified, no APK will be available (404).")
	reinitDBArg := flag.Bool("reinit-db", false, "Reinitializes the DB, which means all the tables will be recreated, deleting all data.")
	minZoomArg := flag.Int("min-zoom", 1, "Minimum zoom that will be sent to clients.")
	defaultCenterLatArg := flag.Float64("default-center-lat", 0, "Latitude of the default map center.")
	defaultCenterLngArg := flag.Float64("default-center-lng", 0, "Longitude of the default map center.")

	flag.Parse()

	if *tilepackFileArg == "" {
		fmt.Fprintln(os.Stderr, "Tile pack not specified.")
		flag.Usage()
		os.Exit(1)
	}

	s := server.New(server.ServerConfig{
		Port:         *portArg,
		DbPath:       *dbFileArg,
		TilepackPath: *tilepackFileArg,
		ApkPath:      *apkFileArg,
		ReinitDB:     *reinitDBArg,
		MinZoom:      *minZoomArg,
		DefaultCenter: models.Coords{
			Lat: *defaultCenterLatArg,
			Lng: *defaultCenterLngArg,
		},
	})

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT)
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		sig := <-sigs
		logrus.Warnf("Received signal: %s", sig)
		cancel()
	}()
	s.Run(ctx)
}
