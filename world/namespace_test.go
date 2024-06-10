package world

import "testing"

func TestNamespace(t *testing.T) {
	t.Parallel()

	t.Run("SaveLocation", func(t *testing.T) {
		t.Parallel()
		t.Run("should add a location to the namespace", func(t *testing.T) {
			t.Parallel()
			ns := NewNamespace("test")

			loc, err := ns.SaveLocation("id", 87, 125)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if len(ns.locations) != 1 {
				t.Errorf("expected 1 location, got %d", len(ns.locations))
			}

			if ns.locations["id"] != loc {
				t.Errorf("expected location to be added to the namespace")
			}

			if loc.Lat != 87 {
				t.Errorf("expected latitude to be 87, got %f", loc.Lat)
			}

			if loc.Lon != 125 {
				t.Errorf("expected longitude to be 125, got %f", loc.Lon)
			}

			if loc.Ns != "test" {
				t.Errorf("expected namespace to be test, got %s", loc.Ns)
			}

			if loc.Id != "id" {
				t.Errorf("expected location id to be id, got %s", loc.Id)
			}
		})

		t.Run("should update a location in the namespace if already exist", func(t *testing.T) {
			t.Parallel()
			ns := NewNamespace("test")
			loc, err := ns.SaveLocation("id", 87, 125)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			loc2, err := ns.SaveLocation("id", 88, 126)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if len(ns.locations) != 1 {
				t.Errorf("expected 1 location, got %d", len(ns.locations))
			}

			if loc != loc2 {
				t.Errorf("expect same location pointer to be used")
			}

			if loc.Lat != 88 || loc.Lon != 126 {
				t.Errorf("expected location to be updated")
			}
		})
	})

	t.Run("DeleteLocation", func(t *testing.T) {
		t.Parallel()
		t.Run("should delete a location from the namespace", func(t *testing.T) {
			t.Parallel()
			ns := NewNamespace("test")
			_, _ = ns.SaveLocation("id", 87, 125)

			ns.DeleteLocation("id")

			if len(ns.locations) != 0 {
				t.Errorf("expected 0 location, got %d", len(ns.locations))
			}
		})

		t.Run("should not error if location does not exist", func(t *testing.T) {
			t.Parallel()
			ns := NewNamespace("test")
			ns.DeleteLocation("id")
		})
	})
}
