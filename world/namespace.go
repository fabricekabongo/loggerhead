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
	n.mu.RLock()
	loc, ok := n.locations[id]
	n.mu.RUnlock()

	if ok {
		err := loc.Update(lat, lon)
		if err != nil {
			return nil, err
		}
	} else {
		newLoc, err := NewLocation(n.Name, id, lat, lon)
		if err != nil {
			return nil, err
		}
		loc = newLoc

		n.mu.Lock()
		n.locations[id] = loc
		n.mu.Unlock()
	}

	err := n.tree.Insert(loc)
	if err != nil {
		return nil, err
	}

	return loc, nil
}

func (n *Namespace) DeleteLocation(id string) {
	n.mu.RLock()
	loc, ok := n.locations[id]
	n.mu.RUnlock()

	if !ok {
		return
	}

	if loc.Node != nil {
		loc.Node.Delete(loc.Id())
	}

	n.mu.Lock()
	delete(n.locations, id)
	n.mu.Unlock()
}

func (n *Namespace) GetLocation(id string) (*Location, bool) {
	n.mu.RLock()
	loc, ok := n.locations[id]
	n.mu.RUnlock()

	return loc, ok
}

func (n *Namespace) QueryRange(lat1, lat2, lon1, lon2 float64) []*Location {
	return n.tree.Root.QueryRange(lat1, lat2, lon1, lon2)
}
