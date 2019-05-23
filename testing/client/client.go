package main

import (
	"log"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/foomo/contentserver/client"
)

func main() {
	serverAdr := "http://127.0.0.1:9191/contentserver"
	c, errClient := client.NewHTTPClient(serverAdr)
	if errClient != nil {
		log.Fatal(errClient)
	}

	for i := 1; i <= 50; i++ {
		go func(num int) {
			log.Println("start update")
			resp, errUpdate := c.Update()
			if errUpdate != nil {
				spew.Dump(resp)
				log.Fatal(errUpdate)
			}
			log.Println(num, "update done", resp)
		}(i)
		time.Sleep(5 * time.Second)
	}

	log.Println("done")
}
