package world

import (
	"github.com/stretchr/testify/assert"
	"sync"
	"testing"
)

func TestNode(t *testing.T) {
	t.Parallel()
	t.Run("Delete", func(t *testing.T) {
		t.Parallel()
		t.Run("Should delete a location", func(t *testing.T) {
			t.Parallel()
			node := createTestNode(t)

			node.Delete("locId2")

			if len(node.Objects) != 2 {
				t.Fatalf("Expected 0 locations to be returned, got %v locations", len(node.Objects))
			}
		})

		t.Run("should not panic if the last element in the slice is deleted", func(t *testing.T) {
			t.Parallel()

			node := createTestNode(t)

			node.Delete("locId3")
		})

		t.Run("should not panic if the first element in the slice is deleted", func(t *testing.T) {
			t.Parallel()

			node := createTestNode(t)
			waitGroup := sync.WaitGroup{}
			for i := 0; i < 100; i++ {
				waitGroup.Add(1)
				go func() {
					defer waitGroup.Done()
					node.Delete("locId")
				}()
			}

			waitGroup.Wait()
		})

		t.Run("should not panic if the last element in the slice is deleted", func(t *testing.T) {
			t.Parallel()

			node := createTestNode(t)

			node.Delete("locId3")
		})

		t.Run("should not panic if the first element in the slice is deleted", func(t *testing.T) {
			t.Parallel()

			node := createTestNode(t)

			node.Delete("locId")
		})
	})

	t.Run("Insert", func(t *testing.T) {
		t.Parallel()
		t.Run("Should divide when reaching capacity", func(t *testing.T) {
			t.Parallel()
			node := &TreeNode{
				Objects:  make(map[string]*Location, 0),
				Capacity: 2,
				Lat1:     -90,
				Lat2:     90,
				Lon1:     -180,
				Lon2:     180,
			}

			loc, err := NewLocation("ns", "locId", -67.0, 1.0)
			assert.ErrorIs(t, err, nil)

			err = node.insert(loc)
			assert.ErrorIs(t, err, nil)

			loc, err = NewLocation("ns", "locId2", 2.0, -45.0)
			assert.ErrorIs(t, err, nil)

			err = node.insert(loc)
			assert.ErrorIs(t, err, nil)

			// Inserting a third location should trigger a divide
			loc, err = NewLocation("ns", "locId3", 2.0, 2.2)
			assert.ErrorIs(t, err, nil)

			err = node.insert(loc)
			assert.ErrorIs(t, err, nil)

			assert.True(t, node.IsDivided)

			assert.NotNil(t, node.NE)
			assert.NotNil(t, node.NW)
			assert.NotNil(t, node.SE)
			assert.NotNil(t, node.SW)

			assert.Len(t, node.Objects, 0)
			assert.Len(t, node.NE.Objects, 1)
		})
	})
}

func createTestNode(t *testing.T) *TreeNode {
	node := &TreeNode{
		Objects: make(map[string]*Location, 0),
	}
	loc, err := NewLocation("ns", "locId", 1.0, 1.0)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	node.Objects[loc.Id()] = loc
	loc, err = NewLocation("ns", "locId2", 2.0, 2.0)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	node.Objects[loc.Id()] = loc
	loc, err = NewLocation("ns", "locId3", 2.0, 2.2)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	node.Objects[loc.Id()] = loc
	return node
}
