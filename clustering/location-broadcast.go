package clustering

import (
	"github.com/hashicorp/memberlist"
)

type LocationBroadcast struct {
	command string
	msg     []byte
	notify  chan<- struct{}
}

func NewLocationBroadcast(command string) *LocationBroadcast {
	return &LocationBroadcast{
		command: command,
		msg:     []byte(command),
	}
}

func (b *LocationBroadcast) Invalidates(old memberlist.Broadcast) bool {
	return b.command == old.(*LocationBroadcast).command // Prevents duplicate messages but breaks the message ordering
}

func (b *LocationBroadcast) Message() []byte {
	return b.msg
}

func (b *LocationBroadcast) Finished() {
	if b.notify != nil {
		close(b.notify) // Notify when message sending is finished
	}
}
