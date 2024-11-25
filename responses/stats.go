package responses

type Stats struct {
	NumberOfNodes int `json:"numberOfNodes"`
	NumberOfURIs  int `json:"numberOfURIs"`
	// seconds
	RepoRuntime float64 `json:"repoRuntime"`
	// seconds
	OwnRuntime float64 `json:"ownRuntime"`
}
