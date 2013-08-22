package repo

import (
	"encoding/json"
	"fmt"
	"github.com/foomo/ContentServer/server/node"
	"io/ioutil"
	"net/http"
)

type SiteContent struct {
	Path string `json: path`
}

func NewSiteContent() *SiteContent {
	content := new(SiteContent)
	return content
}

type Repo struct {
	server    string
	Directory map[string]node.Node
}

func NewRepo(server string) *Repo {
	repo := new(Repo)
	repo.server = server
	return repo
}

func (repo *Repo) GetContent(path string) *SiteContent {
	content := NewSiteContent()
	content.Path = path
	return content
}

func (repo *Repo) builDirectory(dirNode *node.Node) {
	repo.Directory[dirNode.Path] = *dirNode
	for _, childNode := range dirNode.Nodes {
		repo.builDirectory(childNode)
	}
}

func (repo *Repo) Update() interface{} {
	response, err := http.Get(repo.server)
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

		repo.Directory = make(map[string]node.Node)
		repo.builDirectory(jsonNode)
		return jsonNode
	}
}
