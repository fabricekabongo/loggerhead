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
	levels     map[int8]*Level
	namespaces map[string]*Namespace
	mu         sync.RWMutex // to protect the location
	stat       Stats
}

func init() {
	gob.Register(World{})
	gob.Register(Level{})
	gob.Register(Grid{})
	gob.Register(Location{})
	gob.Register(Namespace{})

}

func NewWorld() *World {
	var levels = make(map[int8]*Level, 16)
	for i := int8(0); i < 16; i++ {
		level, err := NewLevel(i)
		if err != nil {
			log.Fatalf("Error creating level: %v", err)
		}

		levels[i] = level
	}

	return &World{
		levels:     levels,
		namespaces: make(map[string]*Namespace),
		mu:         sync.RWMutex{},
		stat:       Stats{},
	}
}

func (m *World) Stats() Stats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.stat
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

	for _, level := range m.levels {
		err := level.PlaceLocation(location)
		if err != nil {
			return err
		}
	}

	return nil
}

func (m *World) getNamespace(ns string) *Namespace {
	m.mu.Lock()
	defer m.mu.Unlock()

	_, ok := m.namespaces[ns]

	if !ok {
		m.namespaces[ns] = NewNamespace(ns)
	}

	return m.namespaces[ns]
}

func (m *World) ToBytes() []byte {
	m.mu.RLock()
	defer m.mu.RUnlock()

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
	m.mu.Lock()
	defer m.mu.Unlock()

	for ns, n := range w.namespaces {
		m.namespaces[ns] = n
	}

	for l, level := range w.levels {
		m.levels[l] = level
	}
}
