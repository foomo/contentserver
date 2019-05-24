package server

import (
	"io/ioutil"
	"net/http"
	"strings"

	"go.uber.org/zap"

	. "github.com/foomo/contentserver/logger"
	"github.com/foomo/contentserver/repo"
)

type webServer struct {
	path string
	r    *repo.Repo
}

// NewWebServer returns a shiny new web server
func NewWebServer(path string, r *repo.Repo) http.Handler {
	return &webServer{
		path: path,
		r:    r,
	}
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
	h := Handler(strings.TrimPrefix(r.URL.Path, s.path+"/"))
	if h == HandlerGetRepo {
		s.r.WriteRepoBytes(w)
		w.Header().Set("Content-Type", "application/json")
		return
	}
	reply, errReply := handleRequest(s.r, h, jsonBytes, "webserver")
	if errReply != nil {
		http.Error(w, errReply.Error(), http.StatusInternalServerError)
		return
	}
	_, err := w.Write(reply)
	if err != nil {
		Log.Error("failed to write webServer reply", zap.Error(err))
	}
}
