package stigma

import "time"

type entityState struct {
	EntityID  string
	LastCell  CellID
	LastTime  time.Time
	FirstTime time.Time
	visited   map[CellID]struct{}
}

// EntityStateStore tracks last seen state per entity.
type EntityStateStore struct {
	states map[string]*entityState
}

// NewEntityStateStore constructs the store.
func NewEntityStateStore() *EntityStateStore {
	return &EntityStateStore{states: make(map[string]*entityState)}
}

// Update records the latest cell and timestamp for an entity.
func (s *EntityStateStore) Update(entityID string, cell CellID, ts time.Time) *entityState {
	state, ok := s.states[entityID]
	if !ok {
		state = &entityState{EntityID: entityID, FirstTime: ts, visited: make(map[CellID]struct{})}
		s.states[entityID] = state
	}
	state.LastCell = cell
	state.LastTime = ts
	state.visited[cell] = struct{}{}
	return state
}

// Get returns the current state for an entity.
func (s *EntityStateStore) Get(entityID string) (*entityState, bool) {
	st, ok := s.states[entityID]
	return st, ok
}

// PathSummary returns a snapshot of path information.
func (s *EntityStateStore) PathSummary(entityID string) (EntityPathSummary, bool) {
	st, ok := s.states[entityID]
	if !ok {
		return EntityPathSummary{}, false
	}

	cells := make([]CellID, 0, len(st.visited))
	for cell := range st.visited {
		cells = append(cells, cell)
	}

	return EntityPathSummary{
		EntityID:  entityID,
		FirstSeen: st.FirstTime,
		LastSeen:  st.LastTime,
		Cells:     cells,
	}, true
}
