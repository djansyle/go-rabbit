package rpc

import (
	"djansyle/rabbit"
	"encoding/json"
	"log"
	"time"

	"github.com/streadway/amqp"
)

// Sender is the interface that satisfies the sending of messages to the server
type Sender interface {
	Send(interface{}, interface{}) error
}

type client struct {
	requests    map[string]chan []byte
	serverQueue string
	timeout     time.Duration

	connection *rabbit.Connection
	queue      *amqp.Queue
}

// CreateClient creates a new client for rpc server
func CreateClient(url string, serverQueue string, timeoutRequest time.Duration) (Sender, error) {
	conn, err := rabbit.CreateConnection(url)
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

	msgs, err := ch.Consume(
		q.Name, // queue
		"",     // consumer
		true,   // auto-ack
		false,  // exclusive
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
			d.Ack(false)
		}
	}()

	return (&client{requests: requests, serverQueue: serverQueue, connection: conn, timeout: timeoutRequest, queue: &q}), nil
}

// Sends the message to the rpc server
func (c *client) Send(message interface{}, output interface{}) error {
	corrID := rabbit.RandomID()

	request := make(chan []byte)
	defer close(request)

	c.requests[corrID] = request

	payload, err := json.Marshal(message)
	if err != nil {
		return err
	}

	c.connection.Channel.Publish(
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

	select {
	case message := <-request:
		{
			err := json.Unmarshal(message, output)
			if err != nil {
				return err
			}
		}
	case <-time.After(c.timeout):
		return rabbit.ErrTimeOut
	}

	return nil
}
