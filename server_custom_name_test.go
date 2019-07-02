package rabbit

import (
	"testing"
	"time"
)

// NameService ...
type NameService int

// Add method
func (a *NameService) Add(data AddPayload) (int, *ApplicationError) {
	return data.X + data.Y, nil
}

// Subtract method
func (*NameService) Subtract(data AddPayload) (int, *ApplicationError) {
	return data.X - data.Y, nil
}

// StringReturn method
func (*NameService) StringReturn(data struct{ Message string }) (string, *ApplicationError) {
	return data.Message + " world", nil
}

// ErrorReturn ...
func (*NameService) ErrorReturn(_ interface{}) (interface{}, *ApplicationError) {
	return nil, &ApplicationError{"500", "Message", nil}
}

// StructReturn ...
func (*NameService) StructReturn(_ interface{}) (interface{}, *ApplicationError) {
	return struct {
		Message string `json:"message"`
	}{Message: "hello world"}, nil
}

// ArrayReturn method
func (*NameService) ArrayReturn(_ interface{}) (interface{}, *ApplicationError) {
	return []struct {
		Message string `json:"message"`
	}{{Message: "hello world from array"}}, nil
}

func startNewNameServer(t *testing.T) {
	newServer, err := CreateServer(defaultURL, "ServiceName")
	if err != nil {
		failRabbitMQConnect(t, err)
	}

	newServer.RegisterName("Command", new(NameService))
	go newServer.Serve()
}

// TestRPC ...
func TestNameServerRPC(t *testing.T) {
	startNewNameServer(t)
	t.Log("Server started")
	client, err := CreateClient(&CreateClientOption{URL: defaultURL, Queue: "ServiceName", TimeoutRequest: 5 * time.Second})
	if err != nil {
		failRabbitMQConnect(t, err)
	}

	var intResult int

	err = client.Send(
		request{
			Action: "Command.Add",
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
			Action: "Command.Subtract",
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
			Action: "Command.StringReturn",
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
			Action: "Command.ErrorReturn",
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
			Action: "Command.ArrayReturn",
			Data: struct {
			}{}},
		&arrMessage,
	)

	if arrMessage[0].Message != "hello world from array" {
		t.Fatalf("expect result to equal to 'hello world from array' but got %s", arrMessage[0].Message)
	}
}
