package requests

type URIs struct {
	Ids      []string `json:"ids"`
	Region   string   `json:"region"`
	Language string   `json:"language"`
}

func NewURIs() *URIs {
	return new(URIs)
}
