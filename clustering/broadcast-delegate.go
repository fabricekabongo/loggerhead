package clustering

import (
	"bytes"
	"encoding/gob"
	"github.com/fabricekabongo/loggerhead/world"
	"github.com/hashicorp/memberlist"
	"log"
	"runtime"
)

func init() {
	gob.Register(NodeMetaData{})
	gob.Register(world.Stats{})
}

type BroadcastDelegate struct {
	state      *NodeState
	broadcasts *memberlist.TransmitLimitedQueue
}

type NodeState struct {
	World *world.Map
}

type NodeMetaData struct {
	Locations  int
	Grids      int
	MemStats   MemStats
	CPUs       int
	GoRoutines int
}

type MemStats struct {
	Alloc      uint64
	TotalAlloc uint64
	Sys        uint64
}

func NewBroadcastDelegate(world *world.Map, broadcasts *memberlist.TransmitLimitedQueue) *BroadcastDelegate {
	return &BroadcastDelegate{
		state: &NodeState{
			World: world,
		},
		broadcasts: broadcasts,
	}
}

func (d *BroadcastDelegate) NodeMeta(limit int) []byte {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	stats := d.state.World.Stats()
	metaData := NodeMetaData{
		Locations: stats.Locations,
		Grids:     stats.Grids,
		MemStats: MemStats{
			Alloc:      (memStats.Alloc / 1024) / 1024,
			TotalAlloc: (memStats.TotalAlloc / 1024) / 1024,
			Sys:        (memStats.Sys / 1024) / 1024,
		},
		CPUs:       runtime.NumCPU(),
		GoRoutines: runtime.NumGoroutine(),
	}

	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(metaData)
	if err != nil {
		return []byte{}
	}

	return buf.Bytes()
}

func (d *BroadcastDelegate) NotifyMsg(buf []byte) {
	if len(buf) > 0 {
		dec := gob.NewDecoder(bytes.NewReader(buf))
		// Process the message
		var location world.Location

		err := dec.Decode(&location)
		if err != nil {
			return
		}

		err = d.state.World.Save(location.Id, location.Lat, location.Lon)
		if err != nil {
			return
		}
	}
}

func (d *BroadcastDelegate) GetBroadcasts(overhead, limit int) [][]byte {
	return d.broadcasts.GetBroadcasts(overhead, limit)
}

func (d *BroadcastDelegate) LocalState(join bool) []byte {
	d.state.World.Mu.RLock()
	defer d.state.World.Mu.RUnlock()

	if join {
		log.Println("Sharing local state to a new node")
	} else {
		log.Println("Sharing local state for routine sync")
	}

	var buf bytes.Buffer

	enc := gob.NewEncoder(&buf)
	err := enc.Encode(d.state.World)
	if err != nil {
		return []byte{}
	}

	return buf.Bytes()
}

func (d *BroadcastDelegate) MergeRemoteState(buf []byte, join bool) {
	if join {
		log.Println("Getting state from the cluster to start well")
	} else {
		log.Println("Getting state from the cluster for routine sync")
	}

	dec := gob.NewDecoder(bytes.NewReader(buf))
	var worldMap world.Map

	err := dec.Decode(&worldMap)
	if err != nil {
		return
	}

	go func(worldMap world.Map) {
		for _, loc := range worldMap.Locations {
			err := d.state.World.Save(loc.Id, loc.Lat, loc.Lon)
			if err != nil {
				continue
			}
		}
	}(worldMap)
}
