package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"cernobor.cz/oko-server/models"
	"cernobor.cz/oko-server/server"
	"github.com/sirupsen/logrus"
)

var (
	sha1ver   string
	buildTime string
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
	maxPhotoXArg := flag.Int("max-photo-width", 0, "Maximum width of photos. 0 means no limit.")
	maxPhotoYArg := flag.Int("max-photo-height", 0, "Maximum height of photos. 0 means no limit.")
	photoQualityArg := flag.Int("photo-quality", 90, "Photo JPEG quality.")

	flag.Parse()

	if *tilepackFileArg == "" {
		fmt.Fprintln(os.Stderr, "Tile pack not specified.")
		flag.Usage()
		os.Exit(1)
	}

	t, err := time.Parse(time.RFC3339, buildTime)
	if err != nil {
		t = time.Now()
	}

	if *maxPhotoXArg < 0 || *maxPhotoYArg < 0 {
		fmt.Fprintln(os.Stderr, "Max photo width and height cannot be less than 0.")
		os.Exit(1)
	}

	s := server.New(server.ServerConfig{
		VersionHash:  sha1ver,
		BuildTime:    &t,
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
		MaxPhotoX:    *maxPhotoXArg,
		MaxPhotoY:    *maxPhotoYArg,
		PhotoQuality: *photoQualityArg,
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
