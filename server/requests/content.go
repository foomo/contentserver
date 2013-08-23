package requests

type Content struct {
	Env struct {
		Groups []string    `json:"groups"`
		Data   interface{} `json:"data"`
	} `json:"env"`
	URI   string
	Nodes map[string]struct {
		Id        string   `json:"id"`
		MimeTypes []string `json:"mimeTypes"`
		Expand    bool     `json:"expand"`
	} `json:"nodes"`
}

func NewContent() *Content {
	return new(Content)
}
