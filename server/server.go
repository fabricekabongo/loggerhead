package server

import (
	"net"
	"strconv"
	"sync"
)

type Server struct {
	listeners    []*Listener
	closeChannel chan int
	stopOnce     sync.Once
}

type ConnectionType string

const (
	TCP ConnectionType = "tcp"
	UDP ConnectionType = "udp"
)

type Listener struct {
	Port    int
	Handler ListenerHandler
	Type    ConnectionType
}

type ListenerHandler interface {
	listen(listener net.Listener)
	close() error
}

func NewServer(listeners []*Listener) *Server {
	return &Server{
		listeners:    listeners,
		closeChannel: make(chan int),
		stopOnce:     sync.Once{},
	}
}

func (s *Server) Stop() {
	s.stopOnce.Do(func() {
		close(s.closeChannel)
	})
}

func (s *Server) Start() {
	for _, listener := range s.listeners {
		go s.startListener(listener)
	}
	<-s.closeChannel
	for _, listener := range s.listeners {
		err := listener.Handler.close()
		if err != nil {
			continue
		}
	}
}

func (s *Server) startListener(listener *Listener) {
	netListener := s.createListener(listener)
	listener.Handler.listen(netListener)
}

func (*Server) createListener(listener *Listener) net.Listener {
	netListener, err := net.Listen(string(listener.Type), ":"+strconv.Itoa(listener.Port))
	if err != nil {
		panic("Error creating listener: " + err.Error())
	}
	return netListener
}
