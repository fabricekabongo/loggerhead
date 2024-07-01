package world

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestTree(t *testing.T) {
	t.Parallel()
	t.Run("Delete", func(t *testing.T) {
		t.Parallel()
		t.Run("Should delete a location", func(t *testing.T) {
			t.Parallel()
			tree := NewQuadTree(-90, 90, -180, 180)
			loc, err := NewLocation("ns", "locId", 1.0, 1.0)
			assert.ErrorIs(t, err, nil)

			err = tree.Insert(loc)
			assert.ErrorIs(t, err, nil)
			assert.NotNil(t, loc.Node)

			tree.Root.Delete("locId")

			locations := tree.Root.QueryRange(-90, 90, -180, 180)

			assert.Len(t, locations, 1, "Delete was cascaded, which we don't want as it is slow")

			loc.Node.Delete("locId")
			locations = tree.Root.QueryRange(-90, 90, -180, 180)
			assert.Len(t, locations, 0)
		})

		t.Run("Should not panic if location not found", func(t *testing.T) {
			t.Parallel()
			tree := NewQuadTree(-90, 90, -180, 180)

			tree.Root.Delete("locId")
		})
	})
	t.Run("Insert", func(t *testing.T) {
		t.Parallel()
		t.Run("Should insert a location", func(t *testing.T) {
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

			locations := tree.Root.QueryRange(-90, 90, -180, 180)

			if len(locations) != 1 {
				t.Fatalf("Expected 1 location to be returned, got %v locations", len(locations))
			}
		})
		t.Run("Should move the location in the tree if the location is updated", func(t *testing.T) {
			t.Parallel()
			tree := NewQuadTree(-90, 90, -180, 180)
			tree.Root.Capacity = 2
			loc, err := NewLocation("ns", "locId", -89.0, -179.0)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			err = tree.Insert(loc)
			if err != nil {
				t.Fatalf("Error inserting location: %v", err)
			}

			loc2, err := NewLocation("ns", "locId2", 89.0, 179.0)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			err = tree.Insert(loc2)
			if err != nil {
				t.Fatalf("Error inserting location: %v", err)
			}

			locations := tree.Root.QueryRange(-90, 90, -180, 180)
			if len(locations) != 2 {
				t.Fatalf("Expected 2 locations to be returned, got %v locations", len(locations))
			}

			locations = tree.Root.QueryRange(-90, 0, -180, 0)
			if len(locations) != 1 {
				t.Fatalf("Expected 2 locations to be returned, got %v locations", len(locations))
			}

			locations = tree.Root.QueryRange(0, 90, 0, 180)
			if len(locations) != 1 {
				t.Fatalf("Expected 2 locations to be returned, got %v locations", len(locations))
			}

			err = loc.Update(89.0, 179.0)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			err = tree.Insert(loc)
			if err != nil {
				t.Fatalf("Error inserting location: %v", err)
			}

			locations = tree.Root.QueryRange(-90, 90, -180, 180)
			if len(locations) != 2 {
				t.Fatalf("Expected 2 locations to be returned, got %v locations", len(locations))
			}

			locations = tree.Root.QueryRange(-90, 0, -180, 0)
			if len(locations) != 0 {
				t.Fatalf("Expected 2 locations to be returned, got %v locations", len(locations))
			}

			locations = tree.Root.QueryRange(0, 90, 0, 180)
			if len(locations) != 2 {
				t.Fatalf("Expected 2 locations to be returned, got %v locations", len(locations))
			}
		})
	})
}
