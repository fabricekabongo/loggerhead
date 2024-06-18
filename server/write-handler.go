package server

import (
	"bufio"
	"github.com/fabricekabongo/loggerhead/query"
	"github.com/hashicorp/memberlist"
	"log"
	"net"
	"sync"
	"time"
)

type WriteCommand struct {
	LodId string  `json:"loc_id"`
	Lat   float64 `json:"lat"`
	Lon   float64 `json:"lon"`
}

type WriteHandler struct {
	QueryEngine   *query.Engine
	closeChan     chan struct{}
	Broadcasts    *memberlist.TransmitLimitedQueue
	MaxConnection int
	maxEOFWait    time.Duration
}

func NewWriteHandler(engine *query.Engine, broadcasts *memberlist.TransmitLimitedQueue, maxConnections int) *WriteHandler {
	return &WriteHandler{
		QueryEngine:   engine,
		closeChan:     make(chan struct{}),
		Broadcasts:    broadcasts,
		MaxConnection: maxConnections,
		maxEOFWait:    30 * time.Second, // 1 minutes
	}
}

func (w *WriteHandler) listen(listener net.Listener) {
	defer func(listener net.Listener) {
		err := listener.Close()
		if err != nil {
			log.Println("Error closing write listener: ", err)
		}
	}(listener)

	waitGroup := sync.WaitGroup{}
	workLimit := make(chan int, w.MaxConnection)

	defer waitGroup.Wait()

	for {
		select {
		case <-w.closeChan:
			return
		default:
			conn, err := listener.Accept()
			if err != nil {
				log.Println("Error accepting connection: ", err)
				continue
			}
			waitGroup.Add(1)
			workLimit <- 0

			if err != nil {
				panic(err)
			}

			go func(conn net.Conn) {
				defer func() {
					<-workLimit
					waitGroup.Done()
				}()

				err := w.handleWriteConnection(conn)
				if err != nil {
					log.Println("Error handling write connection: ", err)
					return
				}

			}(conn)
		}
	}
}

func (w *WriteHandler) handleWriteConnection(conn net.Conn) error {
	log.Println("New write connection from: ", conn.RemoteAddr())

	defer func(conn net.Conn) {
		err := conn.Close()
		if err != nil {
			log.Println("Error closing connection: ", err)
		}
	}(conn)

	wait := 1 * time.Second

	scanner := bufio.NewScanner(conn)

	for wait <= w.maxEOFWait {
		for scanner.Scan() {
			line := scanner.Text()
			if len(line) == 0 {
				break
			}
			wait = 1 * time.Second // Reset wait time

			response := w.QueryEngine.Execute(line)
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
