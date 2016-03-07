package content

// Status status type SiteContent respnses
type Status int

const (
	// StatusOk we found content
	StatusOk Status = 200
	// StatusForbidden we found content but you mst not access it
	StatusForbidden = 403
	// StatusNotFound we did not find content
	StatusNotFound = 404
)

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
