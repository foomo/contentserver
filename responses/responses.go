package responses

import "fmt"

// Error describes an error for humans and machines
type Error struct {
	Status  int    `json:"status"`
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (e Error) Error() string {
	return fmt.Sprintf("status:%q, code: %q, message: %q", e.Status, e.Code, e.Message)
}

// NewError - a brand new error
func NewError(code int, message string) *Error {
	return &Error{
		Status:  500,
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
