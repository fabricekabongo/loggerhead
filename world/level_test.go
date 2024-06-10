package world

import (
	"errors"
	"github.com/uber/h3-go"
	"math/rand"
	"testing"
)

func TestLevel(t *testing.T) {
	t.Parallel()

	t.Run("NewLevel", func(t *testing.T) {
		t.Run("It should return a Level", func(t *testing.T) {
			t.Parallel()

			l, err := NewLevel(5)

			if err != nil {
				t.Errorf("Expected no error, got %v", err)
			}
			if l == nil {
				t.Error("NewLevel should return a Level")
			}

			if l.Level != 5 {
				t.Errorf("Expected name to be test, got %d", l.Level)
			}

			if l.Grids == nil {
				t.Error("Grids should be initialized")
			}
		})

		t.Run("It should fail if level is lower than 0", func(t *testing.T) {
			t.Parallel()

			l, err := NewLevel(-1)

			if l != nil {
				t.Error("Level should not be created")
			}

			if !errors.Is(err, LevelErrorInvalidLevel) {
				t.Errorf("expected error %v, got %v", LevelErrorInvalidLevel, err)
			}
		})

		t.Run("It should fail if level is greater than 15", func(t *testing.T) {
			t.Parallel()

			l, err := NewLevel(16)

			if l != nil {
				t.Error("Level should not be created")
			}

			if !errors.Is(err, LevelErrorInvalidLevel) {
				t.Errorf("expected error %v, got %v", LevelErrorInvalidLevel, err)
			}
		})
	})

	t.Run("PlaceLocation", func(t *testing.T) {
		t.Run("It should place a location in the grid", func(t *testing.T) {
			t.Parallel()
			level := rand.Intn(15)
			l, _ := NewLevel(int8(level))

			loc, err, ns, id, lat, lon := CreateRandomTestLocation()
			geo := h3.GeoCoord{
				Latitude:  lat,
				Longitude: lon,
			}

			geoHash := h3.FromGeo(geo, level)
			geoHashString := h3.ToString(geoHash)

			if err != nil {
				t.Errorf("Expected no error, got %v", err)
			}

			err = l.PlaceLocation(loc)
			if err != nil {
				t.Errorf("Expected no error, got %v", err)
			}

			if len(l.Grids) != 1 {
				t.Errorf("Expected grids to have 1 element, got %d", len(l.Grids))
			}

			grid := l.Grids[geoHashString]
			if grid == nil {
				t.Error("Grid should not be nil")
			}

			locations := grid.GetLocations(ns)
			location, _ := locations[id]
			if location != loc {
				t.Errorf("The location should be the same. Expect %v got %v instead", loc, location)
			}
		})
	})

	t.Run("DeleteLocation", func(t *testing.T) {
		t.Run("It should delete a location from the grid", func(t *testing.T) {
			t.Parallel()
			level := rand.Intn(15)
			l, _ := NewLevel(int8(level))

			loc, err, ns, _, lat, lon := CreateRandomTestLocation()
			geo := h3.GeoCoord{
				Latitude:  lat,
				Longitude: lon,
			}

			geoHash := h3.FromGeo(geo, level)
			geoHashString := h3.ToString(geoHash)

			if err != nil {
				t.Errorf("Expected no error, got %v", err)
			}

			err = l.PlaceLocation(loc)

			grid := l.Grids[geoHashString]

			l.DeleteLocation(loc)

			locations := grid.GetLocations(ns)
			if len(locations) != 0 {
				t.Errorf("The locations to be empty. Expect 0 got %v instead", len(locations))
			}
		})

		t.Run("It should not fail if the location is not in the grid", func(t *testing.T) {
			t.Parallel()
			level := rand.Intn(15)
			l, _ := NewLevel(int8(level))

			loc, err, _, _, lat, lon := CreateRandomTestLocation()
			if err != nil {
				t.Errorf("Expected no error, got %v", err)
			}

			geo := h3.GeoCoord{
				Latitude:  lat,
				Longitude: lon,
			}

			geoHash := h3.FromGeo(geo, level)
			geoHashString := h3.ToString(geoHash)

			grid := l.Grids[geoHashString]
			if grid != nil {
				t.Errorf("No grid should have even been created")
			}

			l.DeleteLocation(loc)
		})
	})
}
