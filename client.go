package rabbit

import (
	"encoding/json"
	"log"
	"reflect"
	"time"

	"github.com/streadway/amqp"
)

// ErrInvalidOutput error when the service method is not supported
type ErrInvalidOutput struct {
	Type reflect.Type
}

type RequestFormatter = func(interface{}) (interface{}, error)

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

	connection       *Connection
	queue            *amqp.Queue
	requestFormatter RequestFormatter
}

// CreateClientOption is the struct that holds the information of the new client
type CreateClientOption struct {
	URL            string
	Queue          string
	TimeoutRequest time.Duration
}

// CreateClient creates a new client for rpc server
func CreateClient(opt *CreateClientOption, requestFormatter RequestFormatter) (Sender, error) {
	conn, err := CreateConnection(opt.URL)
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
			log.Printf("Message received: %s", string(d.Body))
			if request == nil {
				log.Printf("Message unexpectedly arrive with correlation id %s.", d.CorrelationId)
				continue
			}

			request <- d.Body
			delete(requests, d.CorrelationId)
			ack := d.Ack(false)
			if ack != nil {
				log.Printf("Failed to ack: %v", err)
			}
		}
	}()

	return &client{
		requests:         requests,
		serverQueue:      opt.Queue,
		connection:       conn,
		timeout:          opt.TimeoutRequest,
		queue:            &q,
		requestFormatter: requestFormatter,
	}, nil
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

	message, err := c.requestFormatter(message)
	if err != nil {
		return err
	}

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

			err = json.Unmarshal(message, &response)
			if err != nil {
				return err
			}

			err = json.Unmarshal(response.Result, ro.Interface())
			if err != nil {
				return err
			}
		}
	case <-time.After(c.timeout):
		return ErrTimeout
	}

	return nil
}
