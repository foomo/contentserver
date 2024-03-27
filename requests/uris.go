package requests

// URIs - request multiple URIs for a dimension use this resolve uris for links
// in a document
type URIs struct {
	IDs       []string `json:"ids"`
	Dimension string   `json:"dimension"`
}
