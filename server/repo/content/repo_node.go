package content

import (
	"fmt"
	"strings"
)

type RepoNode struct {
	Id            string                 `json:"id"`       // unique identifier - it is your responsibility, that they are unique
	MimeType      string                 `json:"mimeType"` // well a mime type http://www.ietf.org/rfc/rfc2046.txt
	LinkId        string                 `json:"linkId"`   // (symbolic) link/alias to another node
	Groups        []string               `json:"groups"`   // which groups have access to the node, if empty everybody has access to it
	URI           string                 `json:"URI"`
	Name          string                 `json:"name"`
	Hidden        bool                   `json:"hidden"`        // hidden in content.nodes, but can still be resolved when being directly addressed
	DestinationId string                 `json:"destinationId"` // if a node does not have any content like a folder the destinationIds can point to nodes that do aka. the first displayable child node
	Data          map[string]interface{} `json:"data"`          // what ever you want to stuff into it - the payload you want to attach to a node
	Nodes         map[string]*RepoNode   `json:"nodes"`         // child nodes
	Index         []string               `json:"index"`         // defines the order of the child nodes
	parent        *RepoNode              // parent node - helps to resolve a path / bread crumb
	// published from - to is going to be an array of fromTos
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

func (node *RepoNode) GetPath() []*Item {
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
		path[i] = parentNode.ToItem([]string{})
		parentNode = parentNode.parent
		i++
	}
	return path
}

func (node *RepoNode) ToItem(dataFields []string) *Item {
	item := NewItem()
	item.Id = node.Id
	item.Name = node.Name
	item.MimeType = node.MimeType
	item.URI = node.URI
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
	if len(groups) == 0 || len(node.Groups) == 0 {
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
	fmt.Printf("%s %s %s:\n", prefix, id, node.Name)
	for key, childNode := range node.Nodes {
		childNode.PrintNode(key, level+1)
	}
}

func NewRepoNode() *RepoNode {
	node := new(RepoNode)
	return node
}
