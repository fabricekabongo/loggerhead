package server

import (
	"bufio"
	"github.com/ataul443/memnet"
	"github.com/fabricekabongo/loggerhead/query"
	"github.com/fabricekabongo/loggerhead/world"
	"testing"
	"time"
)

func TestListenerCRUDFlow(t *testing.T) {
	netListener, err := memnet.Listen(1, 4096, "test")
	if err != nil {
		t.Fatalf("Failed to create memnet listener: %v", err)
	}
	w := world.NewWorld()
	engine := query.NewQueryEngine(w)
	l := NewListener(19999, 10, time.Second, engine)

	go l.Handler.listen(netListener)
	time.Sleep(100 * time.Millisecond)

	conn, err := netListener.Dial()
	if err != nil {
		t.Fatalf("Failed to dial connection: %v", err)
	}
	defer func() {
		conn.Close()
		if h, ok := l.Handler.(*Handler); ok {
			close(h.closeChan)
		}
		netListener.Close()
	}()

	reader := bufio.NewReader(conn)

	writeAndRead := func(cmd string, lines int) string {
		_, err := conn.Write([]byte(cmd + "\n"))
		if err != nil {
			t.Fatalf("Failed to write command %s: %v", cmd, err)
		}
		var result string
		for i := 0; i < lines; i++ {
			line, err := reader.ReadString('\n')
			if err != nil {
				t.Fatalf("Failed to read response for %s: %v", cmd, err)
			}
			result += line
		}
		return result
	}

	// create
	resp := writeAndRead("SAVE ns loc 1.0 2.0", 1)
	if resp != "1.0,saved\n" {
		t.Fatalf("Unexpected response: %q", resp)
	}

	// read
	resp = writeAndRead("GET ns loc", 2)
	if resp != "1.0,ns,loc,1.000000,2.000000\n1.0,done\n" {
		t.Fatalf("Unexpected get response: %q", resp)
	}

	// update
	resp = writeAndRead("SAVE ns loc 2.0 3.0", 1)
	if resp != "1.0,saved\n" {
		t.Fatalf("Unexpected response after update: %q", resp)
	}
	resp = writeAndRead("GET ns loc", 2)
	if resp != "1.0,ns,loc,2.000000,3.000000\n1.0,done\n" {
		t.Fatalf("Unexpected get after update response: %q", resp)
	}

	// delete
	resp = writeAndRead("DELETE ns loc", 1)
	if resp != "1.0,deleted\n" {
		t.Fatalf("Unexpected delete response: %q", resp)
	}
	resp = writeAndRead("GET ns loc", 1)
	if resp != "1.0,done\n" {
		t.Fatalf("Unexpected final get response: %q", resp)
	}
}
