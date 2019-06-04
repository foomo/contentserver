package main

import (
	"flag"
	"log"
	"net/http"
)

type testServer struct {
	file string
}

func main() {

	var (
		flagJSONFile = flag.String("json-file", "", "provide a json source file")
		flagAddress  = flag.String("addr", ":1234", "set the webserver address")
	)
	flag.Parse()

	if *flagJSONFile == "" {
		log.Fatal("js source file must be provided")
	}

	ts := &testServer{
		file: *flagJSONFile,
	}

	log.Println("start test server at", *flagAddress, "serving file:", ts.file)
	log.Fatal(http.ListenAndServe(*flagAddress, ts))
}

func (ts *testServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, ts.file)
}
