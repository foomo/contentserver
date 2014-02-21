package content

import (
	"fmt"
	"strings"
)

type RepoNode struct {
	Id             string                       `json:"id"`       // unique identifier - it is your responsibility, that they are unique
	MimeType       string                       `json:"mimeType"` // well a mime type http://www.ietf.org/rfc/rfc2046.txt
	LinkId         string                       `json:"linkId"`   // (symbolic) link/alias to another node
	Handler        string                       `json:"handler"`  // that information is for you
	Regions        []string                     `json:"regions"`  // in what regions is this node available, if empty it will be accessible everywhere
	Groups         []string                     `json:"groups"`   // which groups have access to the node, if empty everybody has access to it
	States         []string                     `json:"states"`   // in which states is this node valid, if empty => in all of them
	URIs           map[string]map[string]string `json:"URIs"`
	Names          map[string]map[string]string `json:"names"`
	Hidden         map[string]map[string]bool   `json:"hidden"`         // hidden in content.nodes, but can still be resolved when being directly addressed
	DestinationIds map[string]map[string]string `json:"destinationIds"` // if a node does not have any content like a folder the destinationIds can point to nodes that do aka. the first displayable child node
	Data           map[string]interface{}       `json:"data"`           // what ever you want to stuff into it - the payload you want to attach to a node
	Nodes          map[string]*RepoNode         `json:"nodes"`          // child nodes
	Index          []string                     `json:"index"`          // defines the order of the child nodes
	parent         *RepoNode                    // parent node - helps to resolve a path / bread crumb
	// published from - to is going to be an array of fromTos
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

func (node *RepoNode) InState(state string) bool {
	if(len(node.States) == 0) {
		return true
	} else {
		for _, nodeState := range node.States {
			if(state == nodeState) {
				return true
			}
		}
		return false;
	}
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
		path[i] = parentNode.ToItem(region, language, []string{})
		parentNode = parentNode.parent
		i++
	}
	return path
}

func (node *RepoNode) ToItem(region string, language string, dataFields []string) *Item {
	item := NewItem()
	item.Id = node.Id
	item.Name = node.GetName(region, language)
	item.URI = node.URIs[region][language] //uri //repo.GetURI(region, language, node.Id)
	for _, dataField := range dataFields {
		if data, ok := node.Data[dataField]; ok {
			item.Data[dataField] = data
		}
	}
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
