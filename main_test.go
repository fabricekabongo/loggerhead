package main

import (
	"fmt"
	"github.com/fabricekabongo/loggerhead/world"
	"math/rand"
	"net"
	"strconv"
	"sync/atomic"
	"testing"
)

func CreateRandomLocation(seed int) (*world.Location, error) {
	lat := -90.0 + rand.Float64()*(90.0+90.0)
	lon := -180.0 + rand.Float64()*(180.0+180.0)
	id := "locId" + strconv.Itoa(seed)
	ns := "ns" + strconv.Itoa(seed%10)

	return world.NewLocation(ns, id, lat, lon)
}

func BenchmarkDatabase(b *testing.B) {
	main()
	b.Run("Write to DB", func(b *testing.B) {
		count := atomic.Uint64{}
		connPool := make([]net.Conn, 0, 20)
		for i := 0; i < 20; i++ {
			conn, err := net.Dial("tcp", "localhost:19999")
			if err != nil {
				b.Error("Failed to connect to the database: ", err)
			}
			connPool = append(connPool, conn)
		}

		defer func() {
			for _, conn := range connPool {
				err := conn.Close()
				if err != nil {
					b.Error("Failed to close the connection: ", err)
				}
			}
		}()

		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				i := int(count.Load())
				count.Add(1)
				conn := connPool[i%20]

				loc, err := CreateRandomLocation(int(i) % 1000000)
				if err != nil {
					b.Error("Failed to create a random location: ", err)
					return
				}
				command := fmt.Sprintf("SAVE %s %s %f %f\n", loc.Ns(), loc.Id(), loc.Lat(), loc.Lon())
				_, err = conn.Write([]byte(command))
				if err != nil {
					b.Error("Failed to save location: ", err)
				}
			}
		})
	})

}
