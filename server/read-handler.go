package server

import (
	"bufio"
	"github.com/fabricekabongo/loggerhead/query"
	"log"
	"net"
	"sync"
)

type SubscribeCommand struct {
	GridName string `json:"gridName"`
}

type ReadHandler struct {
	QueryEngine    *query.Engine
	MaxConnections int
	closeChan      chan struct{}
}

func NewReadHandler(engine *query.Engine, maxConnections int) *ReadHandler {
	return &ReadHandler{
		QueryEngine:    engine,
		MaxConnections: maxConnections,
		closeChan:      make(chan struct{}),
	}
}

func (r *ReadHandler) listen(listener net.Listener) {
	defer func(listener net.Listener) {
		err := listener.Close()
		if err != nil {
			log.Println("Error closing listener: ", err)
		}
	}(listener)

	waitGroup := sync.WaitGroup{}

	defer waitGroup.Wait()

	for {
		select {
		case <-r.closeChan:
			return
		default:
			conn, err := listener.Accept()
			waitGroup.Add(1)

			if err != nil {
				panic(err)
			}

			go r.handleReadConnection(conn)
		}
	}
}

func (r *ReadHandler) handleReadConnection(conn net.Conn) {
	log.Println("New read connection from: ", conn.RemoteAddr())

	defer func(conn net.Conn) {
		err := conn.Close()
		if err != nil {
			log.Println("Error closing connection: ", err)
		}
	}(conn)

	scanner := bufio.NewScanner(conn) // to listen to new subscription

	waitGroup := sync.WaitGroup{}
	defer waitGroup.Wait()

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {

			return
		}

		command := string(line)
		response := r.QueryEngine.Execute(command)
		_, err := conn.Write([]byte(response + "\n"))
		if err != nil {
			log.Println("Error writing response: ", err)
			return
		}
	}
}
