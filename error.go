package rabbit

import (
	"errors"
)

// ErrTimeout is return when the request timed out.
var ErrTimeout = errors.New("request timeout")

// ApplicationError is return when request has a validation error
type ApplicationError struct {
	Code    string      `json:"code"`
	Message string      `json:"message"`
	Meta    interface{} `json:"meta"`
}

func (e *ApplicationError) Error() string {
	return e.Message
}
