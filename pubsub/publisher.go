package rabbit

import (
	"djansyle/rabbit"
	"github.com/streadway/amqp"
)

// Publisher is the interface satisfied by all clients that can publish
type Publisher interface {
	Publish([]byte) error
}

// RabbitPublisher holds all the information of the publisher
type rabbitPublisher struct {
	topics   []string
	exchange string // name of the exchange

	connection *rabbit.Connection
}

// CreatePublisher creates a new instance for a publisher
func CreatePublisher(uri string, exchange string, topics []string) (Publisher, error) {
	conn, err := rabbit.CreateConnection(uri)

	if err != nil {
		return nil, err
	}

	err = conn.Channel.ExchangeDeclare(
		exchange, // name
		"topic",  // kind
		true,     // durable
		false,    // autoDelete,
		false,    // internal
		false,    // no-wait
		nil,      // args
	)

	if err != nil {
		return nil, err
	}

	return (&rabbitPublisher{topics: topics, exchange: exchange, connection: conn}), nil
}

// Publish a new message to the connection
func (rp *rabbitPublisher) Publish(message []byte) error {
	for i := 0; i < len(rp.topics); i++ {
		err := rp.connection.Channel.Publish(
			rp.exchange,  // exchange
			rp.topics[i], // key
			false,        // mandatory
			false,        // immediate
			amqp.Publishing{
				ContentType: "application/json",
				Body:        message,
			},
		)

		if err != nil {
			return err
		}
	}

	return nil
}
