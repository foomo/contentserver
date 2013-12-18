package content

import ()

type Item struct {
	Id   string                 `json:"id"`
	Name string                 `json:"name"`
	URI  string                 `json:"URI"`
	Data map[string]interface{} `json:"data"`
}

func NewItem() *Item {
	item := new(Item)
	item.Data = make(map[string]interface{})
	return item
}
