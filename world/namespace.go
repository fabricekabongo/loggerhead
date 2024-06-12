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
	locations sync.Map
	tree      *QuadTree
}

func NewNamespace(name string) *Namespace {
	return &Namespace{
		Name:      name,
		locations: sync.Map{},
		tree:      NewQuadTree(-90, 90, -180, 180),
	}
}

func (n *Namespace) SaveLocation(id string, lat float64, lon float64) (*Location, error) {

	loc, err := NewLocation(n.Name, id, lat, lon)
	if err != nil {
		return nil, err
	}

	n.locations.Store(id, loc)
	n.tree.Insert(loc)

	return loc, nil
}

func (n *Namespace) DeleteLocation(id string) {
	n.locations.Delete(id)
}

func (n *Namespace) GetLocation(id string) (*Location, bool) {
	loc, found := n.locations.Load(id)
	if !found {
		return nil, false
	}
	return loc.(*Location), found
}

func (n *Namespace) QueryRange(lat1, lat2, lon1, lon2 float64) []*Location {
	return n.tree.Root.QueryRange(lat1, lat2, lon1, lon2)
}
