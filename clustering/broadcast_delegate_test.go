package clustering

import (
	"testing"

	"github.com/fabricekabongo/loggerhead/query"
	"github.com/fabricekabongo/loggerhead/world"
	"github.com/hashicorp/memberlist"
)

func TestBroadcastDelegateNodeMeta(t *testing.T) {
	engine := query.NewWriteQueryEngine(world.NewWorld())
	delegate := newBroadcastDelegate(engine, &memberlist.TransmitLimitedQueue{})

	if data := delegate.NodeMeta(0); len(data) != 0 {
		t.Fatalf("expected empty node meta, got %v", data)
	}
}

func TestBroadcastDelegateNotifyMsgExecutesCommand(t *testing.T) {
	w := world.NewWorld()
	engine := query.NewWriteQueryEngine(w)
	delegate := newBroadcastDelegate(engine, &memberlist.TransmitLimitedQueue{})

	delegate.NotifyMsg([]byte("SAVE ns loc 1 2"))

	location, ok := w.GetLocation("ns", "loc")
	if !ok {
		t.Fatalf("expected location to be saved")
	}

	if location.Lat() != 1 || location.Lon() != 2 {
		t.Fatalf("unexpected location coordinates: %v", location)
	}
}

func TestBroadcastDelegateGetBroadcasts(t *testing.T) {
	broadcasts := &memberlist.TransmitLimitedQueue{NumNodes: func() int { return 1 }, RetransmitMult: 1}
	broadcasts.QueueBroadcast(NewLocationBroadcast("SAVE ns loc 1 1"))

	delegate := newBroadcastDelegate(query.NewWriteQueryEngine(world.NewWorld()), broadcasts)

	got := delegate.GetBroadcasts(0, 1024)
	if len(got) != 1 {
		t.Fatalf("expected 1 broadcast, got %d", len(got))
	}

	if string(got[0]) != "SAVE ns loc 1 1" {
		t.Fatalf("unexpected broadcast payload %q", string(got[0]))
	}
}

func TestBroadcastDelegateLocalState(t *testing.T) {
	w := world.NewWorld()
	_ = w.Save("ns", "loc", 10, 20)

	delegate := newBroadcastDelegate(query.NewWriteQueryEngine(w), &memberlist.TransmitLimitedQueue{})

	if data := delegate.LocalState(false); len(data) != 0 {
		t.Fatalf("expected empty state when join is false, got %v", data)
	}

	data := delegate.LocalState(true)
	restored := world.NewWorldFromBytes(data)

	loc, ok := restored.GetLocation("ns", "loc")
	if !ok || loc.Lat() != 10 || loc.Lon() != 20 {
		t.Fatalf("expected saved location after restoration, got %#v, present=%v", loc, ok)
	}
}

func TestBroadcastDelegateMergeRemoteState(t *testing.T) {
	remote := world.NewWorld()
	_ = remote.Save("ns", "loc", 3, 4)
	buf := remote.ToBytes()

	local := world.NewWorld()
	delegate := newBroadcastDelegate(query.NewWriteQueryEngine(local), &memberlist.TransmitLimitedQueue{})

	delegate.MergeRemoteState(buf, true)

	loc, ok := local.GetLocation("ns", "loc")
	if !ok || loc.Lat() != 3 || loc.Lon() != 4 {
		t.Fatalf("expected merged location, got %#v, present=%v", loc, ok)
	}
}
