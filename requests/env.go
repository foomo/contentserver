package requests

// Env - abstract your server state
type Env struct {
	// when resolving conten these are processed in their order
	Dimensions []string `json:"dimensions"`
	// who is it for
	Groups []string `json:"groups"`
}
