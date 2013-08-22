package content

import (
	"fmt"
	"strings"
)

type RepoNode struct {
	Id       string `json:"id"`
	MimeType string `json:"mimeType"`
	Handler  string `json:"handler"`
	//Path     string                       `json:"path"`
	Regions map[string]string            `json:"regions"`
	URIs    map[string]map[string]string `json:"uris"`
	Names   map[string]string            `json:"names"`
	Hidden  bool                         `json:"hidden"` // hidden in tree
	Groups  []string                     `json:"groups"`
	Data    map[string]interface{}       `json:"data"`
	Index   []string                     `json:"index"`
	Nodes   map[string]*RepoNode         `json:"nodes"`
	LinkIds []string                     `json:"linkIds"` // ids to link to
	// published from - to
}

func (node *RepoNode) AddNode(name string, childNode *RepoNode) *RepoNode {
	node.Nodes[name] = childNode
	return node
}

func (node *RepoNode) GetName(language string) string {
	return node.Names[language]
}

func (node *RepoNode) PrintNode(id string, level int) {
	prefix := strings.Repeat(INDENT, level)
	fmt.Printf("%s %s:\n", prefix, id)
	for lang, name := range node.Names {
		fmt.Printf("%s %s: %s\n", prefix+INDENT, lang, name)
	}
	for key, childNode := range node.Nodes {
		childNode.PrintNode(key, level+1)
	}
}

func NewRepoNode() *RepoNode {
	node := new(RepoNode)
	return node
}
