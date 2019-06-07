package server

import (
	"net/http"
	"strings"
	"time"

	"github.com/foomo/contentserver/repo"
)

const sourceWebserver = "webserver"

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

	h := Handler(strings.TrimPrefix(r.URL.Path, s.path+"/"))
	if h == HandlerGetRepo {
		start := time.Now()
		w.Header().Set("Content-Type", "application/json")
		s.r.WriteRepoBytes(w)
		addMetrics(h, start, nil, nil, sourceWebserver)
		return
	}

	if err := handleRequest(s.r, h, r.Body, w, "webserver"); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	r.Body.Close()
}
