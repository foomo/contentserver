package main

import (
	"flag"
	"fmt"
	"github.com/foomo/ContentServer/server"
	"github.com/foomo/ContentServer/server/log"
	"os"
)

const (
	PROTOCOL_TCP  = "tcp"
	PROTOCOL_HTTP = "http"
)

type ExitCode int

const (
	EXIT_CODE_OK                = 0
	EXIT_CODE_INSUFFICIENT_ARGS = 1
)

var contentServer string

var protocol = flag.String("protocol", PROTOCOL_TCP, "what protocol to server for")
var address = flag.String("address", "127.0.0.1:8081", "address to bind host:port")

func exitUsage(code int) {
	fmt.Printf("Usage: %s http(s)://your-content-server/path/to/content.json\n", os.Args[0])
	flag.PrintDefaults()
	os.Exit(code)
}

func main() {
	flag.Parse()
	log.SetLogLevel(log.LOG_LEVEL_DEBUG)
	if len(flag.Args()) == 1 {
		fmt.Println(*protocol, *address, flag.Arg(0))
		//server.Run(":8080", "http://test.bestbytes/foomo/modules/Foomo.Page.Content/services/content.php")
		switch *protocol {
		case PROTOCOL_TCP:
			server.RunSocketServer(flag.Arg(0), *address)
			break
		case PROTOCOL_HTTP:
			fmt.Println("http server does not work yet - use tcp instead")
			break
		default:
			exitUsage(EXIT_CODE_INSUFFICIENT_ARGS)
		}
	} else {
		exitUsage(EXIT_CODE_INSUFFICIENT_ARGS)
	}
}
