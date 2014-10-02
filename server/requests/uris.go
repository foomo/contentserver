package requests

type URIs struct {
	Ids       []string `json:"ids"`
	Dimension string   `json:"dimension"`
}

func NewURIs() *URIs {
	return new(URIs)
}
