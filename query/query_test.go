package query

import (
	"fmt"
	w "github.com/fabricekabongo/loggerhead/world"
	"math/rand/v2"
	"strconv"
	"strings"
	"testing"
)

func TestQuery(t *testing.T) {
	t.Run("Invalid Query", func(t *testing.T) {
		t.Run("should return an error if the query is invalid", func(t *testing.T) {
			world := w.NewWorld()
			queryProcessor := NewQueryEngine(world)

			query := fmt.Sprintf("%s %s %s", strconv.Itoa(int(rand.Int32())), strconv.Itoa(int(rand.Int32())), strconv.Itoa(int(rand.Int32())))

			data := queryProcessor.ExecuteQuery(query)
			if data != "1.0,\"invalid query\"\n" {
				t.Errorf("Expected \"1.0,invalid query\" but got %v", data)
			}
		})

	})
	t.Run("GetQuery", func(t *testing.T) {
		t.Run("should return a location", func(t *testing.T) {
			world := w.NewWorld()
			err := world.Save("ns-id-8", "loc-id-9", 1.0, 2.0)
			if err != nil {
				t.Errorf("Error saving location: %v", err)
			}

			queryProcessor := NewQueryEngine(world)

			query := "GET ns-id-8 loc-id-9"

			data := queryProcessor.ExecuteQuery(query)

			if data != "1.0,ns-id-8,loc-id-9,1.000000,2.000000\n1.0,done\n" {
				t.Errorf("Expected '1.0,ns-id-8,loc-id-9,1.000000,2.000000\n' but got %v", data)
			}
		})

		t.Run("should return an empty string if the result is empty", func(t *testing.T) {
			world := w.NewWorld()
			queryProcessor := NewQueryEngine(world)

			query := "GET ns-id-8 loc-id-9"

			data := queryProcessor.ExecuteQuery(query)

			if data != "1.0,done\n" {
				t.Errorf("Expected '1.0,done\n' but got %v", data)
			}
		})
	})
	t.Run("DeleteQuery", func(t *testing.T) {
		t.Run("should delete a location from the world", func(t *testing.T) {
			world := w.NewWorld()
			err := world.Save("ns-id-8", "loc-id-9", 1.0, 2.0)
			if err != nil {
				t.Errorf("Error saving location: %v", err)
			}

			queryProcessor := NewQueryEngine(world)

			query := "DELETE ns-id-8 loc-id-9"

			data := queryProcessor.ExecuteQuery(query)
			if data != "1.0,deleted\n" {
				t.Errorf("Expected \"1.0,deleted\" got %v", data)
			}

			query = "GET ns-id-8 loc-id-9"

			data = queryProcessor.ExecuteQuery(query)
			if data != "1.0,done\n" {
				t.Errorf("Expected '1.0,' but got %v", data)
			}
		})
	})
	t.Run("SaveQuery", func(t *testing.T) {
		t.Run("should save a location to the world", func(t *testing.T) {
			world := w.NewWorld()
			queryProcessor := NewQueryEngine(world)

			query := "SAVE ns-id-8 loc-id-9 1.0 2.0"

			data := queryProcessor.ExecuteQuery(query)

			if data != "1.0,saved\n" {
				t.Errorf("Expected '1.0,saved' but got %v", data)
			}

			query = "GET ns-id-8 loc-id-9"

			data = queryProcessor.ExecuteQuery(query)

			if data != "1.0,ns-id-8,loc-id-9,1.000000,2.000000\n1.0,done\n" {
				t.Errorf("Expected 'ns-id-8,loc-id-9,1.000000,2.000000\n1.0,done\n' but got %v", data)
			}
		})

		t.Run("should update a location in the world", func(t *testing.T) {
			world := w.NewWorld()
			queryProcessor := NewQueryEngine(world)

			query := "SAVE ns-id-8 loc-id-9 1.0 2.0"

			data := queryProcessor.ExecuteQuery(query)

			if data != "1.0,saved\n" {
				t.Errorf("Expected '1.0,saved' but got %v", data)
			}

			query = "SAVE ns-id-8 loc-id-9 2.0 3.0"

			data = queryProcessor.ExecuteQuery(query)

			if data != "1.0,saved\n" {
				t.Errorf("Expected '1.0,saved' but got %v", data)
			}

			query = "GET ns-id-8 loc-id-9"

			data = queryProcessor.ExecuteQuery(query)

			if data != "1.0,ns-id-8,loc-id-9,2.000000,3.000000\n1.0,done\n" {
				t.Errorf("Expected 'ns-id-8,loc-id-9,2.000000,3.000000\n1.0,done\n' but got %v", data)
			}
		})

		t.Run("should return an error if the longitude is invalid", func(t *testing.T) {
			world := w.NewWorld()
			queryProcessor := NewQueryEngine(world)

			query := "SAVE ns-id-8 loc-id-9 1.0 200.0"

			data := queryProcessor.ExecuteQuery(query)

			if data != "1.0,\"invalid longitude\"\n" {
				t.Errorf("Expected \"1.0,\"invalid longitude\"\n\" but got %v", data)
			}

			query = "GET ns-id-8 loc-id-9"

			data = queryProcessor.ExecuteQuery(query)

			if data != "1.0,done\n" {
				t.Errorf("Expected \"1.0,done\n\" but got %v", data)
			}
		})
		t.Run("should return an error if the latitude is invalid", func(t *testing.T) {
			world := w.NewWorld()
			queryProcessor := NewQueryEngine(world)

			query := "SAVE ns-id-8 loc-id-9 100 80"

			data := queryProcessor.ExecuteQuery(query)

			if data != "1.0,\"invalid latitude\"\n" {
				t.Errorf("Expected \"1.0,invalid latitude\n\" but got %v", data)
			}

			query = "GET ns-id-8 loc-id-9"

			data = queryProcessor.ExecuteQuery(query)

			if data != "1.0,done\n" {
				t.Errorf("Expected \"1.0,done\n\" but got %v", data)
			}
		})
		t.Run("should return an error if location aren't floats", func(t *testing.T) {
			world := w.NewWorld()
			queryProcessor := NewQueryEngine(world)

			query := "SAVE ns-id-8 loc-id-9 ina 90"

			data := queryProcessor.ExecuteQuery(query)

			if data != "1.0,\"Invalid float64 value for latitude\"\n" {
				t.Errorf("Expected \"1.0,Invalid float64 value for latitude\" but got %v", data)
			}

			query = "SAVE ns-id-8 loc-id-9 70 monga"

			data = queryProcessor.ExecuteQuery(query)

			if data != "1.0,\"Invalid float64 value for longitude\"\n" {
				t.Errorf("Expected \"1.0,Invalid float64 value for longitude\" but got %v", data)
			}
		})
	})
	t.Run("POLY Query", func(t *testing.T) {
		t.Run("should return a list of locations", func(t *testing.T) {
			world := w.NewWorld()
			queryProcessor := NewQueryEngine(world)

			query := "SAVE ns-id-8 loc-id-9 1.0 2.0"
			data := queryProcessor.ExecuteQuery(query)
			if data != "1.0,saved\n" {
				t.Errorf("expected \"1.0,saved\" got %v", data)
			}

			query = "SAVE ns-id-8 loc-id-10 1.5 2.0"
			data = queryProcessor.ExecuteQuery(query)
			if data != "1.0,saved\n" {
				t.Errorf("expected \"1.0,saved\" got %v", data)
			}

			query = "POLY ns-id-8 0 0 2 2" // lat1 lon1 lat2 lon2

			data = queryProcessor.ExecuteQuery(query)

			lines := strings.Split(data, "\n")

			expLines := map[string]string{
				"1.0,ns-id-8,loc-id-9,1.000000,2.000000":  "1.0,ns-id-8,loc-id-9,1.000000,2.000000",
				"1.0,ns-id-8,loc-id-10,1.500000,2.000000": "1.0,ns-id-8,loc-id-10,1.500000,2.000000",
				"1.0,done": "1.0,done",
			}

			for _, expLine := range expLines {
				if expLine == lines[0] || expLine == lines[1] || expLine == lines[2] {
					continue
				}

				t.Fatalf("failed to compare expected lines with lines.")
			}
		})

		t.Run("should return an empty string if the result is empty", func(t *testing.T) {
			world := w.NewWorld()
			queryProcessor := NewQueryEngine(world)

			query := "POLY ns-id-8 0 0 2 2" // lat1 lon1 lat2 lon2

			data := queryProcessor.ExecuteQuery(query)

			if data != "1.0,done\n" {
				t.Errorf("Expected '1.0,' but got %v", data)
			}
		})
	})
}
