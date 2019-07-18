package rabbit

import (
	"testing"
	"time"
)

func defaultRequestFormatter(request interface{}) (interface{}, error) {
	return request, nil
}

func TestCreateClient(t *testing.T) {
	c, err := CreateClient(
		&CreateClientOption{URL: "amqp://guest:guest@localhost:5672/", Queue: "go_test", TimeoutRequest: 5 * time.Second},
		defaultRequestFormatter,
	)

	if err != nil {
		t.Fatalf("Failed to connect reason: %v", err)
	}

	if c == nil {
		t.Fatalf("Client has not been initialized")
	}

}
