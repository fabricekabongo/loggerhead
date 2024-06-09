package server

import (
	"bufio"
	"encoding/json"
	"github.com/fabricekabongo/loggerhead/clustering"
	"github.com/fabricekabongo/loggerhead/world"
	"github.com/hashicorp/memberlist"
	"log"
	"net"
	"sync"
)

type WriteCommand struct {
	LodId string  `json:"loc_id"`
	Lat   float64 `json:"lat"`
	Lon   float64 `json:"lon"`
}

type WriteHandler struct {
	WorldMap   *world.Map
	closeChan  chan struct{}
	Broadcasts *memberlist.TransmitLimitedQueue
}

func NewWriteHandler(world *world.Map, broadcasts *memberlist.TransmitLimitedQueue) *WriteHandler {
	return &WriteHandler{
		WorldMap:   world,
		closeChan:  make(chan struct{}),
		Broadcasts: broadcasts,
	}
}

func (w *WriteHandler) listen(listener net.Listener) {
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
		case <-w.closeChan:
			return
		default:
			conn, err := listener.Accept()
			waitGroup.Add(1)

			if err != nil {
				panic(err)
			}

			go w.handleWriteConnection(conn)
		}
	}
}

func (w *WriteHandler) handleWriteConnection(conn net.Conn) {
	log.Println("New write connection from: ", conn.RemoteAddr())

	defer func(conn net.Conn) {
		err := conn.Close()
		if err != nil {
			log.Println("Error closing connection: ", err)
		}
	}(conn)

	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			break
		}

		w.handleWriterCommand(line)
	}
}

func (w *WriteHandler) handleWriterCommand(line []byte) {
	var location world.LocationEntity
	err := json.Unmarshal(line, &location)

	go func(location world.LocationEntity) {
		broadcast := clustering.NewLocationBroadcast(location)
		w.Broadcasts.QueueBroadcast(broadcast)
	}(location)

	if err != nil {
		log.Println("Error parsing command: ", err, line)
		return
	}

	err = w.WorldMap.Save(location.LocId, location.Lat, location.Lon)
	if err != nil {
		log.Println("Error saving location to map: ", err)
		return
	}
}
