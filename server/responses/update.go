package responses

type Update struct {
	Success      bool   `json:"success"`
	ErrorMessage string `json:"errorMessage"`
	Stats        struct {
		NumberOfNodes int     `json:"numberOfNodes"`
		NumberOfURIs  int     `json:"numberOfURIs"`
		RepoRuntime   float64 `json:"repoRuntime"`
		OwnRuntime    float64 `json:"ownRuntime"`
	} `json:"stats"`
}

func NewUpdate() *Update {
	return new(Update)
}
