package api

import (
	"github.com/MustardSeedNetworks/niac-go/internal/api/sse"
	"github.com/MustardSeedNetworks/niac-go/internal/capturering"
	"github.com/MustardSeedNetworks/niac-go/internal/protocols"
)

// attachPacketObservers wires a session's stack into the two consumers of its
// packet stream and returns the capture ring for the session's state.
//
// They are deliberately separate observers with different jobs. The SSE
// bridge is for the live view: it truncates each frame to 256 bytes for the
// browser's hex pane and retains nothing, because a UI that scrolled forever
// would pin memory the operator never asked for. The ring is for export: it
// keeps whole frames, bounded, and is the only thing a pcapng download can
// read — the stream by itself means an exchange that happened before anyone
// connected is gone.
//
// Both must stay cheap: the stack holds an RLock over its observer list on
// the receive path.
func (s *Server) attachPacketObservers(
	sessionID string, stack *protocols.Stack,
) *capturering.Ring {
	ring := capturering.New(capturering.DefaultLimits())
	if stack == nil {
		return ring
	}
	// Previously the hub had BroadcastPacket defined but it was never called,
	// leaving the Packet Capture page perpetually empty even while a
	// simulation was clearly handling traffic.
	if s.sseHub != nil {
		stack.AddPacketObserver(sse.NewPacketObserver(s.sseHub, sessionID))
	}
	stack.AddPacketObserver(ring)

	return ring
}
