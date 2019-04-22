package main

import (
	rpc "djansyle/rabbit/rpc"
	"time"
	"log"
)
type Result struct {
	Result int `json:"Result"`
}


func main() {
	client, err := rpc.CreateClient("amqp://guest:guest@localhost:5672/", "go_test", 5 * time.Second)
	if err != nil {
		log.Fatal(err.Error())
		return
	}

	var output Result
	err = client.Send(rpc.Request{ Action: "Arith.Add", Data: []byte(`{ "x": 1, "y": 2 }`)}, &output)
	if err != nil {
		log.Fatal(err.Error())
		return
	}

	log.Printf("%x", output.Result)
}