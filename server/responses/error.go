package responses

type Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func NewError(code int, message string) *Error {
	error := new(Error)
	error.Code = code
	error.Message = message
	return error
}
