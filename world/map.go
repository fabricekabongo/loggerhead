package world

import (
	"errors"
	"github.com/uber/h3-go"
	"log"
	"sync"
)

var (
	ErrLocIdRequired = errors.New("location id is required")
)

type LocationEntity struct {
	LocId string  `json:"loc_id"`
	Lat   float64 `json:"lat"`
	Lon   float64 `json:"lon"`
}

type Stats struct {
	Locations int
	Grids     int
}

type Map struct {
	Locations map[string]*LocationEntity `json:"locations"`
	Grids     map[string]*Grid           `json:"grids"`
	Mu        sync.RWMutex               // to protect the location
	Stat      Stats
}

func NewMap() *Map {

	return &Map{
		Locations: make(map[string]*LocationEntity),
		Grids:     make(map[string]*Grid),
		Mu:        sync.RWMutex{},
		Stat:      Stats{},
	}
}

func (m *Map) Stats() Stats {
	m.Mu.RLock()
	defer m.Mu.RUnlock()

	return m.Stat
}

func (m *Map) Save(locId string, lat float64, lon float64) error {
	m.Mu.Lock()
	defer m.Mu.Unlock()

	if len(locId) == 0 {
		return ErrLocIdRequired
	}

	currLoc, ok := m.Locations[locId]
	if !ok {
		m.createLocation(locId, lat, lon)
		return nil
	}

	m.assignToGrid(lat, lon, currLoc)

	currLoc.Lat = lat
	currLoc.Lon = lon

	return nil
}

func (m *Map) assignToGrid(lat float64, lon float64, currLoc *LocationEntity) {
	currentGrid := m.getGrid(currLoc.Lat, currLoc.Lon)
	newGrid := m.getGrid(lat, lon)

	if currentGrid.Name != newGrid.Name {
		currentGrid.DeleteLocation(currLoc)
		newGrid.AddLocation(currLoc)
	}
}

func (m *Map) createLocation(locId string, lat float64, lon float64) {
	if len(locId) == 0 {
		log.Fatal("location id is required. It should have never reached this point")
	}

	_, exists := m.Locations[locId]
	if exists {
		log.Fatal("Location already exists. It should have never reached this point")
	}

	loc := &LocationEntity{
		LocId: locId,
		Lat:   lat,
		Lon:   lon,
	}

	m.Locations[locId] = loc

	grid := m.getGrid(lat, lon)
	grid.AddLocation(loc)
	m.Stat.Locations++
}

func (m *Map) getGrid(lat float64, lon float64) *Grid {
	geo := h3.GeoCoord{
		Latitude:  lat,
		Longitude: lon,
	}

	geoHash := h3.FromGeo(geo, 4)
	geoHashString := h3.ToString(geoHash)

	grid, ok := m.Grids[geoHashString]

	if !ok {
		grid = NewGrid(geoHashString)
		m.Grids[geoHashString] = grid
		m.Stat.Grids++
	}

	return grid
}
