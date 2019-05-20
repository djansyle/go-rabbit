package rabbit

import (
	"testing"
	"time"
)

type TestService int

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

func (*TestService) StringReturn(data struct{ Message string }) (string, *ApplicationError) {
	return data.Message + " world", nil
}

func (*TestService) ErrorReturn(_ interface{}) (interface{}, *ApplicationError) {
	return nil, &ApplicationError{"500", "Message", nil}
}

func (*TestService) StructReturn(_ interface{}) (interface{}, *ApplicationError) {
	return struct {
		Message string `json:"message"`
	}{Message: "hello world"}, nil
}

func failRabbitMQConnect(t *testing.T, err error) {
	t.Fatalf("Error connecting to RabbitMQ instance. Err = %v", err)
}

func startNewServer(t *testing.T) {
	newServer, err := CreateServer(url, "Service")
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

func TestRPC(t *testing.T) {
	startNewServer(t)
	t.Log("Server started")
	client, err := CreateClient(url, "Service", 5*time.Second)
	if err != nil {
		failRabbitMQConnect(t, err)
	}

	var intResult int

	err = client.Send(
		request{
			Action: "Service.Add",
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
			Action: "Service.Subtract",
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
			Action: "Service.StringReturn",
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
			Action: "Service.ErrorReturn",
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
}
