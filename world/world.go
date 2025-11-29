package world

import (
	"bytes"
	"encoding/gob"
	"errors"
	"sync"
)

var (
	ErrUnexpectedNilNamespace = errors.New("failed to create namespace")
)

type Stats struct {
	Locations int
	Grids     int
}

type World struct {
	namespaces map[string]*Namespace
	mu         sync.RWMutex
}

func init() {
	gob.Register(World{})
	gob.Register(Stats{})
}

func NewWorld() *World {
	return &World{
		namespaces: map[string]*Namespace{},
		mu:         sync.RWMutex{},
	}
}

func (m *World) Delete(ns, locId string) {
	namespace := m.getNamespace(ns)
	namespace.DeleteLocation(locId)
}

// Save a location to the world. If the location already exists, it will be updated.
func (m *World) Save(ns, locId string, lat, lon float64) error {
	namespace := m.getNamespace(ns)

	_, err := namespace.SaveLocation(locId, lat, lon)

	return err
}

func (m *World) getNamespace(ns string) *Namespace {
	m.mu.Lock()

	namespace, ok := m.namespaces[ns]

	if !ok {
		namespace = NewNamespace(ns)
		m.namespaces[ns] = namespace
	}

	if namespace == nil {
		m.mu.Unlock()
		panic(ErrUnexpectedNilNamespace)
	}

	m.mu.Unlock()

	return namespace
}

func (m *World) ToBytes() []byte {
	type serialLocation struct {
		ID  string
		Lat float64
		Lon float64
	}

	type serialNamespace struct {
		Name      string
		Locations []serialLocation
	}

	type serialWorld struct {
		Namespaces []serialNamespace
	}

	m.mu.RLock()
	dto := serialWorld{}

	for _, namespace := range m.namespaces {
		namespace.mu.RLock()
		nsDTO := serialNamespace{Name: namespace.Name}
		for id, loc := range namespace.locations {
			nsDTO.Locations = append(nsDTO.Locations, serialLocation{ID: id, Lat: loc.Lat(), Lon: loc.Lon()})
		}
		namespace.mu.RUnlock()

		dto.Namespaces = append(dto.Namespaces, nsDTO)
	}
	m.mu.RUnlock()

	var buf bytes.Buffer

	_ = gob.NewEncoder(&buf).Encode(dto)

	return buf.Bytes()
}

func NewWorldFromBytes(buf []byte) *World {
	type serialLocation struct {
		ID  string
		Lat float64
		Lon float64
	}

	type serialNamespace struct {
		Name      string
		Locations []serialLocation
	}

	type serialWorld struct {
		Namespaces []serialNamespace
	}

	var dto serialWorld

	dec := gob.NewDecoder(bytes.NewReader(buf))
	err := dec.Decode(&dto)

	if err != nil {
		panic(err)
	}

	world := NewWorld()

	for _, namespace := range dto.Namespaces {
		for _, loc := range namespace.Locations {
			err := world.Save(namespace.Name, loc.ID, loc.Lat, loc.Lon)
			if err != nil {
				panic(err)
			}
		}
	}

	return world
}

func (m *World) Merge(w *World) {
	w.mu.Lock()
	defer w.mu.Unlock()

	for ns, n := range w.namespaces {
		for locId, loc := range n.locations {
			err := m.Save(ns, locId, loc.Lat(), loc.Lon())
			if err != nil {
				panic(err)
			}
		}
	}
}

func (m *World) GetLocation(ns, id string) (Location, bool) {
	namespace := m.getNamespace(ns)

	location, ok := namespace.GetLocation(id)
	if !ok {
		return Location{}, false
	}

	return *location, true
}

func (m *World) QueryRange(ns string, lat1, lat2, lon1, lon2 float64) []*Location {
	namespace := m.getNamespace(ns)

	return namespace.QueryRange(lat1, lat2, lon1, lon2)
}
