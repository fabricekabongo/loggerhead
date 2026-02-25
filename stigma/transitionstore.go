package stigma

import "time"

type transitionMetrics struct {
	agg       TransitionAggregate
	travelSum time.Duration
}

// TransitionStore aggregates movements between cells.
type TransitionStore struct {
	transitions map[TransitionKey]*transitionMetrics
}

// NewTransitionStore constructs the store.
func NewTransitionStore() *TransitionStore {
	return &TransitionStore{transitions: make(map[TransitionKey]*transitionMetrics)}
}

// Update updates the aggregate for a transition.
func (s *TransitionStore) Update(from, to CellID, res Resolution, delta time.Duration, ts time.Time) TransitionAggregate {
	key := TransitionKey{FromCell: from, ToCell: to}
	t, ok := s.transitions[key]
	if !ok {
		t = &transitionMetrics{agg: TransitionAggregate{FromCell: from, ToCell: to, Resolution: res}}
		s.transitions[key] = t
	}

	t.agg.TransitionCount++
	t.travelSum += delta
	avg := t.travelSum / time.Duration(t.agg.TransitionCount)
	t.agg.AvgTravelTime = avg
	if t.agg.MinTravelTime == 0 || delta < t.agg.MinTravelTime {
		t.agg.MinTravelTime = delta
	}
	if delta > t.agg.MaxTravelTime {
		t.agg.MaxTravelTime = delta
	}
	if ts.After(t.agg.LastSeen) {
		t.agg.LastSeen = ts
	}

	return t.agg
}

// All returns all transition aggregates.
func (s *TransitionStore) All() []TransitionAggregate {
	res := make([]TransitionAggregate, 0, len(s.transitions))
	for _, t := range s.transitions {
		res = append(res, t.agg)
	}
	return res
}

// FilterByCells returns transitions whose endpoints are in the provided set.
func (s *TransitionStore) FilterByCells(cells map[CellID]struct{}, start, end *time.Time) []TransitionAggregate {
	res := make([]TransitionAggregate, 0)
	for key, t := range s.transitions {
		if _, ok := cells[key.FromCell]; !ok {
			continue
		}
		if _, ok := cells[key.ToCell]; !ok {
			continue
		}
		if start != nil && t.agg.LastSeen.Before(*start) {
			continue
		}
		if end != nil && t.agg.LastSeen.After(*end) {
			// transitions are coarse filtered; keep if last seen before end
			// otherwise skip
			continue
		}
		res = append(res, t.agg)
	}
	return res
}
