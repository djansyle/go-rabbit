package rabbit

import (
	"encoding/json"
	"testing"
	"time"
)

// TestService ...
type TestService int

// AddPayload ...
type AddPayload struct {
	X int `json:"x"`
	Y int `json:"y"`
}

// Add method
func (a *TestService) Add(data AddPayload) (int, *ApplicationError) {
	return data.X + data.Y, nil
}

// Subtract method
func (*TestService) Subtract(data AddPayload) (int, *ApplicationError) {
	return data.X - data.Y, nil
}

// StringReturn method
func (*TestService) StringReturn(data struct{ Message string }) (string, *ApplicationError) {
	return data.Message + " world", nil
}

// ErrorReturn ...
func (*TestService) ErrorReturn(_ interface{}) (interface{}, *ApplicationError) {
	return nil, &ApplicationError{"500", "Message", nil}
}

// StructReturn ...
func (*TestService) StructReturn(_ interface{}) (interface{}, *ApplicationError) {
	return struct {
		Message string `json:"message"`
	}{Message: "hello world"}, nil
}

// ArrayReturn method
func (*TestService) ArrayReturn(_ interface{}) (interface{}, *ApplicationError) {
	return []struct {
		Message string `json:"message"`
	}{{Message: "hello world from array"}}, nil
}

func failRabbitMQConnect(t *testing.T, err error) {
	t.Fatalf("Error connecting to RabbitMQ instance. Err = %v", err)
}

func defaultRequestParser(body []byte) (request *Request, err error) {
	request = new(Request)

	if err := json.Unmarshal(body, request); err != nil {
		return nil, err
	}

	return request, nil
}

func startNewServer(t *testing.T) {
	newServer, err := CreateServer(defaultURL, "Service", defaultRequestParser)
	if err != nil {
		failRabbitMQConnect(t, err)
	}

	newServer.Register(new(TestService))
	go newServer.Serve()
}

type request struct {
	Action string `json:"type"`
	Data   interface{}
}

// TestRPC ...
func TestRPC(t *testing.T) {
	startNewServer(t)
	t.Log("Server started")
	client, err := CreateClient(
		&CreateClientOption{URL: defaultURL, Queue: "Service", TimeoutRequest: 5 * time.Second},
		defaultRequestFormatter,
	)
	if err != nil {
		failRabbitMQConnect(t, err)
	}

	var intResult int

	err = client.Send(
		request{
			Action: "TestService.Add",
			Data: struct {
				X int `json:"x"`
				Y int `json:"y"`
			}{X: 1, Y: 2}},
		&intResult)

	if err != nil {
		t.Fatalf("expected no error but got %v", err)
	}

	if intResult != 3 {
		t.Fatalf("expect result to 3 but got %d", intResult)
	}

	// Subtract
	err = client.Send(
		request{
			Action: "TestService.Subtract",
			Data: struct {
				X int `json:"x"`
				Y int `json:"y"`
			}{X: 1, Y: 2}},
		&intResult,
	)

	if err != nil {
		t.Fatalf("expected no error but got %q", err)
	}

	if intResult != -1 {
		t.Fatalf("expect result to 3 but got %d", intResult)
	}

	// StringReturn
	var stringResult string
	err = client.Send(
		request{
			Action: "TestService.StringReturn",
			Data: struct {
				Message string `json:"message"`
			}{Message: "hello"}},
		&stringResult,
	)

	if err != nil {
		t.Fatalf("expected no error but got %v", err)
	}

	if stringResult != "hello world" {
		t.Fatalf("expect result to equal to 'hello world' but got %q", stringResult)
	}

	// ErrorReturn
	var nilInterface interface{}
	err = client.Send(
		request{
			Action: "TestService.ErrorReturn",
			Data: struct {
			}{}},
		&nilInterface,
	)

	if err != nil {
		// should be able to cast to ApplicationError
		ae := err.(*ApplicationError)
		if ae == nil {
			t.Fatalf("expected error can be cast to *ApplicationError")
		}

		if ae.Message != "Message" {
			t.Fatalf("expected 'Message' property to equal to 'Message'")
		}

		if ae.Code != "500" {
			t.Fatalf("expected 'Code' property to equal to '500'")
		}

		if ae.Meta != nil {
			t.Fatalf("expect meta field to be nil")
		}
	}

	if nilInterface != nil {
		t.Fatalf("expect result to <nil> but got %v", nilInterface)
	}

	var arrMessage []struct {
		Message string `json:"message"`
	}

	err = client.Send(
		request{
			Action: "TestService.ArrayReturn",
			Data: struct {
			}{}},
		&arrMessage,
	)

	if arrMessage[0].Message != "hello world from array" {
		t.Fatalf("expect result to equal to 'hello world from array' but got %s", arrMessage[0].Message)
	}
}
