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
	uniqueRecords := 1000000
	locs, ids := CreateSeedData(uniqueRecords)
	world := NewWorld()

	b.Run("Save", func(b *testing.B) {
		b.Run("Should save a new location", func(b *testing.B) {
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
			for i := 0; i < b.N; i++ {
				id := ids[i%uniqueRecords]
				world.GetLocation(id["ns"], id["id"])
			}
		})
	})

	b.Run("GetLocationsInRadius", func(b *testing.B) {
		b.Run("Should return locations in the UAE (83.6k km^2)", func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_ = world.QueryRange("ns"+strconv.Itoa(i%10), 22, 26, 51, 56)
			}
		})
		b.Run("Should return locations in the USA (9.8 m Km^2)", func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_ = world.QueryRange("ns"+strconv.Itoa(i%10), -25, 49, -124, -66)
			}
		})
		b.Run("Should return locations in all of africa (30 m Km^2) ", func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_ = world.QueryRange("ns"+strconv.Itoa(i%10), -34, 37, -17, 51.7)
			}
		})
	})

	b.Run("Delete", func(b *testing.B) {
		b.Run("Should delete a location", func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				id := ids[i%uniqueRecords]
				world.Delete(id["ns"], id["id"])
			}
		})

	})
}
