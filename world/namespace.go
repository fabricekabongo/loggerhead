package world

import "sync"

type Namespace struct {
	Name      string
	locations map[string]*Location
	mu        sync.RWMutex
}

func NewNamespace(name string) *Namespace {
	return &Namespace{
		Name:      name,
		locations: make(map[string]*Location),
	}
}

func (n *Namespace) SaveLocation(id string, lat float64, lon float64) (*Location, error) {
	n.mu.Lock()
	defer n.mu.Unlock()

	currentLoc, ok := n.locations[id]
	if ok {
		currentLoc.Lat = lat
		currentLoc.Lon = lon
		return currentLoc, nil
	}

	loc, err := NewLocation(n.Name, id, lat, lon)
	if err != nil {
		return nil, err
	}

	n.locations[id] = loc

	return loc, nil
}

func (n *Namespace) DeleteLocation(id string) {
	n.mu.Lock()
	defer n.mu.Unlock()

	delete(n.locations, id)
}
