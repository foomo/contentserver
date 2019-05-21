package server

import (
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/foomo/contentserver/log"
	"github.com/foomo/contentserver/status"

	"github.com/foomo/contentserver/repo"
)

type webServer struct {
	r       *repo.Repo
	path    string
	metrics *status.Metrics
}

// NewWebServer returns a shiny new web server
func NewWebServer(path string, r *repo.Repo) (s http.Handler, err error) {
	s = &webServer{
		r:       r,
		path:    path,
		metrics: status.NewMetrics("webserver"),
	}
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
	reply, errReply := handleRequest(s.r, Handler(strings.TrimPrefix(r.URL.Path, s.path+"/")), jsonBytes, s.metrics)
	if errReply != nil {
		http.Error(w, errReply.Error(), http.StatusInternalServerError)
		return
	}
	_, err := w.Write(reply)
	if err != nil {
		log.Error("failed to write webServer reply: ", err)
	}
}
