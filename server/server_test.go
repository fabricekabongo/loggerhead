package server

import "testing"

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
