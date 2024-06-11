package world

import (
	"bytes"
	"encoding/gob"
	"github.com/uber/h3-go"
	"log"
	"sync"
)

const (
	EdgeLevelZeroKm     = 1281.256011
	EdgeLevelOneKm      = 483.0568391
	EdgeLevelTwoKm      = 182.5129565
	EdgeLevelThreeKm    = 68.97922179
	EdgeLevelFourKm     = 26.07175968
	EdgeLevelFiveKm     = 9.85409099
	EdgeLevelSixKm      = 3.724532667
	EdgeLevelSevenKm    = 1.406475763
	EdgeLevelEightKm    = 0.53141401
	EdgeLevelNineKm     = 0.200786148
	EdgeLevelTenKm      = 0.075863783
	EdgeLevelElevenKm   = 0.028591176
	EdgeLevelTwelveKm   = 0.010830188
	EdgeLevelThirteenKm = 0.00409201
	EdgeLevelFourteenKm = 0.0015461
	EdgeLevelFifteenKm  = 0.000584169
)

var (
	EdgeLevels = []float64{EdgeLevelZeroKm, EdgeLevelOneKm, EdgeLevelTwoKm, EdgeLevelThreeKm, EdgeLevelFourKm, EdgeLevelFiveKm, EdgeLevelSixKm, EdgeLevelSevenKm, EdgeLevelEightKm, EdgeLevelNineKm, EdgeLevelTenKm, EdgeLevelElevenKm, EdgeLevelTwelveKm, EdgeLevelThirteenKm, EdgeLevelFourteenKm, EdgeLevelFifteenKm}
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

func (m *World) GetLocationsInRadius(ns string, lat float64, lon float64, radiusInMeters float64) []map[string]*Location {
	namespace := m.getNamespace(ns)

	if namespace == nil {
		return []map[string]*Location{}
	}
	locationMaps := m.getGridsInRadius(ns, lat, lon, radiusInMeters)

	return locationMaps
}

func (m *World) getGridsInRadius(ns string, lat float64, lon float64, radiusInMeters float64) []map[string]*Location {
	level, tooBig := m.getLevelForLocation(radiusInMeters)
	var locations []map[string]*Location

	index := h3.FromGeo(h3.GeoCoord{Latitude: lat, Longitude: lon}, int(level.Level))
	k := 1

	if tooBig {
		k = int(radiusInMeters / EdgeLevels[0])
	}
	indices := h3.KRing(index, k)

	for i := 0; i < len(indices); i++ {
		grid, ok := level.Grids.Load(h3.ToString(indices[i]))

		if !ok {
			continue
		}

		locations = append(locations, grid.(*Grid).GetLocations(ns))
	}

	return locations
}

func (m *World) getLevelForLocation(radiusInMeters float64) (*Level, bool) {
	for i := len(EdgeLevels) - 1; i >= 0; i-- {
		if radiusInMeters < EdgeLevels[i] {
			level, ok := m.levels.Load(int8(i))

			if !ok {
				panic("Level not found")
			}

			return level.(*Level), false
		}
	}

	level, ok := m.levels.Load(int8(0))

	if !ok {
		panic("Level not found")
	}

	return level.(*Level), true
}
