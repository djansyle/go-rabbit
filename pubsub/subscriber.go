package rabbit

import (
	"djansyle/rabbit"
	"fmt"
	"sync"

	"github.com/streadway/amqp"
)

// Subscriber is the interface satisfied by all listeners that can receive message
type Subscriber interface {
	Close(id uint16) error
	CloseAll() error
	Subscribe(exchange string, topic string) (<-chan amqp.Delivery, error)
}

type subscriber struct {
	id         uint16
	connection *rabbit.Connection

	messages <-chan amqp.Delivery
}

type subscribers struct {
	sync.Mutex
	url         string
	queue       string
	idCounter   uint16
	subscribers map[uint16]*subscriber
}

// CreateSubscriber creates a new instance for
func CreateSubscriber(url string, queue string) Subscriber {
	return &subscribers{subscribers: make(map[uint16]*subscriber), idCounter: 0, url: url, queue: queue}
}

func (s *subscribers) Close(id uint16) error {
	subscriber := s.subscribers[id]
	if subscriber == nil {
		return fmt.Errorf("No subscriber found with id %d", id)
	}

	err := subscriber.connection.Close()
	if err != nil {
		return err
	}

	delete(s.subscribers, id)
	return nil
}

func (s *subscribers) CloseAll() error {
	for i := range s.subscribers {
		err := s.Close(i)

		if err != nil {
			return err
		}
	}
	return nil
}

// Subscribe creates a new connection for each topic
func (s *subscribers) Subscribe(exchange string, topic string) (<-chan amqp.Delivery, error) {
	s.Lock()
	id := s.idCounter + 1
	s.idCounter = id
	s.Unlock()

	con, err := rabbit.CreateConnection(s.url)
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

	queue := s.queue
	if queue == "" {
		queue = rabbit.RandomID()
	}

	q, err := ch.QueueDeclare(
		queue,
		true,  // durable
		false, // delete when unused
		true,  // exclusive
		false, // no wait
		nil,   // arguments
	)

	if err != nil {
		return nil, err
	}

	err = ch.QueueBind(
		q.Name,   // name
		topic,    // key
		exchange, // exchange
		false,    // no-wait
		nil,      // args
	)

	if err != nil {
		return nil, err
	}

	messages, err := ch.Consume(
		q.Name, // queue name
		"",     // consumer,
		true,   // auto-ack
		false,  // exclusive
		false,  // no-local
		false,  // no-wait
		nil,
	)

	if err != nil {
		return nil, err
	}

	s.subscribers[id] = &subscriber{connection: con, id: id, messages: messages}
	return messages, nil
}
