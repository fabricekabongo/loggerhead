package stigma

import (
	"errors"
	"fmt"
	"math"
	"time"
)

// EngineConfig configures the stigma engine.
type EngineConfig struct {
	DefaultResolution Resolution
	MinResolution     Resolution
	MaxResolution     Resolution

	WALPath       string
	SnapshotPath  string
	FlushInterval time.Duration
}

// Engine aggregates ingest, query, and persistence logic.
// Engine is NOT safe for concurrent use by multiple goroutines in v1.
type Engine struct {
	cfg         EngineConfig
	cellIndex   *CellIndex
	nodes       *NodeStore
	transitions *TransitionStore
	states      *EntityStateStore
	wal         *WAL
}

// NewEngine creates a new Engine and replays any WAL if present.
func NewEngine(cfg EngineConfig) (*Engine, error) {
	e := &Engine{
		cfg:         cfg,
		cellIndex:   NewCellIndex(),
		nodes:       NewNodeStore(),
		transitions: NewTransitionStore(),
		states:      NewEntityStateStore(),
	}

	if cfg.WALPath != "" {
		wal, err := OpenWAL(cfg.WALPath)
		if err != nil {
			return nil, fmt.Errorf("open wal: %w", err)
		}
		e.wal = wal
		if err := e.replayWAL(); err != nil {
			return nil, err
		}
	}

	return e, nil
}

// Close flushes and closes any persistence layers.
func (e *Engine) Close() error {
	if e.wal != nil {
		return e.wal.Close()
	}
	return nil
}

func (e *Engine) clampResolution(res Resolution) Resolution {
	if e.cfg.MinResolution != 0 && res < e.cfg.MinResolution {
		return e.cfg.MinResolution
	}
	if e.cfg.MaxResolution != 0 && res > e.cfg.MaxResolution {
		return e.cfg.MaxResolution
	}
	return res
}

// IngestSample processes a single sample.
func (e *Engine) IngestSample(sample LocationSample) error {
	return e.ingest([]LocationSample{sample}, true)
}

func validateSample(s LocationSample) error {
	if s.EntityID == "" {
		return errors.New("entity id is required")
	}
	if math.IsNaN(s.Lat) || math.IsNaN(s.Lon) || s.Lat < -90 || s.Lat > 90 || s.Lon < -180 || s.Lon > 180 {
		return errors.New("invalid coordinates")
	}
	if s.Timestamp.IsZero() {
		return errors.New("timestamp is required")
	}
	return nil
}

// IngestBatch processes multiple samples efficiently.
func (e *Engine) IngestBatch(samples []LocationSample) error {
	return e.ingest(samples, true)
}

func (e *Engine) ingest(samples []LocationSample, logToWAL bool) error {
	for i := range samples {
		if err := validateSample(samples[i]); err != nil {
			return err
		}
	}

	if logToWAL && e.wal != nil {
		if err := e.wal.Append(samples); err != nil {
			return err
		}
	}

	res := e.cfg.DefaultResolution
	res = e.clampResolution(res)

	for i := range samples {
		sample := samples[i]
		cellID, err := e.cellIndex.CellForCoord(sample.Lat, sample.Lon, res)
		if err != nil {
			return err
		}

		e.nodes.Update(cellID, res, sample)

		prev, ok := e.states.Get(sample.EntityID)
		if ok {
			if prev.LastCell != cellID {
				delta := sample.Timestamp.Sub(prev.LastTime)
				e.transitions.Update(prev.LastCell, cellID, res, delta, sample.Timestamp)
			}
		}
		e.states.Update(sample.EntityID, cellID, sample.Timestamp)
	}

	return nil
}

// QueryStigmaMap returns aggregates for the query bounds.
func (e *Engine) QueryStigmaMap(q StigmaMapQuery) (StigmaMap, error) {
	res := e.cfg.DefaultResolution
	if q.Resolution != nil {
		res = e.clampResolution(*q.Resolution)
	} else {
		res = e.clampResolution(res)
	}

	cells, err := e.cellIndex.CellsInBBox(q.MinLat, q.MinLon, q.MaxLat, q.MaxLon, res)
	if err != nil {
		return StigmaMap{}, err
	}

	cellSet := make(map[CellID]struct{}, len(cells))
	for _, c := range cells {
		cellSet[c] = struct{}{}
	}

	nodes := e.nodes.TimeFiltered(q.StartTime, q.EndTime)
	filteredNodes := make([]NodeAggregate, 0, len(nodes))
	for _, n := range nodes {
		if _, ok := cellSet[n.CellID]; !ok {
			continue
		}
		if q.MinSampleCount > 0 && n.SampleCount < q.MinSampleCount {
			continue
		}
		filteredNodes = append(filteredNodes, n)
	}

	result := StigmaMap{Nodes: filteredNodes}
	if q.IncludeTransitions {
		result.Transitions = e.transitions.FilterByCells(cellSet, q.StartTime, q.EndTime)
	}

	return result, nil
}

// GetCellStats returns aggregate for a single cell.
func (e *Engine) GetCellStats(cellID CellID) (NodeAggregate, bool) {
	return e.nodes.Get(cellID)
}

// GetEntityPathSummary returns visitation summary for an entity.
func (e *Engine) GetEntityPathSummary(entityID string) (EntityPathSummary, bool) {
	return e.states.PathSummary(entityID)
}

func (e *Engine) replayWAL() error {
	entries, err := e.wal.ReadAll()
	if err != nil {
		return err
	}
	return e.ingest(entries, false)
}
