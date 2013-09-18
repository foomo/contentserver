package requests

type ItemMap struct {
	Id string `json:"id"`
}

func NewItemMap() *ItemMap {
	return new(ItemMap)
}
