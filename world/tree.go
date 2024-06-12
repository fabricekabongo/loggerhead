package world

type QuadTree struct {
	Root *TreeNode
}

type TreeNode struct {
	NE   *TreeNode
	NW   *TreeNode
	SE   *TreeNode
	SW   *TreeNode
	Lat1 float64
	Lat2 float64
	Lon1 float64
	Lon2 float64

	Objects   []*Location
	Capacity  int
	IsDivided bool
}

func NewQuadTree(lat1 float64, lat2 float64, lon1 float64, lon2 float64) *QuadTree {
	return &QuadTree{
		Root: &TreeNode{
			IsDivided: false,
			Capacity:  2000,
			Lat1:      lat1,
			Lat2:      lat2,
			Lon1:      lon1,
			Lon2:      lon2,
		},
	}
}

func (q *QuadTree) Insert(location *Location) {
	q.Root.insert(location)
}

func (n *TreeNode) insert(location *Location) bool {
	// If the location is not within the bound, return
	if !(n.Lon1 < location.Lon && n.Lon2 > location.Lon && n.Lat1 < location.Lat && n.Lat2 > location.Lat) {
		return false
	}

	if n.IsDivided {
		if n.NE.insert(location) || n.NW.insert(location) || n.SE.insert(location) || n.SW.insert(location) {
			return true
		}
	}

	n.Objects = append(n.Objects, location)
	if len(n.Objects) > n.Capacity {
		n.divide()
	}

	return true
}

func (n *TreeNode) Delete(id string) {
	if n.IsDivided {
		n.NE.Delete(id)
		n.NW.Delete(id)
		n.SE.Delete(id)
		n.SW.Delete(id)
	}

	for i, location := range n.Objects {
		if location.Id == id {
			n.Objects = append(n.Objects[:i], n.Objects[i+1:]...)
		}
	}
}

func (n *TreeNode) divide() {
	n.NE = &TreeNode{
		IsDivided: false,
		Capacity:  n.Capacity,
		Lat1:      n.Lat1,
		Lat2:      (n.Lat1 + n.Lat2) / 2,
		Lon1:      n.Lon1,
		Lon2:      (n.Lon1 + n.Lon2) / 2,
	}

	n.NW = &TreeNode{
		IsDivided: false,
		Capacity:  n.Capacity,
		Lat1:      n.Lat1,
		Lat2:      (n.Lat1 + n.Lat2) / 2,
		Lon1:      (n.Lon1 + n.Lon2) / 2,
		Lon2:      n.Lon2,
	}

	n.SE = &TreeNode{
		IsDivided: false,
		Capacity:  n.Capacity,
		Lat1:      (n.Lat1 + n.Lat2) / 2,
		Lat2:      n.Lat2,
		Lon1:      n.Lon1,
		Lon2:      (n.Lon1 + n.Lon2) / 2,
	}

	n.SW = &TreeNode{
		IsDivided: false,
		Capacity:  n.Capacity,
		Lat1:      (n.Lat1 + n.Lat2) / 2,
		Lat2:      n.Lat2,
		Lon1:      (n.Lon1 + n.Lon2) / 2,
		Lon2:      n.Lon2,
	}

	for _, location := range n.Objects {
		if n.NE.insert(location) || n.NW.insert(location) || n.SE.insert(location) || n.SW.insert(location) {
			continue
		}
	}

	n.Objects = nil
	n.IsDivided = true
}

func (q *QuadTree) reBalance() {

}
func (n *TreeNode) QueryRange(lat1 float64, lat2 float64, lon1 float64, lon2 float64) []*Location {
	var locations []*Location

	if n.Lon1 > lon2 || n.Lon2 < lon1 || n.Lat1 > lat2 || n.Lat2 < lat1 {
		return locations
	}

	if n.IsDivided {
		locations = append(locations, n.NE.QueryRange(lat1, lat2, lon1, lon2)...)
		locations = append(locations, n.NW.QueryRange(lat1, lat2, lon1, lon2)...)
		locations = append(locations, n.SE.QueryRange(lat1, lat2, lon1, lon2)...)
		locations = append(locations, n.SW.QueryRange(lat1, lat2, lon1, lon2)...)

		return locations
	}

	for _, location := range n.Objects {
		if location.Lon >= lon1 && location.Lon <= lon2 && location.Lat >= lat1 && location.Lat <= lat2 {
			locations = append(locations, location)
		}
	}

	return locations
}
