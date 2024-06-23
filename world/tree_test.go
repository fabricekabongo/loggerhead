package world

import "testing"

func TestTree(t *testing.T) {
	t.Parallel()
	t.Run("Delete", func(t *testing.T) {
		t.Parallel()
		t.Run("Should delete a location", func(t *testing.T) {
			t.Parallel()
			tree := NewQuadTree(-90, 90, -180, 180)
			loc, err := NewLocation("ns", "locId", 1.0, 1.0)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			err = tree.Insert(loc)
			if err != nil {
				t.Fatalf("Error inserting location: %v", err)
			}

			tree.Root.Delete("locId")

			locations := tree.Root.QueryRange(-90, 90, -180, 180)

			if len(locations) != 0 {
				t.Fatalf("Expected 0 locations to be returned, got %v locations", len(locations))
			}
		})

		t.Run("Should not panic if location not found", func(t *testing.T) {
			t.Parallel()
			tree := NewQuadTree(-90, 90, -180, 180)

			tree.Root.Delete("locId")
		})
	})
}
