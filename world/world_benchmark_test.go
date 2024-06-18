package world

import (
	"math/rand"
	"strconv"
	"sync/atomic"
	"testing"
)

func createRandomLocation(seed int) (*Location, error) {
	lat := -90.0 + rand.Float64()*(90.0+90.0)
	lon := -180.0 + rand.Float64()*(180.0+180.0)
	id := "locId" + strconv.Itoa(seed)
	ns := "ns" + strconv.Itoa(seed%10)

	return NewLocation(ns, id, lat, lon)
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
					location, err := createRandomLocation(i)
					if err != nil {
						b.Fatalf("Error creating  random location: %v", err)
					}

					err = world.Save(location.Ns, location.Id, location.Lat, location.Lon)
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
					id := "locId" + strconv.Itoa(i)
					ns := "ns" + strconv.Itoa(i%10)

					world.GetLocation(ns, id)
				}
			})
		})
	})
	b.Run("GetLocationsInRadius", func(b *testing.B) {
		b.Run("Parallel: Should return locations in the UAE (83.6k km²)", func(b *testing.B) {
			b.RunParallel(func(pb *testing.PB) {
				var ops atomic.Uint64

				for pb.Next() {
					i := int(ops.Load())
					ns := "ns" + strconv.Itoa(i%10)
					_ = world.QueryRange(ns, 22, 26, 51, 56)
					ops.Add(1)
				}
			})
		})
		b.Run("Parallel: Should return locations in the USA (9.8 m km²)", func(b *testing.B) {
			b.RunParallel(func(pb *testing.PB) {
				var ops atomic.Uint64
				for pb.Next() {
					i := int(ops.Load())
					ns := "ns" + strconv.Itoa(i%10)
					_ = world.QueryRange(ns, -25, 49, -124, -66)
					ops.Add(1)
				}
			})
		})
		b.Run("Parallel: Should return locations in all of africa (30 m km²) ", func(b *testing.B) {
			b.RunParallel(func(pb *testing.PB) {
				var ops atomic.Uint64
				for pb.Next() {
					i := int(ops.Load())
					ns := "ns" + strconv.Itoa(i%10)
					_ = world.QueryRange(ns, -34, 37, -17, 51.7)
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
					id := "locId" + strconv.Itoa(i)
					ns := "ns" + strconv.Itoa(i%10)
					world.Delete(ns, id)
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
				location, err := createRandomLocation(i)
				if err != nil {
					b.Fatalf("Error creating  random location: %v", err)
				}

				err = world.Save(location.Ns, location.Id, location.Lat, location.Lon)
				if err != nil {
					b.Fatalf("Error saving location: %v", err)
				}
			}
		})
	})

	b.Run("GetLocation", func(b *testing.B) {
		b.Run("Should return a location", func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				id := "locId" + strconv.Itoa(i)
				ns := "ns" + strconv.Itoa(i%10)

				world.GetLocation(ns, id)
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
	})

	b.Run("Delete", func(b *testing.B) {
		b.Run("Should delete a location", func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				id := "locId" + strconv.Itoa(i)
				ns := "ns" + strconv.Itoa(i%10)
				world.Delete(ns, id)
			}
		})
	})
}
