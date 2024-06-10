package world

import (
	"encoding/gob"
	"errors"
	"github.com/uber/h3-go"
	"log"
)

var (
	LevelErrorInvalidLevel = errors.New("level must be greater than 0 and less than 16")
)

func init() {
	gob.Register(Level{})
}

type Level struct {
	Level int8
	Grids map[string]*Grid

	index map[string]string
}

func NewLevel(level int8) (*Level, error) {
	if level < 0 || level > 15 {
		return nil, LevelErrorInvalidLevel
	}

	return &Level{
		Level: level,
		Grids: make(map[string]*Grid),
		index: make(map[string]string),
	}, nil
}

func (l *Level) PlaceLocation(loc *Location) error {
	if loc == nil {
		return LocationErrorRequiredId
	}

	gridKey, ok := l.index[loc.Id]
	var currentGrid *Grid

	if ok {
		currentGrid = l.Grids[gridKey]
	}

	grid := l.getGrid(loc)

	if currentGrid != nil && currentGrid.Name != grid.Name {
		currentGrid.DeleteLocation(loc)
		delete(l.index, loc.Id)
	}

	grid.AddLocation(loc)
	l.index[loc.Id] = grid.Name

	return nil
}

func (l *Level) getGrid(loc *Location) *Grid {
	geo := h3.GeoCoord{
		Latitude:  loc.Lat,
		Longitude: loc.Lon,
	}

	geoHash := h3.FromGeo(geo, int(l.Level))

	geoHashString := h3.ToString(geoHash)

	grid, ok := l.Grids[geoHashString]

	if !ok {
		newGrid, err := NewGrid(geoHashString)

		if err != nil {
			log.Fatal(err)
		}

		l.Grids[geoHashString] = newGrid

		grid = newGrid
	}

	return grid
}

func (l *Level) DeleteLocation(loc *Location) {
	grid := l.getGrid(loc)

	grid.DeleteLocation(loc)
}
