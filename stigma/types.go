package stigma

import "time"

// LocationSample represents one geospatial observation for an entity.
type LocationSample struct {
	EntityID   string            // required
	Lat        float64           // degrees
	Lon        float64           // degrees
	Timestamp  time.Time         // required, UTC
	SpeedMps   float32           // optional, meters per second (0 if unknown)
	HeadingDeg float32           // optional, 0-360 (0 if unknown)
	Attrs      map[string]string // optional, extra tags (version, appID, etc.)
}

// CellID references a grid cell.
type CellID uint64

// Resolution represents the granularity of the grid (similar to H3 resolutions).
type Resolution int

// NodeAggregate holds aggregate metrics for a cell.
type NodeAggregate struct {
	CellID     CellID
	Resolution Resolution

	// Counts
	SampleCount    uint64 // total samples in this cell
	UniqueEntities uint64 // distinct entities seen

	// Temporal stats
	FirstSeen time.Time
	LastSeen  time.Time

	// Motion stats
	AvgSpeedMps float32 // average speed of samples in cell
	MaxSpeedMps float32 // max speed observed
}

// TransitionKey identifies a movement between two cells.
type TransitionKey struct {
	FromCell CellID
	ToCell   CellID
}

// TransitionAggregate stores metrics for movements between two cells.
type TransitionAggregate struct {
	FromCell   CellID
	ToCell     CellID
	Resolution Resolution

	TransitionCount uint64
	AvgTravelTime   time.Duration
	MinTravelTime   time.Duration
	MaxTravelTime   time.Duration
	LastSeen        time.Time
}

// StigmaMapQuery defines the parameters for querying aggregates.
type StigmaMapQuery struct {
	// Spatial
	MinLat float64
	MinLon float64
	MaxLat float64
	MaxLon float64

	// When empty, use engine default
	Resolution *Resolution

	// Temporal filter (optional)
	StartTime *time.Time
	EndTime   *time.Time

	// Flags to control what to include
	IncludeTransitions bool
	MinSampleCount     uint64 // filter out low-density nodes
}

// StigmaMap is the result of a query.
type StigmaMap struct {
	Nodes       []NodeAggregate
	Transitions []TransitionAggregate
}

// EntityPathSummary summarizes one entity's historical visits.
type EntityPathSummary struct {
	EntityID string

	FirstSeen time.Time
	LastSeen  time.Time
	Cells     []CellID // distinct visited cells (order not guaranteed in v1)
}
