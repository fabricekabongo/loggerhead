package server

import (
	"fmt"
	"net"
)

type Server struct {
	WriteHandler *WriteHandler
	ReadHandler  *ReadHandler
	closeChannel chan struct{}
}

func NewServer(wHandler WriteHandler, rHander ReadHandler) *Server {
	return &Server{
		WriteHandler: &wHandler,
		ReadHandler:  &rHander,
		closeChannel: make(chan struct{}),
	}
}

func (s *Server) Stop() {
	s.closeChannel <- struct{}{}
	close(s.closeChannel)
}

func (s *Server) Start() {
	fmt.Println("Opening read and write ports")
	writerListener, err := net.Listen("tcp", ":19999")
	if err != nil {
		panic(err)
	}

	subscriberListener, err := net.Listen("tcp", ":19998")
	if err != nil {
		panic(err)
	}

	go s.WriteHandler.listen(writerListener)
	go s.ReadHandler.listen(subscriberListener)

	fmt.Println("Read and Write operations ready. Enjoy!")
	<-s.closeChannel
}
