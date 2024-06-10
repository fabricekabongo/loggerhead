package world

import (
	"encoding/gob"
	"errors"
	"github.com/uber/h3-go"
	"log"
	"sync"
)

var (
	LevelErrorInvalidLevel = errors.New("level must be greater than 0 and less than 16")
)

func init() {
	gob.Register(Level{})
}

type Level struct {
	Level int8
	Grids sync.Map
	index sync.Map
}

func NewLevel(level int8) (*Level, error) {
	if level < 0 || level > 15 {
		return nil, LevelErrorInvalidLevel
	}

	return &Level{
		Level: level,
		Grids: sync.Map{},
		index: sync.Map{},
	}, nil
}

func (l *Level) PlaceLocation(loc *Location) error {
	if loc == nil {
		return LocationErrorRequiredId
	}

	iVal, ok := l.index.Load(loc.Id)
	var currentGrid *Grid

	if ok {
		key := iVal.(string)
		gVal, found := l.Grids.Load(key)
		if found {
			currentGrid = gVal.(*Grid)
		}
	}

	grid := l.getGrid(loc)

	if currentGrid != nil && currentGrid.Name != grid.Name {
		currentGrid.DeleteLocation(loc)
		l.index.Delete(loc.Id)
	}

	grid.AddLocation(loc)
	l.index.Store(loc.Id, grid.Name)

	return nil
}

func (l *Level) getGrid(loc *Location) *Grid {
	geo := h3.GeoCoord{
		Latitude:  loc.Lat,
		Longitude: loc.Lon,
	}

	geoHash := h3.FromGeo(geo, int(l.Level))

	geoHashString := h3.ToString(geoHash)

	grid, ok := l.Grids.Load(geoHashString)

	if !ok {
		newGrid, err := NewGrid(geoHashString)

		if err != nil {
			log.Fatal(err)
		}

		l.Grids.Store(geoHashString, newGrid)

		grid = newGrid
	}

	return grid.(*Grid)
}

func (l *Level) DeleteLocation(loc *Location) {
	grid := l.getGrid(loc)

	grid.DeleteLocation(loc)
}
