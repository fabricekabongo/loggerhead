package server

import (
	"bufio"
	"github.com/fabricekabongo/loggerhead/config"
	"github.com/fabricekabongo/loggerhead/query"
	"log"
	"net"
	"time"
)

type Handler struct {
	QueryEngine    query.EngineInterface
	closeChan      chan int
	MaxConnections int
	maxEOFWait     time.Duration
}

func NewListener(config config.Config, engine query.EngineInterface) *Listener {
	return &Listener{
		Port: config.ReadPort,
		Handler: &Handler{
			QueryEngine:    engine,
			closeChan:      make(chan int),
			MaxConnections: config.MaxConnections,
			maxEOFWait:     30 * time.Second,
		},
		Type: TCP,
	}
}

func (h *Handler) close() error {
	h.closeChan <- 0
	close(h.closeChan)
	return nil
}

func (h *Handler) listen(listener net.Listener) {
	defer func(listener net.Listener) {
		err := listener.Close()
		if err != nil {
			log.Println("Error closing write listener: ", err)
		}
	}(listener)

	workLimit := make(chan int, h.MaxConnections)

	for {
		select {
		case <-h.closeChan:
			return
		default:
			conn, err := listener.Accept()
			if err != nil {
				log.Println("Error accepting connection: ", err)
				return
			}
			workLimit <- 0

			go func(conn net.Conn) {
				defer func() {
					<-workLimit
				}()

				err := h.handleConnection(conn)
				if err != nil {
					log.Println("Error handling write connection: ", err)
					return
				}

			}(conn)
		}
	}
}

func (h *Handler) handleConnection(conn net.Conn) error {
	log.Println("New connection from: ", conn.RemoteAddr())

	defer func(conn net.Conn) {
		err := conn.Close()
		if err != nil {
			log.Println("Error closing connection: ", err)
		}
	}(conn)

	wait := 1 * time.Second
	scanner := bufio.NewScanner(conn)

	for wait <= h.maxEOFWait {
		for scanner.Scan() {
			line := scanner.Text()
			if len(line) == 0 {
				break
			}
			wait = 1 * time.Second // Reset wait time
			var response string
			response = h.QueryEngine.ExecuteQuery(line)
			_, err := conn.Write([]byte(response))
			if err != nil {
				log.Println("Error writing to connection: ", err)
				return err
			}
		}

		if err := scanner.Err(); err != nil {
			log.Println("Error reading from connection ", err)
			return err
		}

		if scanner.Err() == nil {
			wait *= 2 // Exponential backoff
			log.Println("EOF detected. Waiting for ", wait, " before trying again")
			time.Sleep(wait)
		}
	}

	return nil
}
