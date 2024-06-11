package world

import (
	"math/rand"
	"strconv"
	"testing"
)

func CreateSeedData(records int) ([]map[string]float64, []map[string]string) {
	locs := []map[string]float64{}
	ids := []map[string]string{}

	for i := 0; i < records; i++ {
		lat := -90.0 + rand.Float64()*(90.0+90.0)
		lon := -180.0 + rand.Float64()*(180.0+180.0)
		id := "locId" + strconv.Itoa(i)
		ns := "ns" + strconv.Itoa(i%10)

		locs = append(locs, map[string]float64{
			"lat": lat,
			"lon": lon,
		})
		ids = append(ids, map[string]string{
			"id": id,
			"ns": ns,
		})
	}

	return locs, ids
}
func BenchmarkWorld(b *testing.B) {
	uniqueRecords := 10000
	locs, ids := CreateSeedData(uniqueRecords)
	b.Run("Save", func(b *testing.B) {
		b.Run("Should save a new location", func(b *testing.B) {
			world := NewWorld()

			for i := 0; i < b.N; i++ {

				loc := locs[i%uniqueRecords]
				id := ids[i%uniqueRecords]

				err := world.Save(id["ns"], id["id"], loc["lat"], loc["lon"])
				if err != nil {
					b.Fatalf("Error saving location: %v", err)
				}
			}
		})
	})

	b.Run("GetLocation", func(b *testing.B) {
		b.Run("Should return a location", func(b *testing.B) {
			world := NewWorld()

			for i := 0; i < b.N; i++ {
				loc := locs[i%uniqueRecords]
				id := ids[i%uniqueRecords]

				err := world.Save(id["ns"], id["id"], loc["lat"], loc["lon"])

				if err != nil {
					b.Fatalf("Error saving location: %v", err)
				}

				_, found := world.GetLocation(id["ns"], id["id"])

				if !found {
					b.Fatalf("Expected location to be returned")
				}
			}
		})
	})

	b.Run("GetLocationsInRadius", func(b *testing.B) {
		b.Run("Should return locations in radius", func(b *testing.B) {
			world := NewWorld()

			for i := 0; i < b.N; i++ {
				loc := locs[i%uniqueRecords]
				id := ids[i%uniqueRecords]

				err := world.Save(id["ns"], id["id"], loc["lat"], loc["lon"])
				if err != nil {
					b.Fatalf("Error saving location: %v", err)
				}

				loc2 := locs[(i+1)%uniqueRecords]
				id2 := ids[(i+1)%uniqueRecords]

				err = world.Save(id2["ns"], id2["id"], loc2["lat"], loc2["lon"])

				if err != nil {
					b.Fatalf("Error saving location: %v", err)
				}

				_ = world.GetLocationsInRadius("ns", loc["lat"], loc["lon"], 1000000)
			}
		})
	})
}
