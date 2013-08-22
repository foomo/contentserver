package content

import ()

type Item struct {
	Id   string `json:"id"`
	Name string `json:"name"`
	URI  string `json:"URI"`
}

func NewItem() *Item {
	return new(Item)
}
