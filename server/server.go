package server

import (
	"encoding/json"
	"fmt"
	//"github.com/foomo/ContentServer/server/node"
	"github.com/foomo/ContentServer/server/repo"
	//"log"
	"net/http"
	"strings"
)

var i int = 0
var contentRepo *repo.Repo

func toJson(obj interface{}) (s string) {
	b, err := json.MarshalIndent(obj, "", "  ")
	if err != nil {
		s = ""
		return
	}
	s = string(b)
	return
}

func jsonResponse(w http.ResponseWriter, obj interface{}) {
	fmt.Fprint(w, toJson(obj))
}

func contentHandler(w http.ResponseWriter, r *http.Request) {
	/*
		i++
		log.Println("request #", i, r)
		childNode := node.NewNode("/foo", map[string]string{"en": "foo"})
		parentNode := node.NewNode("/", map[string]string{"en": "root"}).AddNode("foo", childNode)
		jsonResponse(w, parentNode)
	*/
	uriParts := strings.Split(r.RequestURI, "/")
	jsonResponse(w, contentRepo.GetContent(uriParts[2]))
}

func wtfHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "tank you for your request, but i am totally lost with it\n", r.RequestURI, "\n")
}

func update() interface{} {
	contentRepo.Update()
	/*
		response, err := http.Get("http://test.bestbytes/foomo/modules/Foomo.Page.Content/services/content.php")
		if err != nil {
			fmt.Printf("%s", err)
			return "aua"
		} else {
			defer response.Body.Close()
			contents, err := ioutil.ReadAll(response.Body)
			if err != nil {
				fmt.Printf("%s", err)
			}
			fmt.Printf("json string %s", string(contents))
			jsonNode := node.NewNode("/foo", map[string]string{"en": "foo"})
			jsonErr := json.Unmarshal(contents, &jsonNode)
			if jsonErr != nil {
				fmt.Println("wtf")
			}
			//fmt.Printf("obj %v", jsonNode)
			jsonNode.PrintNode("root", 0)
			return jsonNode
		}
	*/
	return contentRepo.Directory
}

func commandHandler(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.RequestURI, "/")
	switch true {
	case true == (len(parts) > 1):
		switch parts[2] {
		case "update":
			jsonResponse(w, update())
			return
		default:
			help := make(map[string]interface{})
			help["input"] = parts[2]
			help["commands"] = []string{"update", "help"}
			jsonResponse(w, help)
		}
	default:
		wtfHandler(w, r)
	}
}

func Run() {
	contentRepo = repo.NewRepo("http://test.bestbytes/foomo/modules/Foomo.Page.Content/services/content.php")
	http.HandleFunc("/content/", contentHandler)
	http.HandleFunc("/cmd/", commandHandler)
	http.HandleFunc("/", wtfHandler)
	http.ListenAndServe(":8080", nil)
}
