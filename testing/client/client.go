package main

import (
	"flag"
	"log"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/foomo/contentserver/client"
)

// ***REMOVED***
var (
	flagAddr    = flag.String("addr", "http://127.0.0.1:9191/contentserver", "set addr")
	flagGetRepo = flag.Bool("getRepo", false, "get repo")
	flagUpdate  = flag.Bool("update", true, "trigger content update")
	flagNum     = flag.Int("num", 100, "num repititions")
	flagDelay   = flag.Int("delay", 2, "delay in seconds")
)

func main() {

	flag.Parse()

	c, errClient := client.NewHTTPClient(*flagAddr)
	if errClient != nil {
		log.Fatal(errClient)
	}

	for i := 1; i <= *flagNum; i++ {

		if *flagUpdate {
			go func(num int) {
				log.Println("start update")
				resp, errUpdate := c.Update()
				if errUpdate != nil {
					spew.Dump(resp)
					log.Fatal(errUpdate)
				}
				log.Println(num, "update done", resp)
			}(i)
		}

		if *flagGetRepo {
			go func(num int) {
				log.Println("get repo", num)
				_, err := c.GetRepo()
				if err != nil {
					// spew.Dump(resp)
					log.Fatal("failed to get repo")
				}
				log.Println(num, "get repo done")
			}(i)
		}

		time.Sleep(time.Duration(*flagDelay) * time.Second)
	}

	log.Println("done")
}
