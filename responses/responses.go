package responses

import "fmt"

// Error describes an error for humans and machines
type Error struct {
	Status  int    `json:"status"`
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (e Error) Error() string {
	return fmt.Sprintf("status:%d, code:%d, message:%q", e.Status, e.Code, e.Message)
}

// NewError - a brand new error using fmt.Sprintf
func NewErrorf(code int, message string, args ...interface{}) *Error {
	return &Error{
		Status:  500,
		Code:    code,
		Message: fmt.Sprintf(message, args...),
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
