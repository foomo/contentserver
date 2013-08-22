package main

import (
	"github.com/foomo/ContentServer/server"
)

func main() {
	server.Run(":8080", "http://test.bestbytes/foomo/modules/Foomo.Page.Content/services/content.php")
}
