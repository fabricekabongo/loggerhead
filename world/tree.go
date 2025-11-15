package world

import (
	"errors"
	"math"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	treeDivision = promauto.NewCounter(prometheus.CounterOpts{
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

func (node *TreeNode) insert(location *Location) error {
	if location == nil {
		return TreeErrLocationNil
	}
	// If the location is not within the bound, return
	if !(node.Lon1 <= location.lon && location.lon <= node.Lon2 && node.Lat1 <= location.lat && location.lat <= node.Lat2) {
		return TreeErrLocationOutOfBounds
	}

	node.mu.Lock()
	defer node.mu.Unlock()

	if node.IsDivided {
		err := node.NW.insert(location)
		if errors.Is(err, TreeErrLocationOutOfBounds) {
			err = node.NE.insert(location)
			if errors.Is(err, TreeErrLocationOutOfBounds) {
				err = node.SW.insert(location)
				if errors.Is(err, TreeErrLocationOutOfBounds) {
					err = node.SE.insert(location)
				}
			}
		}

		return err
	}

	// If the node is not divided, insert the location into the node
	if location.Node != nil && location.Node != node {
		location.Node.Delete(location.Id())
	}
	node.Objects[location.Id()] = location
	location.SetNode(node)

	if len(node.Objects) > node.Capacity {
		node.divide()
	}

	return nil
}

func (node *TreeNode) Delete(id string) {
	node.mu.Lock()
	delete(node.Objects, id)
	node.mu.Unlock()
}

func (node *TreeNode) divide() {
	defer treeDivision.Inc()
	node.mu.Lock()
	defer node.mu.Unlock()

	if node.IsDivided {
		return
	}

	node.SE = NewTreeNode(node.Lat1, (node.Lat1+node.Lat2)/2, (node.Lon1+node.Lon2)/2, node.Lon2, node.Capacity)
	node.SW = NewTreeNode(node.Lat1, (node.Lat1+node.Lat2)/2, node.Lon1, (node.Lon1+node.Lon2)/2, node.Capacity)
	node.NE = NewTreeNode((node.Lat1+node.Lat2)/2, node.Lat2, (node.Lon1+node.Lon2)/2, node.Lon2, node.Capacity)
	node.NW = NewTreeNode((node.Lat1+node.Lat2)/2, node.Lat2, node.Lon1, (node.Lon1+node.Lon2)/2, node.Capacity)

	node.IsDivided = true
	for i, location := range node.Objects {
		if location == nil {
			panic("The Node is holding nil location. weird don't you think?. Location index: " + i)
		}
		delete(node.Objects, location.Id())
		location.Node = nil

		err := node.NW.insert(location)
		if err != nil {
			err = node.NE.insert(location)
			if err != nil {
				err = node.SW.insert(location)
				if err != nil {
					_ = node.SE.insert(location) // Ignoring the error here as it means the object has been moved mid-flight to another node which is not a problem
				}
			}
		}
	}

	node.Objects = map[string]*Location{}
}

func (*QuadTree) reBalance() {
	// TODO: Implement rebalancing
}

func rectangleOverlap(lat1 float64, lat2 float64, lon1 float64, lon2 float64, lat3 float64, lat4 float64, lon3 float64, lon4 float64) bool {
	return math.Max(lat1, lat3) < math.Min(lat2, lat4) && math.Max(lon1, lon3) < math.Min(lon2, lon4)
}

func (node *TreeNode) QueryRange(lat1 float64, lat2 float64, lon1 float64, lon2 float64) []*Location {

	var locations []*Location

	if !rectangleOverlap(node.Lat1, node.Lat2, node.Lon1, node.Lon2, lat1, lat2, lon1, lon2) {
		return locations
	}

	if !node.IsDivided {
		node.mu.RLock()
		for _, location := range node.Objects {
			if location.Lon() >= lon1 && location.Lon() <= lon2 && location.Lat() >= lat1 && location.Lat() <= lat2 {
				locations = append(locations, location)
			}
		}
		node.mu.RUnlock()

		return locations
	}

	if rectangleOverlap(node.NE.Lat1, node.NE.Lat2, node.NE.Lon1, node.NE.Lon2, lat1, lat2, lon1, lon2) {
		locations = append(locations, node.NE.QueryRange(lat1, lat2, lon1, lon2)...)
	}

	if rectangleOverlap(node.NW.Lat1, node.NW.Lat2, node.NW.Lon1, node.NW.Lon2, lat1, lat2, lon1, lon2) {
		locations = append(locations, node.NW.QueryRange(lat1, lat2, lon1, lon2)...)
	}

	if rectangleOverlap(node.SE.Lat1, node.SE.Lat2, node.SE.Lon1, node.SE.Lon2, lat1, lat2, lon1, lon2) {
		locations = append(locations, node.SE.QueryRange(lat1, lat2, lon1, lon2)...)
	}

	if rectangleOverlap(node.SW.Lat1, node.SW.Lat2, node.SW.Lon1, node.SW.Lon2, lat1, lat2, lon1, lon2) {
		locations = append(locations, node.SW.QueryRange(lat1, lat2, lon1, lon2)...)
	}

	return locations
}

func (node *TreeNode) ForceDivide(level int) {
	if level == 0 {
		return
	}
	if !node.IsDivided {
		node.divide()
	}

	level--

	node.NE.ForceDivide(level)
	node.NW.ForceDivide(level)
	node.SE.ForceDivide(level)
	node.SW.ForceDivide(level)
}
