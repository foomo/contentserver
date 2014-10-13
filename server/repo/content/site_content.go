package content

const (
	STATUS_OK        = 200
	STATUS_FORBIDDEN = 403
	STATUS_NOT_FOUND = 404
)

type SiteContent struct {
	Status    int               `json:"status"`
	URI       string            `json:"URI"`
	Dimension string            `json:"dimension"`
	MimeType  string            `json:"mimeType"`
	Item      *Item             `json:"item"`
	Data      interface{}       `json:"data"`
	Path      []*Item           `json:"path"`
	URIs      map[string]string `json:"URIs"`
	Nodes     map[string]*Node  `json:"nodes"`
}

func NewSiteContent() *SiteContent {
	c := new(SiteContent)
	c.Nodes = make(map[string]*Node)
	c.URIs = make(map[string]string)
	return c
}
