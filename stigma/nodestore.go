package stigma

import "time"

type nodeMetrics struct {
	agg          NodeAggregate
	entityIDs    map[string]struct{}
	speedSum     float64
	speedSamples uint64
}

// NodeStore aggregates samples per cell.
type NodeStore struct {
	nodes map[CellID]*nodeMetrics
}

// NewNodeStore constructs NodeStore.
func NewNodeStore() *NodeStore {
	return &NodeStore{nodes: make(map[CellID]*nodeMetrics)}
}

// Update updates aggregates for a cell and entity.
func (s *NodeStore) Update(cell CellID, res Resolution, sample LocationSample) NodeAggregate {
	n, ok := s.nodes[cell]
	if !ok {
		n = &nodeMetrics{
			agg:       NodeAggregate{CellID: cell, Resolution: res},
			entityIDs: make(map[string]struct{}),
		}
		s.nodes[cell] = n
	}

	n.agg.SampleCount++
	if _, seen := n.entityIDs[sample.EntityID]; !seen {
		n.entityIDs[sample.EntityID] = struct{}{}
		n.agg.UniqueEntities = uint64(len(n.entityIDs))
	}

	if n.agg.FirstSeen.IsZero() || sample.Timestamp.Before(n.agg.FirstSeen) {
		n.agg.FirstSeen = sample.Timestamp
	}
	if sample.Timestamp.After(n.agg.LastSeen) {
		n.agg.LastSeen = sample.Timestamp
	}

	if sample.SpeedMps > 0 {
		n.speedSum += float64(sample.SpeedMps)
		n.speedSamples++
		n.agg.AvgSpeedMps = float32(n.speedSum / float64(n.speedSamples))
		if sample.SpeedMps > n.agg.MaxSpeedMps {
			n.agg.MaxSpeedMps = sample.SpeedMps
		}
	}

	return n.agg
}

// Get returns the aggregate for a cell.
func (s *NodeStore) Get(cell CellID) (NodeAggregate, bool) {
	n, ok := s.nodes[cell]
	if !ok {
		return NodeAggregate{}, false
	}
	return n.agg, true
}

// All returns all aggregates.
func (s *NodeStore) All() []NodeAggregate {
	res := make([]NodeAggregate, 0, len(s.nodes))
	for _, n := range s.nodes {
		res = append(res, n.agg)
	}
	return res
}

// PruneMinSample filters nodes by sample count.
func (s *NodeStore) PruneMinSample(min uint64) []NodeAggregate {
	res := make([]NodeAggregate, 0)
	for _, n := range s.nodes {
		if n.agg.SampleCount >= min {
			res = append(res, n.agg)
		}
	}
	return res
}

// TimeFiltered returns nodes matching coarse time criteria.
func (s *NodeStore) TimeFiltered(start, end *time.Time) []NodeAggregate {
	res := make([]NodeAggregate, 0)
	for _, n := range s.nodes {
		if start != nil && n.agg.LastSeen.Before(*start) {
			continue
		}
		if end != nil && n.agg.FirstSeen.After(*end) {
			continue
		}
		res = append(res, n.agg)
	}
	return res
}
