package world

import (
	"math/rand"
	"strconv"
	"sync/atomic"
	"testing"
)

func CreateRandomLocation(seed int) (string, string, float64, float64) {
	lat := -90.0 + rand.Float64()*(90.0+90.0)
	lon := -180.0 + rand.Float64()*(180.0+180.0)
	id := strconv.Itoa(seed)

	return "1", id, lat, lon
}
func BenchmarkWorldParallel(b *testing.B) {
	world := NewWorld()
	b.Run("Save", func(b *testing.B) {
		b.Run("Parallel: Should save a new location", func(b *testing.B) {
			b.RunParallel(func(pb *testing.PB) {
				var ops atomic.Uint64
				var i int

				for pb.Next() {
					i = int(ops.Load())
					ns, id, lat, lon := CreateRandomLocation(i)

					err := world.Save(ns, id, lat, lon)
					if err != nil {
						b.Fatalf("Error saving location: %v", err)
					}
					ops.Add(1)
				}
			})
		})
	})
	b.Run("GetLocation", func(b *testing.B) {
		b.Run("Parallel: Should return a location", func(b *testing.B) {
			b.RunParallel(func(pb *testing.PB) {
				var ops atomic.Uint64
				var i int

				for pb.Next() {
					i = int(ops.Load())
					id := strconv.Itoa(i)

					world.GetLocation("1", id)
				}
			})
		})
	})
	b.Run("GetLocationsInRadius", func(b *testing.B) {
		b.Run("Parallel: Should return locations in the UAE 83.6k km square", func(b *testing.B) {
			b.RunParallel(func(pb *testing.PB) {

				for pb.Next() {
					_ = world.QueryRange("1", 22, 26, 51, 56)
				}
			})
		})
		b.Run("Parallel: Should return locations in the USA 9.8m km square", func(b *testing.B) {
			b.RunParallel(func(pb *testing.PB) {
				var ops atomic.Uint64
				for pb.Next() {
					_ = world.QueryRange("1", -25, 49, -124, -66)
					ops.Add(1)
				}
			})
		})
		b.Run("Parallel: Should return locations in all of africa 30m km square", func(b *testing.B) {
			b.RunParallel(func(pb *testing.PB) {
				var ops atomic.Uint64
				for pb.Next() {
					_ = world.QueryRange("1", -34, 37, -17, 51.7)
					ops.Add(1)
				}
			})
		})
	})

	b.Run("Delete", func(b *testing.B) {
		b.Run("Parallel: Should delete a location", func(b *testing.B) {
			b.RunParallel(func(pb *testing.PB) {
				var ops atomic.Uint64
				for pb.Next() {
					i := int(ops.Load())
					id := strconv.Itoa(i)
					world.Delete("1", id)
					ops.Add(1)
				}
			})
		})
	})
}
func BenchmarkWorld(b *testing.B) {
	world := NewWorld()

	b.Run("Save", func(b *testing.B) {
		b.Run("Should save a new location", func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				ns, id, lat, lon := CreateRandomLocation(i)

				err := world.Save(ns, id, lat, lon)
				if err != nil {
					b.Fatalf("Error saving location: %v", err)
				}
			}
		})
	})

	b.Run("GetLocation", func(b *testing.B) {
		b.Run("Should return a location", func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				id := strconv.Itoa(i)

				world.GetLocation("1", id)
			}
		})
	})

	b.Run("GetLocationsInRadius", func(b *testing.B) {
		b.Run("Should return locations in Singapore 734.3 km square", func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_ = world.QueryRange(strconv.Itoa(i%10), 1.16, 1.48, 103.6, 104)
			}
		})
		b.Run("Should return locations in the UAE 83.6k km square", func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_ = world.QueryRange(strconv.Itoa(i%10), 22, 26, 51, 56)
			}
		})
		b.Run("Should return locations in the USA 9.8m km square", func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_ = world.QueryRange(strconv.Itoa(i%10), -25, 49, -124, -66)
			}
		})
		b.Run("Should return locations in all of africa 30m km square ", func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_ = world.QueryRange(strconv.Itoa(i%10), -34, 37, -17, 51.7)
			}
		})
	})

	b.Run("Delete", func(b *testing.B) {
		b.Run("Should delete a location", func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				id := strconv.Itoa(i)
				world.Delete("1", id)
			}
		})
	})
}
