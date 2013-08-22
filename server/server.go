package server

import (
	"fmt"
	"github.com/foomo/ContentServer/server/repo"
	"github.com/foomo/ContentServer/server/requests"
	"github.com/foomo/ContentServer/server/utils"
	"net/http"
	"strings"
)

var i int = 0
var contentRepo *repo.Repo

func contentHandler(w http.ResponseWriter, r *http.Request) {
	request := requests.NewContent()
	utils.PopulateRequest(r, request)
	utils.JsonResponse(w, contentRepo.GetContent(request))
}

func wtfHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "thank you for your request, but i am totally lost with it\n", r.RequestURI, "\n")
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
	fmt.Println("staring content server")
	fmt.Printf("  loading content from %s\n", serverUrl)
	contentRepo = repo.NewRepo(serverUrl)
	contentRepo.Update()
	fmt.Printf("    loaded %d items\n", len(contentRepo.Directory))
	http.HandleFunc("/content", contentHandler)
	http.HandleFunc("/cmd/", commandHandler)
	http.HandleFunc("/", wtfHandler)
	fmt.Printf("  starting service on %s\n", addr)
	http.ListenAndServe(addr, nil)
}
