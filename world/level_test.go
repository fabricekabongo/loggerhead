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

			v, ok := l.Grids.Load(geoHashString)
			grid := v.(*Grid)
			if !ok {
				t.Error("Grid should not be nil")
			}

			locations := grid.GetLocations(ns)
			location, _ := locations[id]
			if location != loc {
				t.Errorf("The location should be the same. Expect %v got %v instead", loc, location)
			}
		})
		t.Run("It should update the location in the same grid", func(t *testing.T) {
			t.Parallel()
			level := rand.Intn(15)
			l, _ := NewLevel(int8(level))

			loc1, err, _, _, _, _ := CreateRandomTestLocation()
			if err != nil {
				t.Errorf("Expected no error, got %v", err)
			}
			loc2, err := NewLocation(loc1.Ns, loc1.Id, loc1.Lat+0.00000000000000001, loc1.Lon+0.00000000000000001) //Very small change to accommodate for the smallest possible grid
			if err != nil {
				t.Errorf("Expected no error, got %v", err)
			}

			err = l.PlaceLocation(loc1)
			if err != nil {
				t.Errorf("Expected no error, got %v", err)
			}

			geo := h3.GeoCoord{
				Latitude:  loc1.Lat,
				Longitude: loc1.Lon,
			}

			geoHash := h3.FromGeo(geo, level)
			geoHashString := h3.ToString(geoHash)

			v, _ := l.Grids.Load(geoHashString)
			grid := v.(*Grid)
			locations := grid.GetLocations(loc1.Ns)
			location, _ := locations[loc1.Id]
			if location != loc1 {
				t.Errorf("The location should be the same. Expect %v got %v instead", loc1, location)
			}

			err = l.PlaceLocation(loc2)
			if err != nil {
				t.Errorf("Expected no error, got %v", err)
			}

			locations = grid.GetLocations(loc1.Ns)
			location, _ = locations[loc1.Id]
			if location == loc2 {
				t.Errorf("It should have updated the original location")
			}

			if location.Lat != loc2.Lat || location.Lon != loc2.Lon || location.Ns != loc2.Ns || location.Id != loc2.Id {
				t.Errorf("The location's information should be the same. Expect Lat: exp %v instead %v, Lon: exp %v instead %v, Id: exp %v instead %v, NS: exp %v instead %v", loc2.Lat, location.Lat, loc2.Lon, location.Lon, loc2.Id, location.Id, loc2.Ns, location.Ns)
			}
		})
		t.Run("It should update the location and move to another grid when provided a new object", func(t *testing.T) {
			t.Parallel()
			level := 15
			l, _ := NewLevel(int8(level))

			loc1, err, _, _, _, _ := CreateRandomTestLocation()
			if err != nil {
				t.Errorf("Expected no error, got %v", err)
			}
			loc2, err := NewLocation(loc1.Ns, loc1.Id, loc1.Lat+1, loc1.Lon+1) //Very big change to force moving to another grid
			if err != nil {
				t.Errorf("Expected no error, got %v", err)
			}

			err = l.PlaceLocation(loc1)
			if err != nil {
				t.Errorf("Expected no error, got %v", err)
			}
			err = l.PlaceLocation(loc2)
			if err != nil {
				t.Errorf("Expected no error, got %v", err)
			}

			geo := h3.GeoCoord{
				Latitude:  loc1.Lat,
				Longitude: loc1.Lon,
			}

			geoHash := h3.FromGeo(geo, level)
			geoHashString := h3.ToString(geoHash)

			v, _ := l.Grids.Load(geoHashString)
			grid := v.(*Grid)
			locations := grid.GetLocations(loc1.Ns)
			location, _ := locations[loc1.Id]
			if location == loc1 {
				t.Errorf("The location shouldn't be in the initial grid")
			}

			geo = h3.GeoCoord{
				Latitude:  loc2.Lat,
				Longitude: loc2.Lon,
			}

			geoHash = h3.FromGeo(geo, level)
			geoHashString = h3.ToString(geoHash)

			v, _ = l.Grids.Load(geoHashString)
			grid2 := v.(*Grid)

			locations = grid2.GetLocations(loc1.Ns)
			location, _ = locations[loc1.Id]

			if location.Lat != loc2.Lat || location.Lon != loc2.Lon || location.Ns != loc2.Ns || location.Id != loc2.Id {
				t.Errorf("The location's information should be the same. Expect Lat: exp %v instead %v, Lon: exp %v instead %v, Id: exp %v instead %v, NS: exp %v instead %v", loc2.Lat, location.Lat, loc2.Lon, location.Lon, loc2.Id, location.Id, loc2.Ns, location.Ns)
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

			v, _ := l.Grids.Load(geoHashString)
			grid := v.(*Grid)

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

			_, ok := l.Grids.Load(geoHashString)
			if ok {
				t.Errorf("No grid should have even been created")
			}

			l.DeleteLocation(loc)
		})
	})

}
