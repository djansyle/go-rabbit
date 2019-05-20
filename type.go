package rabbit

import "encoding/json"

// Request holds the information of the request
type Request struct {
	Action string          `json:"type"`
	Data   json.RawMessage `json:"data"`
}

// Response holds the information of the response of the service
type Response struct {
	Result json.RawMessage `json:"result"`
	Error  json.RawMessage `json:"error"` // present when there is a problem processing the request
}

// Service alias for all service type
type Service int

var (
	defaultURL = "amqp://guest:guest@localhost:5672/"
)
