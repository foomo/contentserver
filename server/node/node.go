package node

import (
	"fmt"
	"strings"
)

type Node struct {
	Id    string            `json:"id"`
	Path  string            `json:"path"`
	Names map[string]string `json:"names"`
	Nodes map[string]*Node  `json:"nodes"`
	Index []string          `json:"index"`
}

const (
	INDENT string = "\t"
)

func (node *Node) AddNode(name string, childNode *Node) (me *Node) {
	node.Nodes[name] = childNode
	return node
}

func (node *Node) PrintNode(id string, level int) {
	prefix := strings.Repeat(INDENT, level)
	fmt.Printf("%s %s:\n", prefix, id)
	for lang, name := range node.Names {
		fmt.Printf("%s %s: %s\n", prefix+INDENT, lang, name)
	}
	for key, childNode := range node.Nodes {
		childNode.PrintNode(key, level+1)
	}
}

func NewNode(id string, names map[string]string) *Node {
	node := new(Node)
	node.Id = id
	node.Names = names
	return node
}
