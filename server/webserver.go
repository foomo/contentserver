package server

import (
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/foomo/contentserver/repo"
)

const PathContentserver = "/contentserver"

type webServer struct {
	r *repo.Repo
}

func newWebServer() (s *webServer, err error) {
	s = &webServer{}
	return
}

func (s *webServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Body == nil {
		http.Error(w, "no body", http.StatusBadRequest)
		return
	}
	jsonBytes, readErr := ioutil.ReadAll(r.Body)
	r.Body.Close()
	if readErr != nil {
		http.Error(w, "failed to read incoming request", http.StatusBadRequest)
		return
	}
	reply, errReply := handleRequest(s.r, Handler(strings.TrimPrefix(r.URL.Path, PathContentserver+"/")), jsonBytes)
	if errReply != nil {
		http.Error(w, errReply.Error(), http.StatusInternalServerError)
		return
	}
	w.Write(reply)
}
