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

	"net/http"
	_ "net/http/pprof"

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

	flagShowVersionFlag  = flag.Bool("version", false, "version info")
	flagAddress          = flag.String("address", "", "address to bind socket server host:port")
	flagWebserverAddress = flag.String("webserver-address", "", "address to bind web server host:port, when empty no webserver will be spawned")
	flagWebserverPath    = flag.String("webserver-path", "/contentserver", "path to export the webserver on - useful when behind a proxy")
	flagVarDir           = flag.String("var-dir", "/var/lib/contentserver", "where to put my data")

	// debugging / profiling
	flagFreeOSMem = flag.Int("free-os-mem", 0, "free OS mem every X minutes")
	flagHeapDump  = flag.Int("heap-dump", 0, "dump heap every X minutes")

	logLevelOptions = []string{
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

	go func() {
		fmt.Println(http.ListenAndServe("localhost:6060", nil))
	}()

	if *flagShowVersionFlag {
		fmt.Printf("%v\n", uniqushPushVersion)
		return
	}

	if *flagFreeOSMem > 0 {
		log.Notice("[INFO] freeing OS memory every ", *flagFreeOSMem, " minutes!")
		go func() {
			for {
				select {
				case <-time.After(time.Duration(*flagFreeOSMem) * time.Minute):
					log.Notice("FreeOSMemory")
					debug.FreeOSMemory()
				}
			}
		}()
	}

	if *flagHeapDump > 0 {
		log.Notice("[INFO] dumping heap every ", *flagHeapDump, " minutes!")
		go func() {
			for {
				select {
				case <-time.After(time.Duration(*flagFreeOSMem) * time.Minute):
					log.Notice("HeapDump")
					f, err := os.Create("heapdump")
					if err != nil {
						panic("failed to create heap dump file")
					}
					debug.WriteHeapDump(f.Fd())
					err = f.Close()
					if err != nil {
						panic("failed to create heap dump file")
					}
				}
			}
		}()
	}

	if len(flag.Args()) == 1 {
		fmt.Println(*flagAddress, flag.Arg(0))

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

		// kickoff metric handlers
		go metrics.RunPrometheusHandler(DefaultPrometheusListener)
		go status.RunHealthzHandlerListener(DefaultHealthzHandlerAddress, ServiceName)

		err := server.RunServerSocketAndWebServer(flag.Arg(0), *flagAddress, *flagWebserverAddress, *flagWebserverPath, *flagVarDir)
		if err != nil {
			fmt.Println("exiting with error", err)
			os.Exit(1)
		}
	} else {
		exitUsage(1)
	}
}
