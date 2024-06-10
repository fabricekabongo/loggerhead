package world

import (
	"encoding/gob"
	"errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"sync"
)

var (
	metricGauge = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "geo_db_grid_locations_total",
		Help: "The total number of locations in the grid",
	}, []string{"grid_name"})
	opsProcessed = promauto.NewCounter(prometheus.CounterOpts{
		Name: "geo_db_grid_processed_ops_total",
		Help: "The total number of processed operations by the grid",
	})
	ErrorGridNameRequired = errors.New("grid name is required")
)

func init() {
	gob.Register(Grid{})
	gob.Register(LocationUpdateEvent{})
	gob.Register(LocationAddedEvent{})
	gob.Register(LocationDeletedEvent{})
}

type LocationUpdateEvent struct {
	LocId   string  `json:"loc_id"`
	Lat     float64 `json:"lat"`
	Lon     float64 `json:"lon"`
	PrevLat float64 `json:"prev_lat"`
	PrevLon float64 `json:"prev_lon"`
}

type LocationAddedEvent struct {
	LocId string  `json:"loc_id"`
	Lat   float64 `json:"lat"`
	Lon   float64 `json:"lon"`
}

type LocationDeletedEvent struct {
	LocId string `json:"loc_id"`
}

type Grid struct {
	Name                   string
	namespaces             map[string]map[string]*Location
	AddEventSubscribers    map[string]chan LocationAddedEvent
	UpdateEventSubscribers map[string]chan LocationUpdateEvent
	DeleteEventSubscribers map[string]chan LocationDeletedEvent
	index                  map[string]string
	mu                     sync.RWMutex
}

func NewGrid(name string) (*Grid, error) {
	if len(name) == 0 {
		return nil, ErrorGridNameRequired
	}
	return &Grid{
		Name:                   name,
		namespaces:             make(map[string]map[string]*Location),
		AddEventSubscribers:    make(map[string]chan LocationAddedEvent),
		UpdateEventSubscribers: make(map[string]chan LocationUpdateEvent),
		DeleteEventSubscribers: make(map[string]chan LocationDeletedEvent),
		mu:                     sync.RWMutex{},
	}, nil
}

func (g *Grid) DeleteLocation(loc *Location) {
	g.mu.Lock()
	defer g.mu.Unlock()
	defer metricGauge.WithLabelValues(g.Name).Dec()
	defer opsProcessed.Inc()

	namespace, _ := g.namespaces[loc.Ns]

	delete(namespace, loc.Id)

	//go func() {
	//	for _, subscriber := range g.DeleteEventSubscribers {
	//		subscriber <- LocationDeletedEvent{LocId: loc.Id}
	//	}
	//}()
}

func (g *Grid) UpdateLocation(ns string, loc *Location, lat float64, lon float64) error {
	g.mu.Lock()
	defer metricGauge.WithLabelValues(g.Name).Inc()
	defer opsProcessed.Inc()

	if len(loc.Id) == 0 {
		panic("locationId is required. It should have never reached the grid")
	}

	if len(ns) == 0 {
		panic("namespace is required. It should have never reached the grid")
	}

	//_, ok := g.locations[loc.Id]
	//if !ok {
	//	panic("Location is not in the grid. Update should have never reached the grid")
	//}
	//prevLat := g.locations[loc.Id].Lat
	//prevLon := g.locations[loc.Id].Lon
	//
	//g.locations[loc.Id].Lat = lat
	//g.locations[loc.Id].Lon = lon
	//
	//go func() {
	//	for _, subscriber := range g.UpdateEventSubscribers {
	//		subscriber <- LocationUpdateEvent{
	//			Id:   loc.Id,
	//			Lat:     lat,
	//			Lon:     lon,
	//			PrevLat: prevLat,
	//			PrevLon: prevLon,
	//		}
	//	}
	//}()

	return nil
}

func (g *Grid) AddLocation(loc *Location) {
	g.mu.Lock()
	defer g.mu.Unlock()
	defer opsProcessed.Inc()
	defer metricGauge.WithLabelValues(g.Name).Inc()
	namespace, ok := g.namespaces[loc.Ns]

	if ok {
		location, exists := namespace[loc.Id]
		if exists {
			location.Lat = loc.Lat
			location.Lon = loc.Lon
		} else {
			namespace[loc.Id] = loc
		}
	} else {
		g.namespaces[loc.Ns] = make(map[string]*Location)
		g.namespaces[loc.Ns][loc.Id] = loc
	}

	//go func() {
	//	for _, subscriber := range g.AddEventSubscribers {
	//		subscriber <- LocationAddedEvent{
	//			Id: loc.Id,
	//			Lat:   loc.Lat,
	//			Lon:   loc.Lon,
	//		}
	//	}
	//}()
}

func (g *Grid) GetLocations(ns string) map[string]*Location {
	g.mu.RLock()
	defer g.mu.RUnlock()

	namespace, ok := g.namespaces[ns]
	if !ok {
		return map[string]*Location{}
	}

	return namespace
}
