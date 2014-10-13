package requests

type Content struct {
	Env   *Env             `json:"env"`
	URI   string           `json:"URI"`
	Nodes map[string]*Node `json:"nodes"`
}

func NewContent() *Content {
	return new(Content)
}
