package world

import (
	"testing"
)

func TestWorld(t *testing.T) {
	t.Parallel()

	t.Run("Merge", func(t *testing.T) {
		t.Parallel()
		t.Run("Should merge two worlds", func(t *testing.T) {
			t.Parallel()
			world1 := NewWorld()
			world2 := NewWorld()

			err := world1.Save("ns", "locId1", 1.0, 1.0)
			if err != nil {
				t.Fatalf("Error saving location: %v", err)
			}
			err = world2.Save("ns", "locId2", 2.0, 2.0)
			if err != nil {
				t.Fatalf("Error saving location: %v", err)
			}

			world1.Merge(world2)

			loc1, found1 := world1.GetLocation("ns", "locId1")
			if !found1 {
				t.Fatalf("Expected locId1 to be found after merge")
			}
			if loc1.Lat() != 1.0 || loc1.Lon() != 1.0 {
				t.Fatalf("Expected locId1 to have correct coordinates after merge")
			}

			loc2, found2 := world1.GetLocation("ns", "locId2")
			if !found2 {
				t.Fatalf("Expected locId2 to be found after merge")
			}
			if loc2.Lat() != 2.0 || loc2.Lon() != 2.0 {
				t.Fatalf("Expected locId2 to have correct coordinates after merge")
			}
		})
	})
	t.Run("Save", func(t *testing.T) {
		t.Parallel()
		t.Run("Should save a new location", func(t *testing.T) {
			t.Parallel()
			world := NewWorld()

			err := world.Save("ns", "locId", 1.0, 1.0)
			if err != nil {
				t.Fatalf("Error saving location: %v", err)
			}

			loc, ok := world.GetLocation("ns", "locId")

			if !ok {
				t.Fatalf("Expected location to be saved")
			}

			if loc.Lat() != 1.0 || loc.Lon() != 1.0 || loc.Id() != "locId" || loc.Ns() != "ns" {
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

			loc, ok := world.GetLocation("ns", "locId")

			if !ok {
				t.Fatalf("Expected location to be updated")
			}

			if loc.Lat() != 2.0 || loc.Lon() != 2.0 || loc.Id() != "locId" || loc.Ns() != "ns" {
				t.Fatalf("Expected location to be updated")
			}
		})
	})

	t.Run("GetLocation", func(t *testing.T) {
		t.Parallel()
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

			locations := world.QueryRange("ns", 1.0001, 1.0006, 1.0001, 1.0006)

			if len(locations) != 2 {
				t.Fatalf("Expected 2 locations to be returned, got %v locations", len(locations))
			}
		})

		t.Run("Should return empty array if no locations in polygon", func(t *testing.T) {
			t.Parallel()
			world := NewWorld()

			err := world.Save("ns", "locId", 1.0, 1.0)

			if err != nil {
				t.Fatalf("Error saving location: %v", err)
			}

			locations := world.QueryRange("ns", 80, 90, 30, 40)

			if len(locations) != 0 {
				t.Fatalf("Expected 0 locations to be returned, got %v locations", len(locations))
			}
		})
	})

	t.Run("Delete", func(t *testing.T) {
		t.Parallel()
		t.Run("Should delete a location", func(t *testing.T) {
			t.Parallel()
			world := NewWorld()

			err := world.Save("ns", "locId", 1.0, 1.0)

			if err != nil {
				t.Fatalf("Error saving location: %v", err)
			}

			world.Delete("ns", "locId")

			_, found := world.GetLocation("ns", "locId")

			if found {
				t.Fatalf("Expected location to be deleted")
			}
		})

		t.Run("Should not panic if location not found", func(t *testing.T) {
			t.Parallel()
			world := NewWorld()

			world.Delete("ns", "locId")
		})

		t.Run("Should not panic if namespace not found", func(t *testing.T) {
			t.Parallel()
			world := NewWorld()

			world.Delete("ns", "locId")
		})
	})
}
