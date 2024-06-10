package world

import "testing"

func TestGrid(t *testing.T) {
	t.Parallel()

	t.Run("NewGridError", func(t *testing.T) {
		t.Parallel()

		_, err := NewGrid("")
		if err == nil {
			t.Error("Expected error for empty name")
		}

		if err != ErrorGridNameRequired {
			t.Errorf("Expected ErrorGridNameRequired, got %v", err)
		}

	})
	t.Run("NewGrid", func(t *testing.T) {
		t.Parallel()

		g, err := NewGrid("test")
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if g == nil {
			t.Error("NewGrid should return a Grid")
		}

		if g.Name != "test" {
			t.Errorf("Expected name to be test, got %s", g.Name)
		}

		if g.AddEventSubscribers == nil {
			t.Error("AddEventSubscribers should be initialized")
		}

		if g.UpdateEventSubscribers == nil {
			t.Error("UpdateEventSubscribers should be initialized")
		}

		if g.DeleteEventSubscribers == nil {
			t.Error("DeleteEventSubscribers should be initialized")
		}
	})

	t.Run("SaveLocation", func(t *testing.T) {

	})
}
