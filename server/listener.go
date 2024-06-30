package server

import (
	"bufio"
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

func NewListener(port int, maxConn int, maxEOF time.Duration, engine query.EngineInterface) *Listener {
	return &Listener{
		Port: port,
		Handler: &Handler{
			QueryEngine:    engine,
			closeChan:      make(chan int),
			MaxConnections: maxConn,
			maxEOFWait:     maxEOF,
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
	defer func(conn net.Conn) {
		log.Println("Closing connection from: ", conn.RemoteAddr())
		err := conn.Close()
		if err != nil {
			log.Println("Error closing connection: ", err)
			return
		}
	}(conn)

	scanner := bufio.NewScanner(conn)

	var startOfEOF time.Time

	for {
		if !scanner.Scan() {
			if err := scanner.Err(); err != nil {
				log.Println("Error reading from connection ", err)
				return err
			}

			if startOfEOF.IsZero() {
				startOfEOF = time.Now()
			} else {
				if time.Since(startOfEOF) > h.maxEOFWait {
					log.Println("Connection timed out. Closing connection")
					return nil
				}
			}

			continue
		}
		startOfEOF = time.Time{}
		line := scanner.Text()
		if len(line) == 0 {
			log.Println("Empty line received. Closing connection")
			break
		}

		var response string
		response = h.QueryEngine.ExecuteQuery(line)
		_, err := conn.Write([]byte(response))
		if err != nil {
			log.Println("Error writing to connection: ", err)
			return err
		}

	}

	return nil
}
