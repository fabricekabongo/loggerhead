package world

import (
	"bytes"
	"encoding/gob"
	"log"
	"sync"
)

type Stats struct {
	Locations int
	Grids     int
}

type World struct {
	levels     *sync.Map
	namespaces *sync.Map
}

func init() {
	gob.Register(World{})
	gob.Register(Stats{})
}

func NewWorld() *World {
	var levels = &sync.Map{}
	for i := int8(0); i < 16; i++ {
		level, err := NewLevel(i)
		if err != nil {
			log.Fatalf("Error creating level: %v", err)
		}

		levels.Store(i, level)
	}

	return &World{
		levels:     levels,
		namespaces: &sync.Map{},
	}
}

func (m *World) Save(ns string, locId string, lat float64, lon float64) error {
	namespace := m.getNamespace(ns)

	if namespace == nil {
		panic("Namespace not found")
	}

	location, ok := namespace.locations[locId]

	if !ok {
		saveLocation, err := namespace.SaveLocation(locId, lat, lon)
		if err != nil {
			return err
		}
		location = saveLocation
	} else {
		location.Lat = lat
		location.Lon = lon
	}
	var topErr error

	m.levels.Range(func(key, value interface{}) bool {
		level := value.(*Level)
		err := level.PlaceLocation(location)
		if err != nil {
			topErr = err
			return false
		}
		return true
	})

	return topErr
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

	w.levels.Range(func(key, level interface{}) bool {
		m.levels.Store(key, level)
		return true
	})
}

func (m *World) GetLocation(ns string, id string) (Location, bool) {
	namespace := m.getNamespace(ns)

	if namespace == nil {
		return Location{}, false
	}

	location, ok := namespace.locations[id]

	if !ok {
		return Location{}, false
	}

	return *location, true
}
