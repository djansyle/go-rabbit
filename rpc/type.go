package rpc

import "encoding/json"

// Request holds the information of the request
type Request struct {
	Action string          `json:"type"`
	Data   json.RawMessage `json:"data"`
}