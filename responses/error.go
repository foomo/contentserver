package responses

import (
	"fmt"
)

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
