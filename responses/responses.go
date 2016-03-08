package responses

// Error describes an error for humans and machines
type Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// NewError - a brand new error
func NewError(code int, message string) *Error {
	return &Error{
		Code:    code,
		Message: message,
	}
}

// Update - information about an update
type Update struct {
	// did it work or not
	Success bool `json:"success"`
	// this is for humand
	ErrorMessage string `json:"errorMessage"`
	Stats        struct {
		NumberOfNodes int `json:"numberOfNodes"`
		NumberOfURIs  int `json:"numberOfURIs"`
		// seconds
		RepoRuntime float64 `json:"repoRuntime"`
		// seconds
		OwnRuntime float64 `json:"ownRuntime"`
	} `json:"stats"`
}
