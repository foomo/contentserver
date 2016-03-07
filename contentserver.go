package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/foomo/contentserver/log"
	"github.com/foomo/contentserver/server"
)

const (
	logLevelDebug   = "debug"
	logLevelNotice  = "notice"
	logLevelWarning = "warning"
	logLevelRecord  = "record"
	logLevelError   = "error"
)

var (
	uniqushPushVersion = "content-server 1.3.0"
	showVersionFlag    = flag.Bool("version", false, "version info")
	address            = flag.String("address", "127.0.0.1:8081", "address to bind host:port")
	varDir             = flag.String("var-dir", "/var/lib/contentserver", "where to put my data")
	logLevelOptions    = []string{
		logLevelError,
		logLevelRecord,
		logLevelWarning,
		logLevelNotice,
		logLevelDebug,
	}
	logLevel = flag.String(
		"log-level",
		logLevelRecord,
		fmt.Sprintf(
			"one of %s",
			strings.Join(logLevelOptions, ", "),
		),
	)
)

func exitUsage(code int) {
	fmt.Printf("Usage: %s http(s)://your-content-server/path/to/content.json\n", os.Args[0])
	flag.PrintDefaults()
	os.Exit(code)
}

func main() {
	flag.Parse()
	if *showVersionFlag {
		fmt.Printf("%v\n", uniqushPushVersion)
		return
	}
	if len(flag.Args()) == 1 {
		fmt.Println(*address, flag.Arg(0))
		level := log.LevelRecord
		switch *logLevel {
		case logLevelError:
			level = log.LevelError
		case logLevelRecord:
			level = log.LevelRecord
		case logLevelWarning:
			level = log.LevelWarning
		case logLevelNotice:
			level = log.LevelNotice
		case logLevelDebug:
			level = log.LevelDebug

		}
		log.SelectedLevel = level
		err := server.Run(flag.Arg(0), *address, *varDir)
		if err != nil {
			fmt.Println("exiting with error", err)
			os.Exit(1)
		}
	} else {
		exitUsage(1)
	}
}
