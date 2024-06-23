package world

import (
	"errors"
	"strconv"
	"sync"
	"sync/atomic"
)

var (
	TreeErrLocationNil         = errors.New("insertion failed because location is nil")
	TreeErrLocationOutOfBounds = errors.New("insertion failed because location is out of bounds")
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
		Objects:   []*Location{},
	}
}

func (n *TreeNode) insert(location *Location) error {
	if location == nil {
		return TreeErrLocationNil
	}
	// If the location is not within the bound, return
	if !(n.Lon1 < location.Lon() && n.Lon2 > location.Lon() && n.Lat1 < location.Lat() && n.Lat2 > location.Lat()) {
		return TreeErrLocationOutOfBounds
	}

	if n.IsDivided {
		n.insertIntoChildren(location)
	}
	n.mu.Lock()
	n.Objects = append(n.Objects, location)
	n.mu.Unlock()

	if len(n.Objects) > n.Capacity {
		n.divide()
	}

	return nil
}

func (n *TreeNode) insertIntoChildren(location *Location) {
	if location == nil {
		panic("Location is nil. It should never reach this point")
	}
	wg := sync.WaitGroup{}
	wg.Add(4)
	passedCount := atomic.Int32{}

	go func() {
		defer wg.Done()
		err := n.NE.insert(location)
		if err == nil {
			passedCount.Add(1)
		}
	}()

	go func() {
		defer wg.Done()
		err := n.NW.insert(location)
		if err == nil {
			passedCount.Add(1)
		}
	}()

	go func() {
		defer wg.Done()
		err := n.SE.insert(location)
		if err == nil {
			passedCount.Add(1)
		}
	}()

	go func() {
		defer wg.Done()
		err := n.SW.insert(location)
		if err == nil {
			passedCount.Add(1)
		}
	}()

	wg.Wait()
	if passedCount.Load() != 1 {
		panic("Location should have been inserted into one of the nodes. Number of nodes inserted: " + string(passedCount.Load()))
	}
}

func (n *TreeNode) Delete(id string) {
	if n.IsDivided {
		n.NE.Delete(id)
		n.NW.Delete(id)
		n.SE.Delete(id)
		n.SW.Delete(id)
	}

	n.mu.Lock()
	for i, location := range n.Objects {
		if location.Id() == id {
			if i >= 0 && i < len(n.Objects) {
				n.Objects = append(n.Objects[:i], n.Objects[i+1:]...)
			}
		}
	}
	n.mu.Unlock()
}

func (n *TreeNode) divide() {
	n.NE = NewTreeNode(n.Lat1, (n.Lat1+n.Lat2)/2, (n.Lon1+n.Lon2)/2, n.Lon2, n.Capacity)

	n.NW = NewTreeNode(n.Lat1, (n.Lat1+n.Lat2)/2, n.Lon1, (n.Lon1+n.Lon2)/2, n.Capacity)

	n.SE = NewTreeNode((n.Lat1+n.Lat2)/2, n.Lat2, (n.Lon1+n.Lon2)/2, n.Lon2, n.Capacity)

	n.SW = NewTreeNode((n.Lat1+n.Lat2)/2, n.Lat2, n.Lon1, (n.Lon1+n.Lon2)/2, n.Capacity)

	n.mu.Lock()
	for i, location := range n.Objects {
		if location == nil {
			panic("The Node is holding nil location. weird don't you think?. Location index: " + strconv.Itoa(i))
		}
		n.insertIntoChildren(location)
	}

	n.Objects = []*Location{}
	n.IsDivided = true
	n.mu.Unlock()
}

func (q *QuadTree) reBalance() {
	// TODO: Implement rebalancing
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
		if location.Lon() >= lon1 && location.Lon() <= lon2 && location.Lat() >= lat1 && location.Lat() <= lat2 {
			locations = append(locations, location)
		}
	}

	return locations
}
