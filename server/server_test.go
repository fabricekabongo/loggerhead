package server

import (
	"net"
	"testing"
	"time"
)

type mockHandler struct {
	listenCalled chan net.Listener
	closeCalled  chan struct{}
	closeNotify  chan struct{}
}

func newMockHandler() *mockHandler {
	return &mockHandler{
		listenCalled: make(chan net.Listener, 1),
		closeCalled:  make(chan struct{}, 1),
		closeNotify:  make(chan struct{}),
	}
}

func (m *mockHandler) listen(l net.Listener) {
	m.listenCalled <- l
	<-m.closeNotify
}

func (m *mockHandler) close() error {
	close(m.closeNotify)
	m.closeCalled <- struct{}{}
	return nil
}

func TestServerStopIdempotent(t *testing.T) {
	s := NewServer([]*Listener{})
	// call Stop multiple times; should not panic
	s.Stop()
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("Stop panicked on second call: %v", r)
		}
	}()
	s.Stop()
}

func TestServerStartAndStopClosesHandlers(t *testing.T) {
	handler := newMockHandler()
	s := NewServer([]*Listener{{Port: 0, Handler: handler, Type: TCP}})

	done := make(chan struct{})
	go func() {
		s.Start()
		close(done)
	}()

	select {
	case l := <-handler.listenCalled:
		if l == nil {
			t.Fatalf("listener was nil")
		}
	case <-time.After(time.Second):
		t.Fatalf("listen was not called")
	}

	s.Stop()

	select {
	case <-handler.closeCalled:
	case <-time.After(time.Second):
		t.Fatalf("close was not called")
	}

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatalf("server did not exit after stop")
	}
}

func TestCreateListenerPanicsOnFailure(t *testing.T) {
	s := &Server{}
	l := &Listener{Port: -1, Type: ConnectionType("invalid")}
	defer func() {
		if r := recover(); r == nil {
			t.Fatalf("expected panic creating listener")
		}
	}()

	_ = s.createListener(l)
}
