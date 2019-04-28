package rabbit

import (
	"github.com/streadway/amqp"
)

// Connection holds the information of rabbitmq Connection
type Connection struct {
	Connection *amqp.Connection
	Channel    *amqp.Channel
}

// CreateConnection creates a new client
func CreateConnection(url string) (*Connection, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, err
	}

	ch, err := conn.Channel()
	if err != nil {
		return nil, err
	}

	return &Connection{Connection: conn, Channel: ch}, nil
}

// Close closes the rabbit connection and it's channels
func (conn *Connection) Close() error {
	err := conn.Channel.Close()
	if err != nil {
		return err
	}

	err = conn.Connection.Close()
	if err != nil {
		return err
	}

	return nil
}
