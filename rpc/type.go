package rpc

import "encoding/json"

// Request holds the information of the request
type Request struct {
	Action string          `json:"type"`
	Data   json.RawMessage `json:"data"`
}

type Response struct {
	Result json.RawMessage `json:"result"`
	Error  json.RawMessage `json:"error"`
}
