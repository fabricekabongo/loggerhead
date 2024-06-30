package world

import (
	"encoding/gob"
	"sync"
)

func init() {
	gob.Register(Namespace{})
}

type Namespace struct {
	Name      string
	locations map[string]*Location
	tree      *QuadTree
	mu        sync.RWMutex
}

func NewNamespace(name string) *Namespace {
	return &Namespace{
		Name:      name,
		locations: map[string]*Location{},
		tree:      NewQuadTree(-90, 90, -180, 180),
		mu:        sync.RWMutex{},
	}
}

func (n *Namespace) SaveLocation(id string, lat float64, lon float64) (*Location, error) {

	loc, err := NewLocation(n.Name, id, lat, lon)
	if err != nil {
		return nil, err
	}

	n.mu.Lock()
	n.locations[id] = loc
	n.mu.Unlock()

	err = n.tree.Insert(loc)
	if err != nil {
		return nil, err
	}

	return loc, nil
}

func (n *Namespace) DeleteLocation(id string) {
	n.mu.Lock()
	delete(n.locations, id)
	n.mu.Unlock()

	n.tree.Root.Delete(id)
}

func (n *Namespace) GetLocation(id string) (*Location, bool) {
	n.mu.RLock()
	defer n.mu.RUnlock()
	loc, ok := n.locations[id]

	return loc, ok
}

func (n *Namespace) QueryRange(lat1, lat2, lon1, lon2 float64) []*Location {
	return n.tree.Root.QueryRange(lat1, lat2, lon1, lon2)
}
