package content

// SiteContent resolved content for a site
type SiteContent struct {
	Status    Status            `json:"status"`
	URI       string            `json:"URI"`
	Dimension string            `json:"dimension"`
	MimeType  string            `json:"mimeType"`
	Item      *Item             `json:"item"`
	Data      interface{}       `json:"data"`
	Path      []*Item           `json:"path"`
	URIs      map[string]string `json:"URIs"`
	Nodes     map[string]*Node  `json:"nodes"`
}

// NewSiteContent constructor
func NewSiteContent() *SiteContent {
	return &SiteContent{
		Nodes: make(map[string]*Node),
		URIs:  make(map[string]string),
	}
}
