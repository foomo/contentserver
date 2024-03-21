package requests

// Nodes - which nodes in which dimensions
type Nodes struct {
	// map[dimension]*node
	Nodes map[string]*Node `json:"nodes"`
	Env   *Env             `json:"env"`
}
