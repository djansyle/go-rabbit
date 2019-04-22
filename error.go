package rabbit

import "errors"

// ErrTimeOut is return when the request timesout.
var ErrTimeOut = errors.New("request timeout")
