package world

import (
	"math/rand"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
)

func CreateSeedData(records int) (sync.Map, sync.Map) {
	locs := sync.Map{}
	ids := sync.Map{}

	for i := 0; i < records; i++ {
		lat := -90.0 + rand.Float64()*(90.0+90.0)
		lon := -180.0 + rand.Float64()*(180.0+180.0)
		id := "locId" + strconv.Itoa(i)
		ns := "ns" + strconv.Itoa(i%10)
		locs.Store(i, map[string]float64{
			"lat": lat,
			"lon": lon,
		})

		ids.Store(i, map[string]string{
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
				entry1, ok := locs.Load(i % uniqueRecords)
				if !ok {
					b.Fatalf("Error loading location with id: %v", i%uniqueRecords)
				}

				loc := entry1.(map[string]float64)

				entry2, ok2 := ids.Load(i % uniqueRecords)
				if !ok2 {
					b.Fatalf("Error loading id")
				}

				id := entry2.(map[string]string)

				err := world.Save(id["ns"], id["id"], loc["lat"], loc["lon"])
				if err != nil {
					b.Fatalf("Error saving location: %v", err)
				}
			}
		})
		b.Run("Parallel: Should save a new location", func(b *testing.B) {
			b.RunParallel(func(pb *testing.PB) {
				var ops atomic.Uint64

				for pb.Next() {
					i := int(ops.Load())
					entry1, ok := locs.Load(i % uniqueRecords)
					if !ok {
						b.Fatalf("Error loading location with id: %v", i%uniqueRecords)
					}
					loc := entry1.(map[string]float64)

					entry2, ok := ids.Load(i % uniqueRecords)
					if !ok {
						b.Fatalf("Error loading id with id: %v", i%uniqueRecords)
					}
					id := entry2.(map[string]string)

					err := world.Save(id["ns"], id["id"], loc["lat"], loc["lon"])
					if err != nil {
						b.Fatalf("Error saving location: %v", err)
					}
					ops.Add(1)
				}
			})
		})
	})

	b.Run("GetLocation", func(b *testing.B) {
		b.Run("Should return a location", func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				entry2, _ := ids.Load(i % uniqueRecords)
				id := entry2.(map[string]string)
				world.GetLocation(id["ns"], id["id"])
			}
		})
	})

	b.Run("GetLocationsInRadius", func(b *testing.B) {
		b.Run("Should return locations in Singapore (734.3 km²)", func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_ = world.QueryRange("ns"+strconv.Itoa(i%10), 1.16, 1.48, 103.6, 104)
			}
		})
		b.Run("Should return locations in the UAE (83.6k km²)", func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_ = world.QueryRange("ns"+strconv.Itoa(i%10), 22, 26, 51, 56)
			}
		})
		b.Run("Should return locations in the USA (9.8 m km²)", func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_ = world.QueryRange("ns"+strconv.Itoa(i%10), -25, 49, -124, -66)
			}
		})
		b.Run("Should return locations in all of africa (30 m km²) ", func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_ = world.QueryRange("ns"+strconv.Itoa(i%10), -34, 37, -17, 51.7)
			}
		})
		b.Run("Parallel: Should return locations in the UAE (83.6k km²)", func(b *testing.B) {
			b.RunParallel(func(pb *testing.PB) {
				var ops atomic.Uint64

				for pb.Next() {
					i := ops.Load()
					_ = world.QueryRange("ns"+strconv.Itoa(int(i%10)), 22, 26, 51, 56)
					ops.Add(1)
				}
			})
		})
		b.Run("Parallel: Should return locations in the USA (9.8 m km²)", func(b *testing.B) {
			b.RunParallel(func(pb *testing.PB) {
				var ops atomic.Uint64
				for pb.Next() {
					i := ops.Load()
					_ = world.QueryRange("ns"+strconv.Itoa(int(i%10)), -25, 49, -124, -66)
					ops.Add(1)
				}
			})
		})
		b.Run("Parallel: Should return locations in all of africa (30 m km²) ", func(b *testing.B) {
			b.RunParallel(func(pb *testing.PB) {
				var ops atomic.Uint64
				for pb.Next() {
					i := ops.Load()
					_ = world.QueryRange("ns"+strconv.Itoa(int(i%10)), -34, 37, -17, 51.7)
					ops.Add(1)
				}
			})
		})
	})

	b.Run("Delete", func(b *testing.B) {
		b.Run("Should delete a location", func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				entry, _ := ids.Load(i % uniqueRecords)
				id := entry.(map[string]string)

				world.Delete(id["ns"], id["id"])
			}
		})
	})
}
