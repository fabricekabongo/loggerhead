package server

import (
	"github.com/ataul443/memnet"
	"github.com/fabricekabongo/loggerhead/query"
	"github.com/fabricekabongo/loggerhead/subscription"
	"github.com/fabricekabongo/loggerhead/world"
	"math/rand/v2"
	"strconv"
	"testing"
	"time"
)

func CreateRandomLocation(seed int) (string, string, float64, float64) {
	lat := -90.0 + rand.Float64()*(90.0+90.0)
	lon := -180.0 + rand.Float64()*(180.0+180.0)
	id := strconv.Itoa(seed)

	return "1", id, lat, lon
}

func BenchmarkListener(b *testing.B) {
	b.Run("NewListener", func(b *testing.B) {
		netListener, err := memnet.Listen(1, 4096, "bob")
		if err != nil {
			b.Fatal("Failed to create a memnet listener: ", err)
		}
		w := world.NewWorld()
		engine := query.NewQueryEngine(w, subscription.NewManager())
		l := NewListener(19999, 100, 20*time.Second, engine)

		go l.Handler.listen(netListener)
		time.Sleep(5 * time.Second)

		conn, err := netListener.Dial()
		if err != nil {
			b.Fatal("Failed to dial the connection: ", err)
		}
		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			ns, id, lat, lon := CreateRandomLocation(i)
			_, err := conn.Write([]byte("SAVE " + ns + " " + id + " " + strconv.FormatFloat(lat, 'f', -1, 64) + " " + strconv.FormatFloat(lon, 'f', -1, 64) + "\n"))
			if err != nil {
				b.Fatal("Failed to write to the connection: ", err)
			}
		}
	})
}
