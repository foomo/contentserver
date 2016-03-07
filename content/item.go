package content

// Item on a node in a content tree - "payload" of an item
type Item struct {
	ID       string                 `json:"id"`
	Name     string                 `json:"name"`
	URI      string                 `json:"URI"`
	MimeType string                 `json:"mimeType"`
	Data     map[string]interface{} `json:"data"`
}

// NewItem item contructor
func NewItem() *Item {
	item := new(Item)
	item.Data = make(map[string]interface{})
	return item
}
