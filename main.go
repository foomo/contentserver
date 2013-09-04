package main

import (
	"github.com/foomo/ContentServer/server"
	"github.com/foomo/ContentServer/server/log"
)

func main() {
	log.SetLogLevel(log.LOG_LEVEL_DEBUG)
	//server.Run(":8080", "http://test.bestbytes/foomo/modules/Foomo.Page.Content/services/content.php")
	server.RunSocketServer("http://test.bestbytes/foomo/modules/Foomo.Page.Content/services/content.php", "0.0.0.0:8081")
}
