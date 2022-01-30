package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"cernobor.cz/oko-server/server"
	"github.com/sirupsen/logrus"
)

func main() {
	dbFileArg := flag.String("dbfile", "./data.sqlite3", "File that holds the server's sqlite3 database. Will be created if it does not exist. Default is \"./data.sqlite3\".")
	tilepackFileArg := flag.String("tilepack", "", "File that will be sent to clients when they request a tile pack. Required.")
	apkFileArg := flag.String("apk", "", "APK file with the client app. If not specified, no APK will be available (404).")

	flag.Parse()

	if *tilepackFileArg == "" {
		fmt.Fprintln(os.Stderr, "Tile pack not specified.")
		flag.Usage()
		os.Exit(1)
	}

	s := server.New(*dbFileArg, *tilepackFileArg, *apkFileArg)

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
