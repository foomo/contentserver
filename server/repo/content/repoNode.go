package content

import (
	"fmt"
	"strings"

//	"github.com/foomo/ContentServer/server/repo"
)

type RepoNode struct {
	Id            string                       `json:"id"`
	MimeType      string                       `json:"mimeType"`
	Handler       string                       `json:"handler"`
	Regions       []string                     `json:"regions"`
	URIs          map[string]map[string]string `json:"URIs"`
	DestinationId string                       `json:"destinationId"`
	Names         map[string]map[string]string `json:"names"`
	Hidden        map[string]map[string]bool   `json:"hidden"` // hidden in tree
	Groups        []string                     `json:"groups"`
	Data          map[string]interface{}       `json:"data"`
	Content       map[string]interface{}       `json:"content"`
	Nodes         map[string]*RepoNode         `json:"nodes"`
	Index         []string                     `json:"index"`
	LinkId        map[string]map[string]string `json:"linkIds"` // ids to link to
	parent        *RepoNode
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

func (node *RepoNode) InPath(path []*Item) bool {
	myParentId := node.parent.Id
	for _, pathItem := range path {
		if pathItem.Id == myParentId {
			return true
		}
	}
	return false
}

func (node *RepoNode) InRegion(region string) bool {
	for _, nodeRegion := range node.Regions {
		if nodeRegion == region {
			return true
		}
	}
	return false
}

func (node *RepoNode) GetPath(region string, language string) []*Item {
	parentNode := node.parent
	pathLength := 0
	for parentNode != nil {
		parentNode = parentNode.parent
		pathLength++
	}
	parentNode = node.parent
	i := 0
	path := make([]*Item, pathLength)
	for parentNode != nil {
		path[i] = parentNode.ToItem(region, language)
		parentNode = parentNode.parent
		i++
	}
	return path
}

func (node *RepoNode) ToItem(region string, language string) *Item {
	item := NewItem()
	item.Id = node.Id
	item.Name = node.GetName(region, language)
	item.URI = node.URIs[region][language] //uri //repo.GetURI(region, language, node.Id)
	return item
}

func (node *RepoNode) GetParent() *RepoNode {
	return node.parent
}

func (node *RepoNode) AddNode(name string, childNode *RepoNode) *RepoNode {
	node.Nodes[name] = childNode
	return node
}

func (node *RepoNode) IsHidden(region string, language string) bool {
	if regionMap, ok := node.Hidden[region]; ok {
		if languageHidden, ok := regionMap[language]; ok {
			return languageHidden
		} else {
			return false
		}
	}
	return false
}

func (node *RepoNode) GetName(region string, language string) string {
	return node.Names[region][language]
}

func (node *RepoNode) IsOneOfTheseMimeTypes(mimeTypes []string) bool {
	if len(mimeTypes) == 0 {
		return true
	} else {
		for _, mimeType := range mimeTypes {
			if mimeType == node.MimeType {
				return true
			}
		}
		return false
	}
}

func (node *RepoNode) CanBeAccessedByGroups(groups []string) bool {
	if len(groups) == 0 {
		return true
	} else {
		// @todo is there sth like in_array ... or some array intersection
		for _, group := range groups {
			for _, myGroup := range node.Groups {
				if group == myGroup {
					return true
				}
			}
		}
		return false
	}
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
