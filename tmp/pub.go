package main

import (
	"fmt"
	rabbit "djansyle/rabbit"
)

func main() {
	topics := []string{"reactor.123"}
	pub, err := rabbit.CreatePublisher("amqp://guest:guest@localhost:5672/", "OneWallet", topics)

	if err != nil {
		fmt.Printf(err.Error())
	}

	err = pub.Publish([]byte("hello world"))
	if err != nil {
		fmt.Printf("%x", err)
	}
}