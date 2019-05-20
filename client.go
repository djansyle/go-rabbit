package rabbit

import (
	"encoding/json"
	"fmt"
	"reflect"
	"time"

	"github.com/streadway/amqp"
)

// ErrInvalidOutput error when the service method is not supported
type ErrInvalidOutput struct {
	Type reflect.Type
}

func (e *ErrInvalidOutput) Error() string {
	if e.Type == nil {
		return "rabbit: Send Output(nil)"
	}

	if e.Type.Kind() != reflect.Ptr {
		return "rabbit: Send Output(non-pointer " + e.Type.String() + ")"
	}
	return "rabbit: Send Output(nil " + e.Type.String() + ")"
}

// Sender is the interface that satisfies the sending of messages to the server
type Sender interface {
	Send(interface{}, interface{}) error
}

type client struct {
	requests    map[string]chan []byte
	serverQueue string
	timeout     time.Duration

	connection *Connection
	queue      *amqp.Queue
}

// CreateClient creates a new client for rpc server
func CreateClient(url string, serverQueue string, timeoutRequest time.Duration) (Sender, error) {
	conn, err := CreateConnection(url)
	if err != nil {
		return nil, err
	}

	ch := conn.Channel

	q, err := ch.QueueDeclare(
		"",    // name
		true,  // durable
		false, // auto-delete
		true,  // exclusive
		false, // no-wait
		nil,   // args
	)

	if err != nil {
		return nil, err
	}

	msgs, err := ch.Consume(
		q.Name, // queue
		"",     // consumer
		false,  // auto-ack
		true,   // exclusive
		false,  // no-local
		false,  // no-wait
		nil,    // args
	)

	requests := make(map[string]chan []byte)
	go func() {
		for d := range msgs {
			request := requests[d.CorrelationId]
			fmt.Printf("Message received: %s", string(d.Body))
			if request == nil {
				fmt.Printf("Message unexpectedly arrive with correlation id %s.", d.CorrelationId)
				continue
			}

			request <- d.Body
			delete(requests, d.CorrelationId)
			ack := d.Ack(false)
			if ack != nil {
				fmt.Printf("Failed to ack: %v", err)
			}
		}
	}()

	return &client{requests: requests, serverQueue: serverQueue, connection: conn, timeout: timeoutRequest, queue: &q}, nil
}

// Sends the message to the rpc server
func (c *client) Send(message interface{}, output interface{}) error {
	ro := reflect.ValueOf(output)
	if ro.Kind() != reflect.Ptr || ro.IsNil() {
		return &ErrInvalidOutput{reflect.TypeOf(ro)}
	}

	corrID := RandomID()

	request := make(chan []byte)
	defer close(request)

	c.requests[corrID] = request

	payload, err := json.Marshal(message)
	if err != nil {
		return err
	}

	err = c.connection.Channel.Publish(
		"",
		c.serverQueue,
		false,
		false,
		amqp.Publishing{
			CorrelationId: corrID,
			ContentType:   "application/json",
			Body:          payload,
			ReplyTo:       c.queue.Name,
		},
	)

	if err != nil {
		return err
	}

	select {
	case message := <-request:
		{
			var response Response

			_ = json.Unmarshal(message, &response)

			fmt.Printf("tmp response: %s", string(response.Result))

			_ = json.Unmarshal(response.Result, ro.Interface())
		}
	case <-time.After(c.timeout):
		return ErrTimeout
	}

	return nil
}
