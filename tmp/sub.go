package main

import (
	"fmt"
	rabbit "djansyle/rabbit"
)

func main() {
	sub := rabbit.CreateSubscriber("amqp://guest:guest@localhost:5672/", "")

	messages, err := sub.Subscribe("OneWallet", "reactor.#")
	if err != nil {
		fmt.Sprintf("Erroring: %s", err.Error())
	}

	forever := make(chan bool)

	go func() {
		for d := range messages {
			fmt.Printf("[x] %s", d.Body)
		}
	}()

	<- forever
}