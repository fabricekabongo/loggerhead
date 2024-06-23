package world

import (
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
}

func createTestNode(t *testing.T) *TreeNode {
	node := &TreeNode{
		Objects: []*Location{},
	}
	loc, err := NewLocation("ns", "locId", 1.0, 1.0)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	node.Objects = append(node.Objects, loc)
	loc, err = NewLocation("ns", "locId2", 2.0, 2.0)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	node.Objects = append(node.Objects, loc)
	loc, err = NewLocation("ns", "locId3", 2.0, 2.2)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	node.Objects = append(node.Objects, loc)
	return node
}
