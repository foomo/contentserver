package requests

type Content struct {
	Env   *Env `json:"env"`
	URI   string
	Nodes map[string]*Node `json:"nodes"`
}

func NewContent() *Content {
	return new(Content)
}
