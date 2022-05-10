package main

import (
	"flag"
	"fmt"
	_ "net/http/pprof"
	"os"
	"runtime/debug"
	"time"

	. "github.com/foomo/contentserver/logger"
	"github.com/foomo/contentserver/metrics"
	"github.com/foomo/contentserver/server"
	"github.com/foomo/contentserver/status"
	"go.uber.org/zap"
)

const (
	ServiceName                  = "Content Server"
	DefaultHealthzHandlerAddress = ":8080"
	DefaultPrometheusListener    = ":9200"
)

var (
	flagAddress                   = flag.String("address", "", "address to bind socket server host:port")
	flagWebserverAddress          = flag.String("webserver-address", "", "address to bind web server host:port, when empty no webserver will be spawned")
	flagWebserverPath             = flag.String("webserver-path", "/contentserver", "path to export the webserver on - useful when behind a proxy")
	flagVarDir                    = flag.String("var-dir", "/var/lib/contentserver", "where to put my data")
	flagPrometheusListener        = flag.String("prometheus-listener", getenv("PROMETHEUS_LISTENER", DefaultPrometheusListener), "address for the prometheus listener")
	flagRepositoryTimeoutDuration = flag.Duration("repository-timeout-duration", server.DefaultRepositoryTimeout, "timeout duration for the contentserver")

	// debugging / profiling
	flagDebug     = flag.Bool("debug", false, "toggle debug mode")
	flagFreeOSMem = flag.Int("free-os-mem", 0, "free OS mem every X minutes")
	flagHeapDump  = flag.Int("heap-dump", 0, "dump heap every X minutes")
)

func getenv(env, fallback string) string {
	if value, ok := os.LookupEnv(env); ok {
		return value
	}
	return fallback
}

func exitUsage(code int) {
	fmt.Println("Usage:", os.Args[0], "http(s)://your-content-server/path/to/content.json")
	flag.PrintDefaults()
	os.Exit(code)
}

func main() {
	flag.Parse()

	SetupLogging(*flagDebug, "contentserver.log")

	if *flagFreeOSMem > 0 {
		Log.Info("freeing OS memory every $interval minutes", zap.Int("interval", *flagFreeOSMem))
		go func() {
			for {
				select {
				case <-time.After(time.Duration(*flagFreeOSMem) * time.Minute):
					Log.Info("FreeOSMemory")
					debug.FreeOSMemory()
				}
			}
		}()
	}

	if *flagHeapDump > 0 {
		Log.Info("dumping heap every $interval minutes", zap.Int("interval", *flagHeapDump))
		go func() {
			for {
				select {
				case <-time.After(time.Duration(*flagFreeOSMem) * time.Minute):
					Log.Info("HeapDump")
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

		// kickoff metric handlers
		go metrics.RunPrometheusHandler(*flagPrometheusListener)
		go status.RunHealthzHandlerListener(DefaultHealthzHandlerAddress, ServiceName)

		err := server.RunServerSocketAndWebServer(flag.Arg(0), *flagAddress, *flagWebserverAddress, *flagWebserverPath, *flagVarDir, *flagRepositoryTimeoutDuration)
		if err != nil {
			fmt.Println("exiting with error", err)
			os.Exit(1)
		}
	} else {
		exitUsage(1)
	}
}
