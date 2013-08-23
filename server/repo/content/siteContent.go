package content

const (
	STATUS_OK        = 200
	STATUS_FORBIDDEN = 403
	STATUS_NOT_FOUND = 404
)

type SiteContent struct {
	Status   int    `json:"status"`
	Region   string `json:"region"`
	Language string `json:"language"`
	Content  struct {
		Item *Item       `json:"item"`
		Data interface{} `json:"data"`
	} `json:"content"`
	URIs  map[string]string `json:"URIs"`
	Nodes map[string]*Node  `json:"nodes"`
}

func NewSiteContent() *SiteContent {
	c := new(SiteContent)
	c.Nodes = make(map[string]*Node)
	c.URIs = make(map[string]string)
	return c
}
