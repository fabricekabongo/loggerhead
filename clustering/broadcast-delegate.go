package clustering

import (
	"github.com/fabricekabongo/loggerhead/query"
	"github.com/fabricekabongo/loggerhead/world"
	"github.com/hashicorp/memberlist"
	"log"
)

type BroadcastDelegate struct {
	state      *NodeState
	broadcasts *memberlist.TransmitLimitedQueue
}

type NodeState struct {
	engine *query.Engine
}

func newBroadcastDelegate(engine *query.Engine, broadcasts *memberlist.TransmitLimitedQueue) *BroadcastDelegate {
	return &BroadcastDelegate{
		state: &NodeState{
			engine: engine,
		},
		broadcasts: broadcasts,
	}
}

func (d *BroadcastDelegate) NodeMeta(limit int) []byte {
	return []byte{}
}

func (d *BroadcastDelegate) NotifyMsg(buf []byte) {
	go func(buf []byte) {
		if len(buf) > 0 {
			command := string(buf)

			_ = d.state.engine.ExecuteQuery(command) // Adding the ignored return so if I change the return definition (and add an error for example) the build will fail and I can fix this.
		}
	}(buf) // Execute the command in a goroutine to prevent blocking the memberlist
}

func (d *BroadcastDelegate) GetBroadcasts(overhead, limit int) [][]byte {
	return d.broadcasts.GetBroadcasts(overhead, limit)
}

func (d *BroadcastDelegate) LocalState(join bool) []byte {
	if join {
		log.Println("Sharing local state to a new node")
		return d.state.engine.World().ToBytes()
	}

	return []byte{}
}

func (d *BroadcastDelegate) MergeRemoteState(buf []byte, join bool) {
	if join {
		log.Println("Bootstrapping new node with remote state")
		w := world.NewWorldFromBytes(buf)

		d.state.engine.World().Merge(w)
	}
}
