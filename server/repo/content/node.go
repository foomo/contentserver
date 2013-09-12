package content

import ()

type Node struct {
	Item  *Item            `json:"item"`
	Nodes map[string]*Node `json:"nodes"`
	Index []string         `json:"index"`
}

func NewNode() *Node {
	node := new(Node)
	node.Item = NewItem()
	node.Nodes = make(map[string]*Node)
	return node
}
