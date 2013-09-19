package responses

type Update struct {
	Stats struct {
		NumberOfNodes int     `json:"numberOfNodes"`
		NumberOfURIs  int     `json:"numberOfURIs"`
		RepoRuntime   float64 `json:"repoRuntime"`
		OwnRuntime    float64 `json:"ownRuntime"`
	} `json:"stats"`
}

func NewUpdate() *Update {
	return new(Update)
}
