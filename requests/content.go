package requests

// Content - the standard request to contentserver
type Content struct {
	Env            *Env             `json:"env"`
	URI            string           `json:"URI"`
	Nodes          map[string]*Node `json:"nodes"`
	DataFields     []string         `json:"dataFields"`
	PathDataFields []string         `json:"pathDataFields"`
}
