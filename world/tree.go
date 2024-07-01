package world

import (
	"errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"math"
	"sync"
)

var (
	TreeErrLocationNil         = errors.New("insertion failed because location is nil")
	TreeErrLocationOutOfBounds = errors.New("insertion failed because location is out of bounds")
	treeDivision               = promauto.NewCounter(prometheus.CounterOpts{
		Name: "loggerhead_world_tree_division",
		Help: "The number of time the tree divides itself",
	})
)

type QuadTree struct {
	Root *TreeNode
}

type TreeNode struct {
	NE        *TreeNode
	NW        *TreeNode
	SE        *TreeNode
	SW        *TreeNode
	Lat1      float64
	Lat2      float64
	Lon1      float64
	Lon2      float64
	mu        sync.RWMutex
	Objects   map[string]*Location
	Capacity  int
	IsDivided bool
}

func NewQuadTree(lat1 float64, lat2 float64, lon1 float64, lon2 float64) *QuadTree {
	qt := &QuadTree{
		Root: &TreeNode{
			IsDivided: false,
			Capacity:  500,
			Lat1:      lat1,
			Lat2:      lat2,
			Lon1:      lon1,
			Lon2:      lon2,
			Objects:   make(map[string]*Location),
		},
	}

	qt.Root.ForceDivide(5)

	return qt
}

func (q *QuadTree) Insert(location *Location) error {
	if location == nil {
		return TreeErrLocationNil
	}
	return q.Root.insert(location)
}

func NewTreeNode(lat1 float64, lat2 float64, lon1 float64, lon2 float64, capacity int) *TreeNode {
	return &TreeNode{
		IsDivided: false,
		Capacity:  capacity,
		Lat1:      lat1,
		Lat2:      lat2,
		Lon1:      lon1,
		Lon2:      lon2,
		Objects:   make(map[string]*Location),
	}
}

func (n *TreeNode) insert(location *Location) error {
	if location == nil {
		panic("Location is nil. It should never reach this point")
	}
	// If the location is not within the bound, return
	if !(n.Lon1 <= location.lon && location.lon <= n.Lon2 && n.Lat1 <= location.lat && location.lat <= n.Lat2) {
		return TreeErrLocationOutOfBounds
	}

	if n.IsDivided {
		err := n.NW.insert(location)
		if err != nil {
			err = n.NE.insert(location)
			if err != nil {
				err = n.SW.insert(location)
				if err != nil {
					err = n.SE.insert(location)
				}
			}
		}

		return nil
	}

	// If the node is not divided, insert the location into the node
	n.mu.Lock()
	if location.Node != nil && location.Node != n {
		location.Node.Delete(location.Id())
	}
	n.Objects[location.Id()] = location
	location.Node = n
	n.mu.Unlock()

	if len(n.Objects) > n.Capacity {
		n.divide()
	}

	return nil
}

func (n *TreeNode) Delete(id string) {
	if n.IsDivided {
		n.NE.Delete(id)
		n.NW.Delete(id)
		n.SE.Delete(id)
		n.SW.Delete(id)
		return
	}

	n.mu.Lock()
	delete(n.Objects, id)
	n.mu.Unlock()
}

func (n *TreeNode) divide() {
	defer treeDivision.Inc()
	n.SE = NewTreeNode(n.Lat1, (n.Lat1+n.Lat2)/2, (n.Lon1+n.Lon2)/2, n.Lon2, n.Capacity)
	n.SW = NewTreeNode(n.Lat1, (n.Lat1+n.Lat2)/2, n.Lon1, (n.Lon1+n.Lon2)/2, n.Capacity)
	n.NE = NewTreeNode((n.Lat1+n.Lat2)/2, n.Lat2, (n.Lon1+n.Lon2)/2, n.Lon2, n.Capacity)
	n.NW = NewTreeNode((n.Lat1+n.Lat2)/2, n.Lat2, n.Lon1, (n.Lon1+n.Lon2)/2, n.Capacity)

	n.IsDivided = true
	n.mu.Lock()
	for i, location := range n.Objects {
		if location == nil {
			panic("The Node is holding nil location. weird don't you think?. Location index: " + i)
		}
		delete(n.Objects, location.Id())
		location.Node = nil

		_ = n.insert(location)
	}
	n.mu.Unlock()

	// TODO: I want to set Objects as nil but some test fail, maybe running to fast in a concurrent manner. Fix this so we don't waste memory
	n.mu.Lock()
	n.Objects = map[string]*Location{}
	n.mu.Unlock()
}

func (q *QuadTree) reBalance() {
	// TODO: Implement rebalancing
}

func rectangleOverlap(lat1 float64, lat2 float64, lon1 float64, lon2 float64, lat3 float64, lat4 float64, lon3 float64, lon4 float64) bool {
	return math.Max(lat1, lat3) < math.Min(lat2, lat4) && math.Max(lon1, lon3) < math.Min(lon2, lon4)
}

func (n *TreeNode) QueryRange(lat1 float64, lat2 float64, lon1 float64, lon2 float64) []*Location {

	var locations []*Location

	if !rectangleOverlap(n.Lat1, n.Lat2, n.Lon1, n.Lon2, lat1, lat2, lon1, lon2) {
		return locations
	}

	if !n.IsDivided {

		for _, location := range n.Objects {
			if location.Lon() >= lon1 && location.Lon() <= lon2 && location.Lat() >= lat1 && location.Lat() <= lat2 {
				locations = append(locations, location)
			}
		}

		return locations
	}

	if rectangleOverlap(n.NE.Lat1, n.NE.Lat2, n.NE.Lon1, n.NE.Lon2, lat1, lat2, lon1, lon2) {
		locations = append(locations, n.NE.QueryRange(lat1, lat2, lon1, lon2)...)
	}

	if rectangleOverlap(n.NW.Lat1, n.NW.Lat2, n.NW.Lon1, n.NW.Lon2, lat1, lat2, lon1, lon2) {
		locations = append(locations, n.NW.QueryRange(lat1, lat2, lon1, lon2)...)
	}

	if rectangleOverlap(n.SE.Lat1, n.SE.Lat2, n.SE.Lon1, n.SE.Lon2, lat1, lat2, lon1, lon2) {
		locations = append(locations, n.SE.QueryRange(lat1, lat2, lon1, lon2)...)
	}

	if rectangleOverlap(n.SW.Lat1, n.SW.Lat2, n.SW.Lon1, n.SW.Lon2, lat1, lat2, lon1, lon2) {
		locations = append(locations, n.SW.QueryRange(lat1, lat2, lon1, lon2)...)
	}

	return locations
}

func (n *TreeNode) ForceDivide(level int) {
	if level == 0 {
		return
	}

	n.divide()
	level--

	n.NE.ForceDivide(level)
	n.NW.ForceDivide(level)
	n.SE.ForceDivide(level)
	n.SW.ForceDivide(level)
}
