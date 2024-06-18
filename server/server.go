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

func NewServer(writeHandler WriteHandler, readHandler ReadHandler) *Server {
	return &Server{
		WriteHandler: &writeHandler,
		ReadHandler:  &readHandler,
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

	readListener, err := net.Listen("tcp", ":19998")
	if err != nil {
		panic(err)
	}

	go s.WriteHandler.listen(writerListener)
	go s.ReadHandler.listen(readListener)

	fmt.Println("Read and Write operations ready. Enjoy!")
	<-s.closeChannel
}
