package content

// Item on a node in a content tree - "payload" of an item
type Item struct {
	ID       string                 `json:"id"`
	Name     string                 `json:"name"`
	URI      string                 `json:"URI"`
	MimeType string                 `json:"mimeType"`
	Hidden   bool                   `json:"hidden,omitempty"`
	Data     map[string]interface{} `json:"data"`
	Groups   []string               `json:"groups"`
}

// NewItem item contructor
func NewItem() *Item {
	return &Item{
		Data: map[string]interface{}{},
	}
}
