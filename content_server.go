package main

import (
	"flag"
	"fmt"
	"github.com/foomo/contentserver/server"
	"github.com/foomo/contentserver/server/log"
	"os"
	"strings"
)

const (
	PROTOCOL_TCP = "tcp"
)

type ExitCode int

const (
	EXIT_CODE_OK                = 0
	EXIT_CODE_INSUFFICIENT_ARGS = 1
)

var contentServer string

var uniqushPushVersion = "content-server 1.2.0"

var showVersionFlag = flag.Bool("version", false, "Version info")
var protocol = flag.String("protocol", PROTOCOL_TCP, "what protocol to server for")
var address = flag.String("address", "127.0.0.1:8081", "address to bind host:port")
var varDir = flag.String("vardir", "127.0.0.1:8081", "where to put my data")
var logLevelOptions = []string{
	log.LOG_LEVEL_NAME_ERROR,
	log.LOG_LEVEL_NAME_RECORD,
	log.LOG_LEVEL_NAME_WARNING,
	log.LOG_LEVEL_NAME_NOTICE,
	log.LOG_LEVEL_NAME_DEBUG}

var logLevel = flag.String(
	"logLevel",
	log.LOG_LEVEL_NAME_RECORD,
	fmt.Sprintf(
		"one of %s",
		strings.Join(logLevelOptions, ", ")))

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
		fmt.Println(*protocol, *address, flag.Arg(0))
		log.SetLogLevel(log.GetLogLevelByName(*logLevel))
		switch *protocol {
		case PROTOCOL_TCP:
			server.RunSocketServer(flag.Arg(0), *address, *varDir)
			break
		default:
			exitUsage(EXIT_CODE_INSUFFICIENT_ARGS)
		}
	} else {
		exitUsage(EXIT_CODE_INSUFFICIENT_ARGS)
	}
}
