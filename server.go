package rabbit

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"sync"

	"github.com/streadway/amqp"
)

var typeOfApplicationError = reflect.TypeOf(&(ApplicationError{}))

var unsupportedReplyType = []reflect.Kind{
	reflect.Ptr,
	reflect.Chan,
	reflect.Func,
	reflect.Map,
	reflect.Slice,
	reflect.UnsafePointer,
}

// Server is the interface implemented for all servers
type Server interface {
	Close()
	Register(interface{}, bool)
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
	name    string // name of the service
	rcvr    reflect.Value
	methods map[string]*methodType
}

// rpcServer instance holds the information of the server
type rpcServer struct {
	connection *Connection
	messages   <-chan amqp.Delivery
	done       chan bool

	serviceMap sync.Map // map[string]*service

	methods map[string]*methodType
}

// CreateServer creates a new server
func CreateServer(url string, queue string) (Server, error) {
	conn, err := CreateConnection(url)
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
		true,  // durable
		false, // auto-delete
		false, // exclusive
		false, // no-wait
		nil,   // args
	)

	if err != nil {
		return nil, err
	}

	messages, err := ch.Consume(
		q.Name,
		"",    // consumer
		false, // auto-ack
		false, // exclusive
		false, // no-local
		false, // no-wait
		nil,   // args
	)

	return &rpcServer{connection: conn, messages: messages, done: make(chan bool)}, nil
}

// Close the connection of the server
func (server *rpcServer) Close() {
	if server == nil {
		return
	}

	server.done <- true

	if server.connection != nil {
		e := server.connection.Close()
		if e != nil {
			fmt.Printf("Failed to close connection: %v", e)
		}
	}

	return
}

func isTypeSupported(t reflect.Kind) bool {
	for i := 0; i < len(unsupportedReplyType); i++ {
		if t == unsupportedReplyType[i] {
			return false
		}
	}

	return true
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

		if mtype.NumOut() != 2 {
			fmt.Printf("rpc.Register: method %q needs exactly 2 parameter but got %d", mname, mtype.NumOut())
		}

		replyType := mtype.Out(0).Kind()
		if !isTypeSupported(replyType) {
			fmt.Printf("rpc.Register: method %q first out parameter type is not supported", mname)
		}

		if errType := mtype.Out(1); errType != typeOfApplicationError {
			fmt.Printf("rpc.Register: method %q's, second out parameter type must be an %q but got %q", mname, typeOfApplicationError, errType)
		}
	}
	return methods
}

// Register a service that handles a request
func (server *rpcServer) Register(rcvr interface{}, noScope bool) {
	s := new(service)

	s.methods = suitableMethods(rcvr)
	s.rcvr = reflect.ValueOf(rcvr)
	s.name = ""

	if !noScope {
		s.name = reflect.Indirect(reflect.ValueOf(rcvr)).Type().Name()
	}

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

func process(server *rpcServer, msg *amqp.Delivery) {
	var body Request
	fmt.Printf("rpc.process: message received: %s", string(msg.Body))
	if err := json.Unmarshal(msg.Body, &body); err != nil {
		reply(
			server,
			msg,
			nil,
			&ApplicationError{
				"500",
				"Failed to parse data",
				struct{ body string }{string(msg.Body)},
			})
		return
	}

	dot := strings.LastIndex(body.Action, ".")
	serviceName := ""
	methodName := body.Action

	if dot != -1 {
		serviceName = body.Action[:dot]
		methodName = body.Action[dot+1:]
	}

	svci, _ := server.serviceMap.Load(serviceName)
	if svci == nil {
		fmt.Printf("rpc.process: service %q not found", serviceName)
		reply(
			server,
			msg,
			nil,
			&ApplicationError{
				"500",
				fmt.Sprintf("Service %q not found", serviceName),
				nil,
			})
		return
	}

	svc := svci.(*service)
	fn := svc.methods[methodName]

	if fn == nil {
		fmt.Printf("rpc.process: service %q has no method %q", serviceName, methodName)
		return
	}

	param := reflect.New(fn.typ.In(1))
	if err := json.Unmarshal(body.Data, param.Interface()); err != nil {
		reply(
			server,
			msg,
			nil,
			&ApplicationError{
				"500",
				"Failed to parse data",
				struct{ body string }{string(msg.Body)},
			})
		return
	}

	result := fn.method.Func.Call([]reflect.Value{fn.rcvr, param.Elem()})

	err := result[1].Interface().(*ApplicationError)
	if err != nil {
		reply(server, msg, nil, err)
		return
	}

	reply(server, msg, result[0].Interface(), nil)
}

func getReplyPayload(data interface{}, appErr *ApplicationError) (reply []byte) {
	var err = []byte("null")
	var ae = []byte("null")

	fmt.Println(fmt.Sprintf("tmp reply: %v null", data))

	if data == nil && appErr == nil {
		reply, _ = json.Marshal(Response{
			Error:  err,
			Result: ae,
		})
		return
	}

	if appErr != nil {
		err, _ = json.Marshal(*appErr)
	}

	if data != nil {
		ae, _ = json.Marshal(data)
	}

	ae, _ = json.Marshal(data)
	reply, _ = json.Marshal(Response{
		Error:  err,
		Result: ae,
	})

	return
}

func reply(server *rpcServer, msg *amqp.Delivery, data interface{}, appErr *ApplicationError) {
	response := getReplyPayload(data, appErr)

	err := server.connection.Channel.Publish(
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
		fmt.Printf("Failed to send message: %v", err)
		return
	}
	_ = msg.Ack(false)
}
