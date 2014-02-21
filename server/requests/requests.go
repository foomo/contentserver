package requests

type Env struct {
	Defaults struct {
		Region   string `json:"region"`
		Language string `json:"language"`
	} `json:"defaults"`
	Groups []string    `json:"groups"`
	State string
	Data   interface{} `json:"data"`
}

type Node struct {
	Id         string   `json:"id"`
	MimeTypes  []string `json:"mimeTypes"`
	Expand     bool     `json:"expand"`
	DataFields []string `json:"dataFields"`
}
