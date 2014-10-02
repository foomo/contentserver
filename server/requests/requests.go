package requests

type Env struct {
	Dimensions []string    `json:"dimensions"`
	Groups     []string    `json:"groups"`
	Data       interface{} `json:"data"`
}

type Node struct {
	Id         string   `json:"id"`
	Dimension  string   `json:"id"`
	MimeTypes  []string `json:"mimeTypes"`
	Expand     bool     `json:"expand"`
	DataFields []string `json:"dataFields"`
}
