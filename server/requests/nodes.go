package requests

type Nodes struct {
	Nodes map[string]*Node `json:"nodes"`
	Env   *Env             `json:"env"`
}

func NewNodes() *Nodes {
	return new(Nodes)
}
