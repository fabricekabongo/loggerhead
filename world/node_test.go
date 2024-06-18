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
			node := &TreeNode{
				Objects: []*Location{
					{
						Lat: 1.0,
						Lon: 1.0,
						Id:  "locId",
					},
					{
						Lat: 2.0,
						Lon: 2.0,
						Id:  "locId2",
					},
					{
						Lat: 2.0,
						Lon: 2.2,
						Id:  "locId3",
					},
				},
			}

			node.Delete("locId2")

			if len(node.Objects) != 2 {
				t.Fatalf("Expected 0 locations to be returned, got %v locations", len(node.Objects))
			}
		})

		t.Run("should not panic if the last element in the slice is deleted", func(t *testing.T) {
			t.Parallel()
			node := &TreeNode{
				Objects: []*Location{
					{
						Lat: 1.0,
						Lon: 1.0,
						Id:  "locId",
					},
					{
						Lat: 2.0,
						Lon: 2.0,
						Id:  "locId2",
					},
					{
						Lat: 2.0,
						Lon: 2.2,
						Id:  "locId3",
					},
				},
			}

			node.Delete("locId3")
		})

		t.Run("should not panic if the first element in the slice is deleted", func(t *testing.T) {
			t.Parallel()
			node := &TreeNode{
				Objects: []*Location{
					{
						Lat: 1.0,
						Lon: 1.0,
						Id:  "locId",
					},
					{
						Lat: 2.0,
						Lon: 2.0,
						Id:  "locId2",
					},
					{
						Lat: 2.0,
						Lon: 2.2,
						Id:  "locId3",
					},
				},
			}
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
			node := &TreeNode{
				Objects: []*Location{
					{
						Lat: 1.0,
						Lon: 1.0,
						Id:  "locId",
					},
					{
						Lat: 2.0,
						Lon: 2.0,
						Id:  "locId2",
					},
					{
						Lat: 2.0,
						Lon: 2.2,
						Id:  "locId3",
					},
				},
			}

			node.Delete("locId3")
		})

		t.Run("should not panic if the first element in the slice is deleted", func(t *testing.T) {
			t.Parallel()
			node := &TreeNode{
				Objects: []*Location{
					{
						Lat: 1.0,
						Lon: 1.0,
						Id:  "locId",
					},
					{
						Lat: 2.0,
						Lon: 2.0,
						Id:  "locId2",
					},
					{
						Lat: 2.0,
						Lon: 2.2,
						Id:  "locId3",
					},
				},
			}

			node.Delete("locId")
		})
	})
}
