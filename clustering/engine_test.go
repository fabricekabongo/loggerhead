package clustering

import (
	"context"
	"testing"
	"time"

	"github.com/fabricekabongo/loggerhead/query"
	"github.com/fabricekabongo/loggerhead/world"
	"github.com/hashicorp/memberlist"
)

func TestEngineDecoratorQueuesBroadcasts(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cluster := &Cluster{broadcasts: &memberlist.TransmitLimitedQueue{NumNodes: func() int { return 1 }, RetransmitMult: 1}}
	engine := query.NewWriteQueryEngine(world.NewWorld())

	decorator := NewEngineDecorator(ctx, cluster, engine)
	ed := decorator.(*EngineDecorator)

	result := ed.ExecuteQuery("SAVE ns loc 1 1")
	if result != "1.0,saved\n" {
		t.Fatalf("expected save confirmation, got %q", result)
	}

	var broadcasts [][]byte
	deadline := time.Now().Add(200 * time.Millisecond)
	for time.Now().Before(deadline) {
		broadcasts = cluster.broadcasts.GetBroadcasts(0, 1024)
		if len(broadcasts) > 0 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	if len(broadcasts) != 1 {
		t.Fatalf("expected 1 broadcast, got %d", len(broadcasts))
	}

	if string(broadcasts[0]) != "SAVE ns loc 1 1" {
		t.Fatalf("unexpected broadcast message %q", string(broadcasts[0]))
	}
}

func TestEngineDecoratorStopsWithCanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cluster := &Cluster{broadcasts: &memberlist.TransmitLimitedQueue{NumNodes: func() int { return 1 }, RetransmitMult: 1}}
	engine := query.NewWriteQueryEngine(world.NewWorld())

	NewEngineDecorator(ctx, cluster, engine)
	cancel()

	time.Sleep(10 * time.Millisecond)

	select {
	case <-ctx.Done():
	default:
		t.Fatal("context should be cancelled")
	}
}

func TestLocationBroadcastBehaviors(t *testing.T) {
	first := NewLocationBroadcast("SAVE ns loc 1 1")
	second := NewLocationBroadcast("SAVE ns loc 1 1")
	different := NewLocationBroadcast("SAVE ns loc 2 2")

	if !first.Invalidates(second) {
		t.Fatal("expected identical commands to invalidate each other")
	}

	if different.Invalidates(first) {
		t.Fatal("expected different commands to not invalidate")
	}

	notify := make(chan struct{})
	withNotify := &LocationBroadcast{command: "SAVE ns loc 1 1", msg: []byte("payload"), notify: notify}
	withNotify.Finished()

	select {
	case <-notify:
	case <-time.After(time.Second):
		t.Fatal("expected notify channel to be closed")
	}

	if string(first.Message()) != "SAVE ns loc 1 1" {
		t.Fatalf("unexpected message payload %q", string(first.Message()))
	}
}
