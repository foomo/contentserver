package main

import (
	"flag"
	"fmt"
	"os"
	"runtime/debug"
	"strings"
	"time"

	"github.com/foomo/contentserver/metrics"
	"github.com/foomo/contentserver/status"

	"github.com/foomo/contentserver/log"
	"github.com/foomo/contentserver/server"
)

const (
	logLevelDebug   = "debug"
	logLevelNotice  = "notice"
	logLevelWarning = "warning"
	logLevelRecord  = "record"
	logLevelError   = "error"

	ServiceName                  = "Content Server"
	DefaultHealthzHandlerAddress = ":8080"
	DefaultPrometheusListener    = ":9200"
)

var (
	uniqushPushVersion = "content-server 1.5.0"
	showVersionFlag    = flag.Bool("version", false, "version info")
	address            = flag.String("address", "", "address to bind socket server host:port")
	webserverAddress   = flag.String("webserver-address", "", "address to bind web server host:port, when empty no webserver will be spawned")
	webserverPath      = flag.String("webserver-path", "/contentserver", "path to export the webserver on - useful when behind a proxy")
	varDir             = flag.String("var-dir", "/var/lib/contentserver", "where to put my data")
	flagFreeOSMem      = flag.Int("free-os-mem", 0, "free OS mem every X minutes")
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
	fmt.Println("Usage:", os.Args[0], "http(s)://your-content-server/path/to/content.json")
	flag.PrintDefaults()
	os.Exit(code)
}

func main() {
	flag.Parse()

	if *showVersionFlag {
		fmt.Printf("%v\n", uniqushPushVersion)
		return
	}

	if *flagFreeOSMem > 0 {
		log.Notice("[INFO] freeing OS memory every ", *flagFreeOSMem, " minutes!")
		go func() {
			for _ = range time.After(time.Duration(*flagFreeOSMem) * time.Minute) {
				log.Notice("FreeOSMemory")
				debug.FreeOSMemory()
			}
		}()
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
		go metrics.RunPrometheusHandler(DefaultPrometheusListener)
		go status.RunHealthzHandlerListener(DefaultHealthzHandlerAddress, ServiceName)

		err := server.RunServerSocketAndWebServer(flag.Arg(0), *address, *webserverAddress, *webserverPath, *varDir)
		if err != nil {
			fmt.Println("exiting with error", err)
			os.Exit(1)
		}
	} else {
		exitUsage(1)
	}
}
