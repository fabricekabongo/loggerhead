package world

import (
	"bytes"
	"encoding/gob"
	"errors"
	"log"
	"sync"
)

var (
	NamespaceErrorNotFound = errors.New("namespace not found")
)

type Stats struct {
	Locations int
	Grids     int
}

type World struct {
	namespaces *sync.Map
}

func init() {
	gob.Register(World{})
	gob.Register(Stats{})
}

func NewWorld() *World {
	return &World{
		namespaces: &sync.Map{},
	}
}

func (m *World) Save(ns string, locId string, lat float64, lon float64) error {
	namespace := m.getNamespace(ns)

	if namespace == nil {
		return NamespaceErrorNotFound
	}

	location, err := NewLocation(ns, locId, lat, lon)

	if err != nil {
		return err
	}

	namespace.locations.Store(locId, location)
	namespace.tree.Insert(location)

	return nil
}

func (m *World) getNamespace(ns string) *Namespace {
	_, ok := m.namespaces.Load(ns)

	if !ok {
		newNamespace := NewNamespace(ns)

		m.namespaces.Store(ns, newNamespace)
	}

	namespace, ok := m.namespaces.Load(ns)

	if !ok {
		panic("sync.Map is not saving data")
	}

	return namespace.(*Namespace)
}

func (m *World) ToBytes() []byte {
	var buf bytes.Buffer

	enc := gob.NewEncoder(&buf)
	err := enc.Encode(m)

	if err != nil {

		return []byte{}
	}

	return buf.Bytes()
}

func NewWorldFromBytes(buf []byte) *World {
	var w World

	dec := gob.NewDecoder(bytes.NewReader(buf))
	err := dec.Decode(&w)

	if err != nil {
		log.Fatal(err)
	}

	return &w
}

func (m *World) Merge(w *World) {
	w.namespaces.Range(func(key, ns interface{}) bool {
		m.namespaces.Store(key, ns)
		return true

	})
}

func (m *World) GetLocation(ns string, id string) (Location, bool) {
	namespace := m.getNamespace(ns)

	if namespace == nil {
		return Location{}, false
	}

	entry, ok := namespace.locations.Load(id)
	location, ok := entry.(*Location)

	if !ok {
		return Location{}, false
	}

	return *location, true
}

func (m *World) QueryRange(ns string, lat1, lat2, lon1, lon2 float64) []*Location {
	namespace := m.getNamespace(ns)

	if namespace == nil {
		return []*Location{}
	}

	return namespace.QueryRange(lat1, lat2, lon1, lon2)
}
