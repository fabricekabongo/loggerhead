package subscription

import (
	"net"
	"sync"

	w "github.com/fabricekabongo/loggerhead/world"
)

// Subscription represents a polygon subscription for a single client.
type Subscription struct {
	// ID identifies the subscription so clients can correlate updates when
	// multiple subscriptions are active on the same connection.
	ID string
	// NS is the namespace for which updates are requested.
	NS                     string
	Lat1, Lon1, Lat2, Lon2 float64
	Conn                   net.Conn
}

type Manager struct {
	mu   sync.RWMutex
	subs map[string]*Subscription
}

func NewManager() *Manager {
	return &Manager{
		subs: make(map[string]*Subscription),
	}
}

func (m *Manager) Add(sub *Subscription) {
	m.mu.Lock()
	m.subs[sub.ID] = sub
	m.mu.Unlock()
}

func (m *Manager) RemoveByConn(conn net.Conn) {
	m.mu.Lock()
	for id, s := range m.subs {
		if s.Conn == conn {
			delete(m.subs, id)
		}
	}
	m.mu.Unlock()
}

func (m *Manager) matchingSubs(loc *w.Location) []*Subscription {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var out []*Subscription
	for _, s := range m.subs {
		if s.NS != loc.Ns() {
			continue
		}
		if loc.Lat() >= s.Lat1 && loc.Lat() <= s.Lat2 &&
			loc.Lon() >= s.Lon1 && loc.Lon() <= s.Lon2 {
			out = append(out, s)
		}
	}
	return out
}

func (m *Manager) Notify(loc *w.Location) {
	subs := m.matchingSubs(loc)
	for _, s := range subs {
		msg := "1.0," + s.ID + "," + loc.String() + "\n"
		_, _ = s.Conn.Write([]byte(msg))
	}
}
