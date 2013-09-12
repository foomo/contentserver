package server

import (
	"fmt"
	"github.com/foomo/ContentServer/server/log"
	"github.com/foomo/ContentServer/server/repo"
	"github.com/foomo/ContentServer/server/requests"
	"github.com/foomo/ContentServer/server/utils"
	"net/http"
	"strconv"
	"strings"
)

func contentHandler(w http.ResponseWriter, r *http.Request) {
	request := requests.NewContent()
	utils.PopulateRequest(r, request)
	utils.JsonResponse(w, contentRepo.GetContent(request))
}

func uriHandler(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.RequestURI, "/")
	if len(parts) == 5 {
		utils.JsonResponse(w, contentRepo.GetURI(parts[2], parts[3], parts[4]))
	} else {
		wtfHandler(w, r)
	}
}

func wtfHandler(w http.ResponseWriter, r *http.Request) {
	msg := "unhandled request: " + r.RequestURI
	fmt.Fprint(w, msg, "\n")
	log.Error(msg)
}

func update() interface{} {
	contentRepo.Update()
	return contentRepo.Directory
}

func commandHandler(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.RequestURI, "/")
	switch true {
	case true == (len(parts) > 1):
		switch parts[2] {
		case "update":
			utils.JsonResponse(w, update())
			return
		default:
			help := make(map[string]interface{})
			help["input"] = parts[2]
			help["commands"] = []string{"update", "help", "content"}
			utils.JsonResponse(w, help)
		}
	default:
		wtfHandler(w, r)
	}
}

func Run(addr string, serverUrl string) {
	log.Notice("starting content server")
	log.Notice("  loading content from " + serverUrl)
	contentRepo = repo.NewRepo(serverUrl)
	contentRepo.Update()
	log.Notice("    loaded " + strconv.Itoa(len(contentRepo.Directory)) + " items")
	http.HandleFunc("/content", contentHandler)
	http.HandleFunc("/uri/", uriHandler)
	http.HandleFunc("/cmd/", commandHandler)
	http.HandleFunc("/", wtfHandler)
	log.Notice("  starting service on " + addr)
	http.ListenAndServe(addr, nil)
}
