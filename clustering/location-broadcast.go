package clustering

import (
	"bytes"
	"encoding/gob"
	"github.com/fabricekabongo/loggerhead/world"
	"github.com/hashicorp/memberlist"
)

type LocationBroadcast struct {
	locId  string
	msg    []byte
	notify chan<- struct{}
}

func NewLocationBroadcast(location world.LocationEntity) *LocationBroadcast {
	var msg bytes.Buffer

	enc := gob.NewEncoder(&msg)
	err := enc.Encode(location)

	if err != nil {
		return nil
	}

	return &LocationBroadcast{
		locId:  location.LocId,
		msg:    msg.Bytes(),
		notify: make(chan struct{}),
	}
}

func (b *LocationBroadcast) Invalidates(old memberlist.Broadcast) bool {
	if old == nil {
		return true
	}

	if old.(*LocationBroadcast).locId == b.locId {
		return true
	}

	return false
}

func (b *LocationBroadcast) Message() []byte {
	return b.msg
}

func (b *LocationBroadcast) Finished() {
	if b.notify != nil {
		close(b.notify) // Notify when message sending is finished
	}
}
