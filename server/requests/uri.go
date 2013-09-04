package requests

type URI struct {
	Id       string `json:"id"`
	Region   string `json:"region"`
	Language string `json:"language"`
}

func NewURI() *URI {
	return new(URI)
}
