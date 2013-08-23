package content

import (
	"fmt"
	"strings"
)

type RepoNode struct {
	Id       string                       `json:"id"`
	MimeType string                       `json:"mimeType"`
	Handler  string                       `json:"handler"`
	Regions  []string                     `json:"regions"`
	URIs     map[string]map[string]string `json:"URIs"`
	Names    map[string]string            `json:"names"`
	Hidden   bool                         `json:"hidden"` // hidden in tree
	Groups   []string                     `json:"groups"`
	Data     map[string]interface{}       `json:"data"`
	Content  map[string]interface{}       `json:"content"`
	Nodes    map[string]*RepoNode         `json:"nodes"`
	LinkIds  map[string]map[string]string `json:"linkIds"` // ids to link to
	parent   *RepoNode
	// published from - to
}

func (node *RepoNode) GetLanguageAndRegionForURI(URI string) (resolved bool, region string, language string) {
	for possibleRegion, URIs := range node.URIs {
		for possibleLanguage, regionLangURI := range URIs {
			if regionLangURI == URI {
				resolved = true
				region = possibleRegion
				language = possibleLanguage
				return
			}
		}
	}
	resolved = false
	return
}

func (node *RepoNode) WireParents() {
	for _, childNode := range node.Nodes {
		childNode.parent = node
		childNode.WireParents()
	}
}

func (node *RepoNode) GetPath(region string, language string) map[string]*Item {
	path := make(map[string]*Item)
	parentNode := node.parent
	for parentNode != nil {
		path[parentNode.Id] = parentNode.ToItem(region, language)
		parentNode = parentNode.parent
	}
	return path
}

func (node *RepoNode) ToItem(region string, language string) *Item {
	item := NewItem()
	item.Id = node.Id
	item.Name = node.GetName(language)
	item.URI = node.URIs[region][language]
	return item
}

func (node *RepoNode) GetParent() *RepoNode {
	return node.parent
}

func (node *RepoNode) AddNode(name string, childNode *RepoNode) *RepoNode {
	node.Nodes[name] = childNode
	return node
}

func (node *RepoNode) GetName(language string) string {
	return node.Names[language]
}

func (node *RepoNode) IsOneOfTheseMimeTypes(mimeTypes []string) bool {
	for _, mimeType := range mimeTypes {
		if mimeType == node.MimeType {
			return true
		}
	}
	return false
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
