package rabbit

import (
	"errors"
)

// TimeOutError is return when the request timed out.
var TimeOutError = errors.New("request timeout")

type ApplicationError struct {
	Code string `json:"code"`
	Message string `json:"message"`
	Meta interface{} `json:"meta"`
}

func (e *ApplicationError) Error() string {
	return e.Message
}