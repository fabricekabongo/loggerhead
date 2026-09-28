package stigma

import (
	"errors"
	"math"
)

// CellIndex maps lat/lon coordinates to deterministic CellIDs.
type CellIndex struct{}

// NewCellIndex constructs a CellIndex.
func NewCellIndex() *CellIndex {
	return &CellIndex{}
}

// CellForCoord converts a coordinate into a CellID for the given resolution.
// This is a placeholder grid; the API mirrors an H3-based implementation.
func (c *CellIndex) CellForCoord(lat, lon float64, res Resolution) (CellID, error) {
	if math.IsNaN(lat) || math.IsNaN(lon) || math.IsInf(lat, 0) || math.IsInf(lon, 0) {
		return 0, errors.New("invalid coordinate")
	}

	factor := math.Pow(2, float64(res)+1) // higher resolution => finer grid
	x := int64(math.Floor((lon + 180.0) * factor))
	y := int64(math.Floor((lat + 90.0) * factor))
	return CellID(uint64(y)<<32 | uint64(x&0xffffffff)), nil
}

// CellsInBBox returns all cells intersecting the bounding box for the resolution.
func (c *CellIndex) CellsInBBox(minLat, minLon, maxLat, maxLon float64, res Resolution) ([]CellID, error) {
	if minLat > maxLat || minLon > maxLon {
		return nil, errors.New("invalid bounding box")
	}

	factor := math.Pow(2, float64(res)+1)
	minX := int64(math.Floor((minLon + 180.0) * factor))
	maxX := int64(math.Floor((maxLon + 180.0) * factor))
	minY := int64(math.Floor((minLat + 90.0) * factor))
	maxY := int64(math.Floor((maxLat + 90.0) * factor))

	var cells []CellID
	for y := minY; y <= maxY; y++ {
		for x := minX; x <= maxX; x++ {
			cells = append(cells, CellID(uint64(y)<<32|uint64(x&0xffffffff)))
		}
	}

	return cells, nil
}
