package rpc

import (
	"encoding/json"
	rabbit "djansyle/rabbit"
	"github.com/streadway/amqp"
	"reflect"
	"sync"
	"strings"
	"log"
	"fmt"
)

// Server is the interface implemented for all servers
type Server interface {
	Close()
	Register(interface{})
	Serve()
}

type methodType struct {
	sync.Mutex

	name   string
	typ    reflect.Type
	rcvr   reflect.Value
	method reflect.Method
}

type service struct {
	name string // name of the service
	rcvr reflect.Value
	methods map[string]*methodType
}

// rpcServer instance holds the information of the server
type rpcServer struct {
	connection *rabbit.Connection
	messages   <-chan amqp.Delivery
	done       chan bool

	serviceMap sync.Map // map[string]*service

	methods map[string]*methodType
}

// CreateServer creates a new server
func CreateServer(url string, queue string) (Server, error) {
	conn, err := rabbit.CreateConnection(url)
	if err != nil {
		return nil, err
	}

	ch := conn.Channel
	err = ch.Qos(
		1,     // prefetchCount
		0,     // prefetchSize
		false, // global
	)

	if err != nil {
		return nil, err
	}

	q, err := ch.QueueDeclare(
		queue, // name
		true, // durable
		false, // auto-delete
		false, // exclusive
		false, // no-wait
		nil, // args
	)

	messages, err := ch.Consume(
		q.Name,
		"",    // consumer
		false, // auto-ack
		false, // exclusive
		false, // no-local
		false, // no-wait
		nil,   // args
	)

	return (&rpcServer{connection: conn, messages: messages, done: make(chan bool) }), nil
}

// Close the connection of the server
func (server *rpcServer) Close() {
	if server == nil {
		return
	}

	server.done <- true

	if server.connection != nil {
		server.connection.Close()
	}

	return
}

func suitableMethods(rcvr interface{}) map[string]*methodType {
	methods := make(map[string]*methodType)
	typ := reflect.TypeOf(rcvr)
	for m := 0; m < typ.NumMethod(); m++ {
		method := typ.Method(m)
		mtype := method.Type
		mname := method.Name
		mrcvr := reflect.ValueOf(rcvr)
		methods[mname] = &methodType{method: method, typ: mtype, name: mname, rcvr: mrcvr}
	}
	return methods
}

// Register a service that handles a request
func (server *rpcServer) Register(rcvr interface{}) {
	s := new(service)

	s.methods = suitableMethods(rcvr)
	s.rcvr = reflect.ValueOf(rcvr)
	s.name = reflect.Indirect(reflect.ValueOf(rcvr)).Type().Name()

	if _, exists := server.serviceMap.LoadOrStore(s.name, s); exists {
		panic("Service is already registered.")
	}
}

// Serve start the server
func (server *rpcServer) Serve() {
	finish := false

	for !finish {
		select {
		case msg := <-server.messages:
			go process(server, &msg)
		case <-server.done:
			finish = true
		}
	}
}


type response struct {
	err    error
	result interface{}
}

func process(server *rpcServer, msg *amqp.Delivery) {
	var body Request
	log.Printf("Message recieved: %s", string(msg.Body))
	if err := json.Unmarshal(msg.Body, &body); err != nil {
		reply(server, msg, []byte("{ \"error\": \"Failed to parse data.\"}"))
		return
	}

	dot := strings.LastIndex(body.Action, ".")

	serviceName := body.Action[:dot]
	methodName := body.Action[dot+1:]

	svci, _ := server.serviceMap.Load(serviceName);
	if svci == nil {
		log.Printf("No service")
		reply(server, msg, []byte(fmt.Sprintf(`{"error": "No service %s being registered." }`, serviceName)))
		return
	}

	svc := svci.(*service)
	fn := svc.methods[methodName]

	if fn == nil {
		log.Printf("No method found %s", msg.Body)
		return
	}

	param := reflect.New(fn.typ.In(1))
	if err := json.Unmarshal(body.Data, param.Interface()); err != nil {
		log.Printf("Error: %s", err.Error())
		reply(server, msg, []byte("{ \"error\": \"Failed to parse data.\"}"))
		return
	}

	result := fn.method.Func.Call([]reflect.Value{fn.rcvr, param.Elem()})

	errInter := result[1].Interface()
	if errInter != nil {
		log.Printf("Failing: %s", errInter.(error).Error())
		reply(server, msg, errInter)
		return
	}


	log.Printf("Replying: %v", result[0].Interface())
	reply(server, msg, result[0].Interface())
	msg.Ack(false)
}

func reply(server *rpcServer, msg *amqp.Delivery, data interface{}) {
	response, err := json.Marshal(data)
	if err != nil {
		log.Printf("Failed to encode interface: %v", err)
		return
	}

	err = server.connection.Channel.Publish(
		"",
		msg.ReplyTo,
		false,
		false,
		amqp.Publishing{
			ContentType:   "application/json",
			CorrelationId: msg.CorrelationId,
			Body:          response,
		},
	)

	if err != nil {
		log.Printf("Failed to publish message: %v", err)
		return
	}
}
