package subscription

import (
	"net"
	"testing"

	w "github.com/fabricekabongo/loggerhead/world"
)

func TestNotifyIncludesSubscriptionID(t *testing.T) {
	mgr := NewManager()
	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()

	sub := &Subscription{
		ID:   "mysub",
		NS:   "ns",
		Lat1: 0,
		Lon1: 0,
		Lat2: 2,
		Lon2: 2,
		Conn: server,
	}
	mgr.Add(sub)

	loc, err := w.NewLocation("ns", "id1", 1, 1)
	if err != nil {
		t.Fatalf("failed to create location: %v", err)
	}

	done := make(chan struct{})
	go func() {
		mgr.Notify(loc)
		close(done)
	}()

	buf := make([]byte, 64)
	n, err := client.Read(buf)
	if err != nil {
		t.Fatalf("failed to read from client: %v", err)
	}
	<-done

	got := string(buf[:n])
	want := "1.0,mysub,ns,id1,1.000000,1.000000\n"
	if got != want {
		t.Fatalf("expected %q got %q", want, got)
	}
}
