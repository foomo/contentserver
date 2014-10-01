package requests

type ItemMap struct {
	Id         string   `json:"id"`
	DataFields []string `json:"dataFields"`
}

func NewItemMap() *ItemMap {
	return new(ItemMap)
}
