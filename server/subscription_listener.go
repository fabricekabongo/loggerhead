package server

import (
	"bufio"
	"github.com/fabricekabongo/loggerhead/subscription"
	w "github.com/fabricekabongo/loggerhead/world"
	"net"
	"strconv"
	"strings"
	"time"
)

// SubscriptionHandler listens for subscription requests and keeps the connection open.
type SubscriptionHandler struct {
	World          *w.World
	Manager        *subscription.Manager
	closeChan      chan int
	MaxConnections int
	maxEOFWait     time.Duration
}

func NewSubscriptionListener(port int, maxConn int, maxEOF time.Duration, world *w.World, mgr *subscription.Manager) *Listener {
	return &Listener{
		Port: port,
		Handler: &SubscriptionHandler{
			World:          world,
			Manager:        mgr,
			closeChan:      make(chan int),
			MaxConnections: maxConn,
			maxEOFWait:     maxEOF,
		},
		Type: TCP,
	}
}

func (h *SubscriptionHandler) close() error {
	h.closeChan <- 0
	close(h.closeChan)
	return nil
}

func (h *SubscriptionHandler) listen(listener net.Listener) {
	defer listener.Close()
	workLimit := make(chan int, h.MaxConnections)
	for {
		select {
		case <-h.closeChan:
			return
		default:
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			workLimit <- 0
			go func(c net.Conn) {
				defer func() { <-workLimit }()
				_ = h.handleConnection(c)
			}(conn)
		}
	}
}

func (h *SubscriptionHandler) handleConnection(conn net.Conn) error {
	scanner := bufio.NewScanner(conn)
	var startOfEOF time.Time
	for {
		if !scanner.Scan() {
			if err := scanner.Err(); err != nil {
				conn.Close()
				return err
			}
			if startOfEOF.IsZero() {
				startOfEOF = time.Now()
			} else if time.Since(startOfEOF) > h.maxEOFWait {
				conn.Close()
				return nil
			}
			continue
		}
		line := scanner.Text()
		parts := strings.Split(line, " ")
		if len(parts) != 8 || parts[0] != "SUBPOLY" {
			conn.Write([]byte("1.0,\"invalid query\"\n"))
			continue
		}
		ns := parts[1]
		subID := parts[2]
		dumpFlag := parts[3]
		lat1, _ := strconv.ParseFloat(parts[4], 64)
		lon1, _ := strconv.ParseFloat(parts[5], 64)
		lat2, _ := strconv.ParseFloat(parts[6], 64)
		lon2, _ := strconv.ParseFloat(parts[7], 64)
		sub := &subscription.Subscription{
			ID:   subID,
			NS:   ns,
			Lat1: lat1,
			Lon1: lon1,
			Lat2: lat2,
			Lon2: lon2,
			Conn: conn,
		}
		h.Manager.Add(sub)
		if dumpFlag == "true" {
			locations := h.World.QueryRange(ns, lat1, lat2, lon1, lon2)
			for _, l := range locations {
				conn.Write([]byte("1.0," + subID + "," + l.String() + "\n"))
			}
		}
		conn.Write([]byte("1.0,subscribed\n"))
		// keep connection open until the client closes it
		buf := make([]byte, 1)
		for {
			_, err := conn.Read(buf)
			if err != nil {
				break
			}
		}
		h.Manager.RemoveByConn(conn)
		conn.Close()
		return nil
	}
}
