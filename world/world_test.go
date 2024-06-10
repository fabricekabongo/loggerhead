package world

import (
	"github.com/uber/h3-go"
	"testing"
)

func TestWorld(t *testing.T) {
	t.Run("Save", func(t *testing.T) {
		t.Run("Should save a new location", func(t *testing.T) {
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
			world := NewWorld()

			err := world.Save("ns", "locId", 1.0, 1.0)
			if err != nil {
				t.Fatalf("Error saving location: %v", err)
			}

			for _, level := range world.levels {
				geo := h3.GeoCoord{
					Latitude:  1.0,
					Longitude: 1.0,
				}

				geoHash := h3.FromGeo(geo, int(level.Level))
				geoHashString := h3.ToString(geoHash)

				if level.index["locId"] != geoHashString {
					t.Fatalf("Expected location to be saved to all levels")
				}

				if level.Grids[geoHashString] == nil {
					t.Fatalf("Expected location to be saved to all levels")
				}

				if level.Grids[geoHashString].namespaces["ns"] == nil {
					t.Fatalf("Expected location to be saved to all levels")
				}

				if level.Grids[geoHashString].namespaces["ns"]["locId"] == nil {
					t.Fatalf("Expected location to be saved to all levels")
				}
			}
		})
	})
}
