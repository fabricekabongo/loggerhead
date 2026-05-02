package stigma

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestIngestSingleSampleUpdatesNode(t *testing.T) {
	engine, err := NewEngine(EngineConfig{DefaultResolution: 1})
	if err != nil {
		t.Fatalf("failed to create engine: %v", err)
	}
	now := time.Now().UTC()
	sample := LocationSample{EntityID: "e1", Lat: 1.0, Lon: 1.0, Timestamp: now, SpeedMps: 2.5}
	if err := engine.IngestSample(sample); err != nil {
		t.Fatalf("ingest sample: %v", err)
	}

	cell, _ := engine.cellIndex.CellForCoord(sample.Lat, sample.Lon, engine.cfg.DefaultResolution)
	agg, ok := engine.GetCellStats(cell)
	if !ok {
		t.Fatalf("expected cell aggregate present")
	}
	if agg.SampleCount != 1 || agg.UniqueEntities != 1 {
		t.Fatalf("unexpected counts: %+v", agg)
	}
	if agg.MaxSpeedMps != sample.SpeedMps || agg.AvgSpeedMps == 0 {
		t.Fatalf("unexpected speed aggregates: %+v", agg)
	}
}

func TestTransitionComputedAcrossCells(t *testing.T) {
	engine, _ := NewEngine(EngineConfig{DefaultResolution: 1})
	now := time.Now().UTC()
	samples := []LocationSample{
		{EntityID: "e1", Lat: 0, Lon: 0, Timestamp: now},
		{EntityID: "e1", Lat: 10, Lon: 10, Timestamp: now.Add(10 * time.Second)},
	}
	if err := engine.IngestBatch(samples); err != nil {
		t.Fatalf("ingest batch: %v", err)
	}

	transitions := engine.transitions.All()
	if len(transitions) != 1 {
		t.Fatalf("expected one transition, got %d", len(transitions))
	}
	tr := transitions[0]
	if tr.TransitionCount != 1 || tr.AvgTravelTime != 10*time.Second || tr.MinTravelTime != 10*time.Second || tr.MaxTravelTime != 10*time.Second {
		t.Fatalf("unexpected transition aggregate: %+v", tr)
	}
}

func TestQueryStigmaMapFiltersBySampleCount(t *testing.T) {
	engine, _ := NewEngine(EngineConfig{DefaultResolution: 1})
	now := time.Now().UTC()
	_ = engine.IngestBatch([]LocationSample{
		{EntityID: "a", Lat: 0, Lon: 0, Timestamp: now},
		{EntityID: "a", Lat: 0, Lon: 0, Timestamp: now.Add(time.Second)},
		{EntityID: "b", Lat: 1, Lon: 1, Timestamp: now},
	})

	query := StigmaMapQuery{MinLat: -1, MinLon: -1, MaxLat: 2, MaxLon: 2, MinSampleCount: 2}
	result, err := engine.QueryStigmaMap(query)
	if err != nil {
		t.Fatalf("query failed: %v", err)
	}

	if len(result.Nodes) != 1 {
		t.Fatalf("expected one node after filtering, got %d", len(result.Nodes))
	}
}

func TestWALReplayRestoresAggregates(t *testing.T) {
	dir := t.TempDir()
	walPath := filepath.Join(dir, "wal.log")

	cfg := EngineConfig{DefaultResolution: 1, WALPath: walPath}
	engine, err := NewEngine(cfg)
	if err != nil {
		t.Fatalf("create engine: %v", err)
	}
	now := time.Now().UTC()
	if err := engine.IngestSample(LocationSample{EntityID: "x", Lat: 1, Lon: 1, Timestamp: now}); err != nil {
		t.Fatalf("ingest with wal: %v", err)
	}
	_ = engine.Close()

	// reopen and ensure sample is replayed
	engine2, err := NewEngine(cfg)
	if err != nil {
		t.Fatalf("recreate engine: %v", err)
	}
	defer engine2.Close()

	cell, _ := engine2.cellIndex.CellForCoord(1, 1, cfg.DefaultResolution)
	if agg, ok := engine2.GetCellStats(cell); !ok || agg.SampleCount != 1 {
		t.Fatalf("expected aggregate restored, got %#v", agg)
	}

	// ensure wal file exists
	if _, err := os.Stat(walPath); err != nil {
		t.Fatalf("wal file missing: %v", err)
	}
}

func TestInvalidSamplesReturnErrors(t *testing.T) {
	engine, _ := NewEngine(EngineConfig{DefaultResolution: 1})
	err := engine.IngestSample(LocationSample{})
	if err == nil {
		t.Fatalf("expected validation error")
	}

	err = engine.IngestSample(LocationSample{EntityID: "a", Lat: -91, Lon: 0, Timestamp: time.Now()})
	if err == nil {
		t.Fatalf("expected invalid coordinate error")
	}
}
