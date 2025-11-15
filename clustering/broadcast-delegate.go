package clustering

import (
	"log"
	"os"

	"github.com/fabricekabongo/loggerhead/query"
	"github.com/fabricekabongo/loggerhead/world"
	"github.com/hashicorp/memberlist"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	LocalStateSharedCounter prometheus.Counter
	MergeRemoteStateCounter prometheus.Counter
)

func init() {
	name, err := os.Hostname()
	if err != nil {
		log.Println("Failed to get hostname")
		name = "unknown"
	}
	LocalStateSharedCounter = promauto.NewCounter(prometheus.CounterOpts{
		Name:        "loggerhead_clustering_local_state_shared",
		Help:        "Local state shared with new node",
		ConstLabels: map[string]string{"hostname": name},
	})

	MergeRemoteStateCounter = promauto.NewCounter(prometheus.CounterOpts{
		Name:        "loggerhead_clustering_remote_state_merged",
		Help:        "Remote state merged with local state",
		ConstLabels: map[string]string{"hostname": name},
	})
}

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

func (*BroadcastDelegate) NodeMeta(limit int) []byte {
	return []byte{}
}

func (d *BroadcastDelegate) NotifyMsg(buf []byte) {
	if len(buf) > 0 {
		command := string(buf)

		_ = d.state.engine.ExecuteQuery(command)
		log.Println("Received cluster command: ", command)
	}

}

func (d *BroadcastDelegate) GetBroadcasts(overhead, limit int) [][]byte {
	return d.broadcasts.GetBroadcasts(overhead, limit)
}

func (d *BroadcastDelegate) LocalState(join bool) []byte {
	defer LocalStateSharedCounter.Inc()
	if join {
		log.Println("Sharing local state to a new node")
		return d.state.engine.World().ToBytes()
	}

	return []byte{}
}

func (d *BroadcastDelegate) MergeRemoteState(buf []byte, join bool) {
	defer MergeRemoteStateCounter.Inc()
	if join {
		log.Println("Bootstrapping new node with remote state")
		w := world.NewWorldFromBytes(buf)

		d.state.engine.World().Merge(w)
	}
}
