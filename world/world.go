package world

import (
	"bytes"
	"encoding/gob"
	"errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"log"
	"sync"
)

var (
	NamespaceErrorNotFound = errors.New("namespace not found")
	SaveCounter            = promauto.NewCounter(prometheus.CounterOpts{
		Name: "loggerhead_world_save",
		Help: "Total locations saved",
	})
	DeleteCounter = promauto.NewCounter(prometheus.CounterOpts{
		Name: "loggerhead_world_delete",
		Help: "Total locations deleted",
	})
	QueryCounter = promauto.NewCounter(prometheus.CounterOpts{
		Name: "loggerhead_world_query",
		Help: "Total locations queried",
	})
	GetCounter = promauto.NewCounter(prometheus.CounterOpts{
		Name: "loggerhead_world_get",
		Help: "Total locations retrieved",
	})
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

func (m *World) Delete(ns string, locId string) {
	defer DeleteCounter.Inc()
	namespace := m.getNamespace(ns)

	if namespace == nil {
		return
	}

	namespace.DeleteLocation(locId)
}

// Save a location to the world. If the location already exists, it will be updated.
func (m *World) Save(ns string, locId string, lat float64, lon float64) error {
	defer SaveCounter.Inc()
	namespace := m.getNamespace(ns)

	if namespace == nil {
		return NamespaceErrorNotFound
	}

	_, err := namespace.SaveLocation(locId, lat, lon)

	return err
}

func (m *World) getNamespace(ns string) *Namespace {
	m.mu.RLock()
	namespace, ok := m.namespaces[ns]
	m.mu.RUnlock()

	if !ok {
		namespace = NewNamespace(ns)
		m.mu.Lock()
		m.namespaces[ns] = namespace
		m.mu.Unlock()
	}

	return namespace
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
	w.mu.Lock()
	defer w.mu.Unlock()
	for ns, n := range w.namespaces {
		for locId, loc := range n.locations {
			err := m.Save(ns, locId, loc.Lat(), loc.Lon())
			if err != nil {
				log.Fatal("Error merging world: ", err)
			}
		}
	}
}

func (m *World) GetLocation(ns string, id string) (Location, bool) {
	defer GetCounter.Inc()
	namespace := m.getNamespace(ns)

	if namespace == nil {
		return Location{}, false
	}

	location, ok := namespace.GetLocation(id)
	if !ok {
		return Location{}, false
	}

	return *location, true
}

func (m *World) QueryRange(ns string, lat1, lat2, lon1, lon2 float64) []*Location {
	defer QueryCounter.Inc()
	namespace := m.getNamespace(ns)

	if namespace == nil {
		return []*Location{}
	}

	return namespace.QueryRange(lat1, lat2, lon1, lon2)
}
