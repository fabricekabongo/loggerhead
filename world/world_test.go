package world

import (
	"github.com/uber/h3-go"
	"testing"
)

func TestWorld(t *testing.T) {
	t.Parallel()
	t.Run("Save", func(t *testing.T) {
		t.Parallel()
		t.Run("Should save a new location", func(t *testing.T) {
			t.Parallel()
			world := NewWorld()

			err := world.Save("ns", "locId", 1.0, 1.0)
			if err != nil {
				t.Fatalf("Error saving location: %v", err)
			}

			ns := world.getNamespace("ns")

			if ns == nil {
				t.Fatalf("Namespace not found")
			}

			loc := ns.locations["locId"]
			if loc == nil {
				t.Fatalf("Location not found")
			}

			if loc.Lat != 1.0 || loc.Lon != 1.0 || loc.Id != "locId" || loc.Ns != "ns" {
				t.Fatalf("Expected location to be saved")
			}
		})

		t.Run("Should update an existing location", func(t *testing.T) {
			t.Parallel()
			world := NewWorld()

			err := world.Save("ns", "locId", 1.0, 1.0)
			if err != nil {
				t.Fatalf("Error saving location: %v", err)
			}

			err = world.Save("ns", "locId", 2.0, 2.0)
			if err != nil {
				t.Fatalf("Error saving location: %v", err)
			}

			ns := world.getNamespace("ns")

			if ns == nil {
				t.Fatalf("Namespace not found")
			}

			loc := ns.locations["locId"]
			if loc == nil {
				t.Fatalf("Location not found")
			}

			if loc.Lat != 2.0 || loc.Lon != 2.0 || loc.Id != "locId" || loc.Ns != "ns" {
				t.Fatalf("Expected location to be updated")
			}
		})

		t.Run("Should save location to all levels", func(t *testing.T) {
			t.Parallel()
			world := NewWorld()

			err := world.Save("ns", "locId", 1.0, 1.0)
			if err != nil {
				t.Fatalf("Error saving location: %v", err)
			}

			world.levels.Range(func(key, value interface{}) bool {
				level := value.(*Level)
				geo := h3.GeoCoord{
					Latitude:  1.0,
					Longitude: 1.0,
				}

				geoHash := h3.FromGeo(geo, int(level.Level))
				geoHashString := h3.ToString(geoHash)

				v, _ := level.index.Load("locId")
				gridName := v.(string)
				if gridName != geoHashString {
					t.Fatalf("Expected location to be saved to all levels")
				}

				v, ok := level.Grids.Load(geoHashString)
				if !ok {
					t.Fatalf("Expected location to be saved to all levels")
				}

				grid, ok := v.(*Grid)
				if !ok {
					t.Fatalf("Expected value to be of type *Grid")
				}

				v, ok = grid.namespaces["ns"]
				if !ok {
					t.Fatalf("Expected location to be saved to all levels")
				}

				namespace, ok := v.(map[string]*Location)
				if !ok {
					t.Fatalf("Expected value to be of type map[string]*Location")
				}

				_, ok = namespace["locId"]
				if !ok {
					t.Fatalf("Expected location to be saved to all levels")
				}

				return true
			})
		})
	})

	t.Run("GetLocation", func(t *testing.T) {
		t.Parallel()
		t.Run("Should return a location", func(t *testing.T) {
			t.Parallel()
			world := NewWorld()

			err := world.Save("ns", "locId", 1.0, 1.0)

			if err != nil {
				t.Fatalf("Error saving location: %v", err)
			}

			loc, found := world.GetLocation("ns", "locId")

			if !found {
				t.Fatalf("Expected location to be returned")
			}

			if loc.Lat != 1.0 || loc.Lon != 1.0 || loc.Id != "locId" || loc.Ns != "ns" {
				t.Fatalf("Expected location to be returned")
			}

		})

		t.Run("Should return boolean false if location not found", func(t *testing.T) {
			t.Parallel()
			world := NewWorld()

			_, found := world.GetLocation("ns", "locId")

			if found {
				t.Fatalf("Expected location to be nil")
			}
		})
	})

	t.Run("GetLocationsInRadius", func(t *testing.T) {
		t.Parallel()
		t.Run("Should return locations in radius", func(t *testing.T) {
			t.Parallel()
			world := NewWorld()

			err := world.Save("ns", "locId", 1.0002, 1.0002)
			if err != nil {
				t.Fatalf("Error saving location: %v", err)
			}
			err = world.Save("ns", "locId2", 1.0003, 1.0003)

			if err != nil {
				t.Fatalf("Error saving location: %v", err)
			}

			locations := world.GetLocationsInRadius("ns", 1.0, 1.0, 500)

			if len(locations) != 2 {
				t.Fatalf("Expected 2 location to be returned, got %v locations", len(locations))
			}
		})

		t.Run("Should return empty array if no locations in polygon", func(t *testing.T) {
			t.Parallel()
			world := NewWorld()

			err := world.Save("ns", "locId", 1.0, 1.0)

			if err != nil {
				t.Fatalf("Error saving location: %v", err)
			}

			locations := world.GetLocationsInRadius("ns", 35.0, 90.0, 500)

			if len(locations) != 0 {
				t.Fatalf("Expected 0 locations to be returned, got %v locations", len(locations))
			}
		})
	})
}
