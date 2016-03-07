package content

// Node a node in a content tree
type Node struct {
	Item  *Item            `json:"item"`
	Nodes map[string]*Node `json:"nodes"`
	Index []string         `json:"index"`
}

// NewNode constructor
func NewNode() *Node {
	return &Node{
		Item:  NewItem(),
		Nodes: map[string]*Node{},
	}
}
