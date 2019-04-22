package main

import (
	rpc "djansyle/rabbit/rpc"
	"os"
	"os/signal"
	"syscall"
)

// Arith type
type Arith int

// AddPayload type
type AddPayload struct {
	X int `json:"x"`
	Y int `json:"y"`
}

type Result struct {
	Result int `json:"Result"`
}

// Add method
func (a *Arith) Add(data AddPayload) (Result, error) {
	return Result{ Result: data.X + data.Y }, nil
}

// Subtract method
func (a *Arith) Subtract(data AddPayload) (Result, error) {
	return Result{ Result: data.X - data.Y }, nil
}

func main() {
	server, err := rpc.CreateServer("amqp://guest:guest@localhost:5672/", "go_test")
	if err != nil {
		println("Failed to start server")
		println(err.Error())
	}

	println("Server started")
	osSignal := make(chan os.Signal)

	signal.Notify(osSignal, syscall.SIGINT)
	signal.Notify(osSignal, syscall.SIGTERM)

	go func() {
		<-osSignal
		println("Closing server")
		server.Close()
	}()

	server.Register(new(Arith))
	server.Serve()
}
