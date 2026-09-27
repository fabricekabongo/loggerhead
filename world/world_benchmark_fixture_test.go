package world

import (
	"fmt"
	"math/rand"
	"strconv"
	"testing"
)

type benchmarkPoint struct {
	namespace string
	id        string
	lat       float64
	lon       float64
}

type benchmarkKey struct {
	namespace string
	id        string
}

type benchmarkFixture struct {
	world  *World
	points map[benchmarkKey]benchmarkPoint
}

const maxSequentialInsertOperations = 10_000

func validateSequentialInsertCount(count int) error {
	if count > maxSequentialInsertOperations {
		return fmt.Errorf("NewInsert/Sequential b.N=%d exceeds hard limit %d; run with -benchtime=1000x (or smaller)", count, maxSequentialInsertOperations)
	}
	return nil
}

func newBenchmarkFixture(points []benchmarkPoint) (*benchmarkFixture, error) {
	fixture := &benchmarkFixture{
		world:  NewWorld(),
		points: make(map[benchmarkKey]benchmarkPoint, len(points)),
	}
	for _, point := range points {
		key := benchmarkKey{namespace: point.namespace, id: point.id}
		if _, exists := fixture.points[key]; exists {
			return nil, fmt.Errorf("duplicate benchmark location (%q, %q)", point.namespace, point.id)
		}
		if err := fixture.world.Save(point.namespace, point.id, point.lat, point.lon); err != nil {
			return nil, fmt.Errorf("seed benchmark location (%q, %q): %w", point.namespace, point.id, err)
		}
		fixture.points[key] = point
	}
	return fixture, nil
}

func benchmarkID(index int) string {
	return "location-" + strconv.Itoa(index)
}

func benchmarkPoints(seed int64, count int) []benchmarkPoint {
	rng := rand.New(rand.NewSource(seed))
	points := make([]benchmarkPoint, count)
	for i := range points {
		points[i] = benchmarkPoint{
			namespace: "benchmark",
			id:        benchmarkID(i),
			lat:       -89 + rng.Float64()*178,
			lon:       -179 + rng.Float64()*358,
		}
	}
	return points
}

func localBenchmarkPoints(seed int64, count int) []benchmarkPoint {
	rng := rand.New(rand.NewSource(seed))
	points := make([]benchmarkPoint, count)
	for i := range points {
		points[i] = benchmarkPoint{
			namespace: "benchmark",
			id:        benchmarkID(i),
			lat:       12 + rng.Float64()*0.1,
			lon:       25 + rng.Float64()*0.1,
		}
	}
	return points
}

func movedBenchmarkPoint(point benchmarkPoint, updateRound int) benchmarkPoint {
	movement := float64(updateRound%64+1) * 0.00001
	point.lat += movement
	point.lon += movement
	return point
}

func TestBenchmarkPointsAreDeterministicAndUnique(t *testing.T) {
	first := benchmarkPoints(42, 64)
	second := benchmarkPoints(42, 64)
	otherSeed := benchmarkPoints(43, 64)

	if len(first) != 64 {
		t.Fatalf("got %d fixture points, want 64", len(first))
	}
	for i := range first {
		if first[i] != second[i] {
			t.Fatalf("point %d differs for repeated seed: %#v != %#v", i, first[i], second[i])
		}
		if first[i].namespace != "benchmark" {
			t.Fatalf("point %d namespace = %q, want benchmark", i, first[i].namespace)
		}
		if first[i].id != benchmarkID(i) {
			t.Fatalf("point %d ID = %q, want %q", i, first[i].id, benchmarkID(i))
		}
		if first[i] == otherSeed[i] {
			t.Fatalf("point %d did not change with seed", i)
		}
	}
}

func TestBenchmarkFixtureUsesCompositeIdentity(t *testing.T) {
	points := []benchmarkPoint{
		{namespace: "north", id: "same-id", lat: 12, lon: 25},
		{namespace: "south", id: "same-id", lat: -12, lon: -25},
	}
	fixture, err := newBenchmarkFixture(points)
	if err != nil {
		t.Fatalf("build fixture: %v", err)
	}

	for _, want := range points {
		got, found := fixture.world.GetLocation(want.namespace, want.id)
		if !found {
			t.Fatalf("missing (%q, %q)", want.namespace, want.id)
		}
		if got.Lat() != want.lat || got.Lon() != want.lon {
			t.Errorf("(%q, %q) coordinates = (%v, %v), want (%v, %v)", want.namespace, want.id, got.Lat(), got.Lon(), want.lat, want.lon)
		}
	}
	if got := len(fixture.world.QueryRange("north", -90, 90, -180, 180)); got != 1 {
		t.Fatalf("north population = %d, want 1", got)
	}

	_, err = newBenchmarkFixture([]benchmarkPoint{
		{namespace: "north", id: "duplicate", lat: 0, lon: 0},
		{namespace: "north", id: "duplicate", lat: 1, lon: 1},
	})
	if err == nil {
		t.Fatal("fixture accepted duplicate (namespace, id)")
	}
}

func TestSequentialInsertBenchmarkCountLimit(t *testing.T) {
	if err := validateSequentialInsertCount(maxSequentialInsertOperations); err != nil {
		t.Fatalf("hard limit rejected: %v", err)
	}
	if err := validateSequentialInsertCount(maxSequentialInsertOperations + 1); err == nil {
		t.Fatal("count above hard limit accepted")
	}
}

func BenchmarkWorldSave(b *testing.B) {
	b.Run("NewInsert/Sequential", func(b *testing.B) {
		const seed = 2401
		if err := validateSequentialInsertCount(b.N); err != nil {
			b.Fatal(err)
		}
		points := benchmarkPoints(seed, b.N)
		world := NewWorld()
		b.ReportAllocs()
		b.ResetTimer()
		for _, point := range points {
			if err := world.Save(point.namespace, point.id, point.lat, point.lon); err != nil {
				b.Fatalf("insert (%q, %q): %v", point.namespace, point.id, err)
			}
		}
		b.StopTimer()
		verifyBenchmarkWorld(b, world, points)
		b.ReportMetric(float64(seed), "seed")
		b.ReportMetric(float64(len(points)), "entities")
	})

	b.Run("FixedPopulationLocalUpdate/Sequential", func(b *testing.B) {
		const (
			seed       = 2402
			population = 256
		)
		initial := localBenchmarkPoints(seed, population)
		fixture, err := newBenchmarkFixture(initial)
		if err != nil {
			b.Fatalf("build fixed-population fixture: %v", err)
		}
		b.ReportAllocs()
		b.ResetTimer()
		for iteration := 0; iteration < b.N; iteration++ {
			index := iteration % population
			point := movedBenchmarkPoint(initial[index], iteration/population)
			if err := fixture.world.Save(point.namespace, point.id, point.lat, point.lon); err != nil {
				b.Fatalf("update (%q, %q): %v", point.namespace, point.id, err)
			}
		}
		b.StopTimer()

		final := append([]benchmarkPoint(nil), initial...)
		for index := 0; index < population && index < b.N; index++ {
			lastIteration := b.N - 1 - (b.N-1-index)%population
			final[index] = movedBenchmarkPoint(initial[index], lastIteration/population)
		}
		verifyBenchmarkWorld(b, fixture.world, final)
		b.ReportMetric(float64(seed), "seed")
		b.ReportMetric(float64(population), "entities")
	})
}

func verifyBenchmarkWorld(b *testing.B, world *World, expected []benchmarkPoint) {
	b.Helper()
	for _, point := range expected {
		got, found := world.GetLocation(point.namespace, point.id)
		if !found {
			b.Fatalf("missing benchmark location (%q, %q)", point.namespace, point.id)
		}
		if got.Lat() != point.lat || got.Lon() != point.lon {
			b.Fatalf("location (%q, %q) = (%v, %v), want (%v, %v)", point.namespace, point.id, got.Lat(), got.Lon(), point.lat, point.lon)
		}
	}
	if got := len(world.QueryRange("benchmark", -90, 90, -180, 180)); got != len(expected) {
		b.Fatalf("benchmark population = %d, want %d", got, len(expected))
	}
}
