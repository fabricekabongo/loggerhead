package server

import (
	"errors"
	"github.com/ataul443/memnet"
	"github.com/fabricekabongo/loggerhead/query"
	"github.com/fabricekabongo/loggerhead/world"
	"math/rand/v2"
	"net"
	"strconv"
	"testing"
	"time"
)

type testEngine string

func (e testEngine) ExecuteQuery(string) string {
	return string(e)
}

type errListener struct {
	closed bool
}

func (errListener) Accept() (net.Conn, error) {
	return nil, errors.New("accept failed")
}

func (l *errListener) Close() error {
	l.closed = true
	return nil
}

func (errListener) Addr() net.Addr { return nil }

func CreateRandomLocation(seed int) (string, string, float64, float64) {
	lat := -90.0 + rand.Float64()*(90.0+90.0)
	lon := -180.0 + rand.Float64()*(180.0+180.0)
	id := strconv.Itoa(seed)

	return "1", id, lat, lon
}

func TestNewListenerInitializesHandler(t *testing.T) {
	engine := testEngine("ok")
	listener := NewListener(1234, 5, 2*time.Second, engine)

	if listener.Port != 1234 {
		t.Fatalf("unexpected port: %d", listener.Port)
	}

	handler, ok := listener.Handler.(*Handler)
	if !ok {
		t.Fatalf("handler was not of type *Handler")
	}

	if handler.MaxConnections != 5 {
		t.Fatalf("unexpected max connections: %d", handler.MaxConnections)
	}
	if handler.maxEOFWait != 2*time.Second {
		t.Fatalf("unexpected eof wait: %s", handler.maxEOFWait)
	}
	if handler.QueryEngine != engine {
		t.Fatalf("unexpected engine configured")
	}
	if listener.Type != TCP {
		t.Fatalf("expected listener type %q, got %q", TCP, listener.Type)
	}
}

func TestHandleConnectionProcessesQueries(t *testing.T) {
	handler := &Handler{
		QueryEngine:    testEngine("pong"),
		closeChan:      make(chan int),
		MaxConnections: 1,
		maxEOFWait:     time.Second,
	}

	serverConn, clientConn := net.Pipe()
	t.Cleanup(func() {
		serverConn.Close()
		clientConn.Close()
	})

	errCh := make(chan error, 1)
	go func() {
		errCh <- handler.handleConnection(serverConn)
	}()

	if _, err := clientConn.Write([]byte("PING\n")); err != nil {
		t.Fatalf("failed to write query: %v", err)
	}

	if err := clientConn.SetReadDeadline(time.Now().Add(time.Second)); err != nil {
		t.Fatalf("failed to set read deadline: %v", err)
	}
	buf := make([]byte, 4)
	if _, err := clientConn.Read(buf); err != nil {
		t.Fatalf("failed to read response: %v", err)
	}
	if string(buf) != "pong" {
		t.Fatalf("unexpected response: %q", string(buf))
	}

	if _, err := clientConn.Write([]byte("\n")); err != nil {
		t.Fatalf("failed to write terminator: %v", err)
	}

	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("handleConnection returned error: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatalf("handleConnection did not return")
	}
}

func TestHandleConnectionReturnsAfterEOFTimeout(t *testing.T) {
	handler := &Handler{
		QueryEngine:    testEngine("ignored"),
		closeChan:      make(chan int),
		MaxConnections: 1,
		maxEOFWait:     10 * time.Millisecond,
	}

	serverConn, clientConn := net.Pipe()
	t.Cleanup(func() {
		serverConn.Close()
		clientConn.Close()
	})

	errCh := make(chan error, 1)
	go func() {
		errCh <- handler.handleConnection(serverConn)
	}()

	if err := clientConn.Close(); err != nil {
		t.Fatalf("failed to close client connection: %v", err)
	}

	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("expected nil error after EOF timeout, got %v", err)
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatalf("handleConnection did not exit after EOF timeout")
	}
}

func TestListenClosesListenerOnAcceptError(t *testing.T) {
	handler := &Handler{
		QueryEngine:    testEngine("ignored"),
		closeChan:      make(chan int),
		MaxConnections: 1,
		maxEOFWait:     time.Second,
	}

	l := &errListener{}
	handler.listen(l)

	if !l.closed {
		t.Fatalf("expected listener to be closed on error")
	}
}

func BenchmarkListener(b *testing.B) {
	b.Run("NewListener", func(b *testing.B) {
		netListener, err := memnet.Listen(1, 4096, "bob")
		if err != nil {
			b.Fatal("Failed to create a memnet listener: ", err)
		}
		w := world.NewWorld()
		engine := query.NewQueryEngine(w)
		l := NewListener(19999, 100, 20*time.Second, engine)

		go l.Handler.listen(netListener)
		time.Sleep(5 * time.Second)

		conn, err := netListener.Dial()
		if err != nil {
			b.Fatal("Failed to dial the connection: ", err)
		}
		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			ns, id, lat, lon := CreateRandomLocation(i)
			_, err := conn.Write([]byte("SAVE " + ns + " " + id + " " + strconv.FormatFloat(lat, 'f', -1, 64) + " " + strconv.FormatFloat(lon, 'f', -1, 64) + "\n"))
			if err != nil {
				b.Fatal("Failed to write to the connection: ", err)
			}
		}
	})
}
