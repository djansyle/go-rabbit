package rabbit

import (
	uuid "github.com/satori/go.uuid"
	"github.com/streadway/amqp"
)

// Subscriber holds the information of the subscriber
type Subscriber struct {
	queue      *amqp.Queue
	exchange   string
	connection *Connection
	Messages   <-chan amqp.Delivery
}

// CreateSubscriber creates a new instance for
func CreateSubscriber(url string, exchange string) (*Subscriber, error) {
	con, err := CreateConnection(url)
	if err != nil {
		return nil, err
	}

	ch := con.Channel

	err = ch.ExchangeDeclare(
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

	q, err := ch.QueueDeclare(
		uuid.NewV4().String(),
		false, // durable
		true,  // delete when unused
		false, // exclusive
		false, // no wait
		nil,   // arguments
	)

	if err != nil {
		return nil, err
	}

	messages, err := ch.Consume(
		q.Name, // queue name
		"",     // consumer,
		true,   // auto-ack
		true,   // exclusive
		false,  // no-local
		false,  // no-wait
		nil,
	)

	if err != nil {
		return nil, err
	}

	return &Subscriber{queue: &q, connection: con, Messages: messages, exchange: exchange}, nil
}

// Close the connection of the rabbitmq
func (s *Subscriber) Close() error {
	err := s.connection.Close()
	if err != nil {
		return err
	}

	return nil
}

// AddTopics to the current connection's channel
func (s *Subscriber) AddTopics(topics []string) error {
	for _, topic := range topics {
		err := s.connection.Channel.QueueBind(
			s.queue.Name, // name
			topic,        // key
			s.exchange,   // exchange
			false,        // no-wait
			nil,          // args
		)

		if err != nil {
			return nil
		}
	}

	return nil
}
