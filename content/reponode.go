package content

import (
	"fmt"
	"strings"
)

// RepoNode node in a content tree
type RepoNode struct {
	ID            string                 `json:"id"`       // unique identifier - it is your responsibility, that they are unique
	MimeType      string                 `json:"mimeType"` // well a mime type http://www.ietf.org/rfc/rfc2046.txt
	LinkID        string                 `json:"linkId"`   // (symbolic) link/alias to another node
	Groups        []string               `json:"groups"`   // which groups have access to the node, if empty everybody has access to it
	URI           string                 `json:"URI"`
	Name          string                 `json:"name"`
	Hidden        bool                   `json:"hidden"`        // hidden in content.nodes, but can still be resolved when being directly addressed
	DestinationID string                 `json:"destinationId"` // if a node does not have any content like a folder the destinationIds can point to nodes that do aka. the first displayable child node
	Data          map[string]interface{} `json:"data"`          // what ever you want to stuff into it - the payload you want to attach to a node
	Nodes         map[string]*RepoNode   `json:"nodes"`         // child nodes
	Index         []string               `json:"index"`         // defines the order of the child nodes
	parent        *RepoNode              // parent node - helps to resolve a path / bread crumb
	// published from - to is going to be an array of fromTos
}

// // NewRepoNode constructor
// func NewRepoNode() *RepoNode {
// 	return &RepoNode{
// 		Data:  make(map[string]interface{}, 0), // set initial size to zero explicitly?
// 		Nodes: make(map[string]*RepoNode, 0),
// 	}
// }

// WireParents helper method to reference from child to parent in a tree
// recursively
func (n *RepoNode) WireParents() {
	for _, childNode := range n.Nodes {
		childNode.parent = n
		childNode.WireParents()
	}
}

// InPath is the given node in a path
func (n *RepoNode) InPath(path []*Item) bool {
	myParentID := n.parent.ID
	for _, pathItem := range path {
		if pathItem.ID == myParentID {
			return true
		}
	}
	return false
}

// GetPath get a path for a repo node
func (n *RepoNode) GetPath(dataFields []string) []*Item {
	var (
		parentNode = n.parent
		pathLength = 0
	)
	for parentNode != nil {
		parentNode = parentNode.parent
		pathLength++
	}
	parentNode = n.parent

	var (
		i    = 0
		path = make([]*Item, pathLength)
	)

	if dataFields == nil {
		dataFields = []string{}
	}

	for parentNode != nil {
		path[i] = parentNode.ToItem(dataFields)
		parentNode = parentNode.parent
		i++
	}
	return path
}

// ToItem convert a repo node to a simple repo item
func (n *RepoNode) ToItem(dataFields []string) *Item {
	item := NewItem()
	item.ID = n.ID
	item.Name = n.Name
	item.MimeType = n.MimeType
	item.Hidden = n.Hidden
	item.URI = n.URI
	item.Groups = n.Groups
	if dataFields == nil {
		item.Data = n.Data
	} else {
		for _, dataField := range dataFields {
			if data, ok := n.Data[dataField]; ok {
				item.Data[dataField] = data
			}
		}
	}
	return item
}

// GetParent get the parent node of a node
func (n *RepoNode) GetParent() *RepoNode {
	return n.parent
}

// AddNode adds a named child node
func (n *RepoNode) AddNode(name string, childNode *RepoNode) *RepoNode {
	n.Nodes[name] = childNode
	return n
}

// IsOneOfTheseMimeTypes is the node one of the given mime types
func (n *RepoNode) IsOneOfTheseMimeTypes(mimeTypes []string) bool {
	if len(mimeTypes) == 0 {
		return true
	}
	for _, mimeType := range mimeTypes {
		if mimeType == n.MimeType {
			return true
		}
	}
	return false
}

// CanBeAccessedByGroups can this node be accessed by at least one the given
// groups
func (n *RepoNode) CanBeAccessedByGroups(groups []string) bool {
	// no groups set on node => anybody can access it
	if len(n.Groups) == 0 {
		return true
	}

	for _, group := range groups {
		for _, myGroup := range n.Groups {
			if group == myGroup {
				return true
			}
		}
	}
	return false
}

// PrintNode essentially a recursive dump
func (n *RepoNode) PrintNode(id string, level int) {
	prefix := strings.Repeat(Indent, level)
	fmt.Printf("%s %s %s:\n", prefix, id, n.Name)
	for key, childNode := range n.Nodes {
		childNode.PrintNode(key, level+1)
	}
}
