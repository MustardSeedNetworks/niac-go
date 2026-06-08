// SPDX-License-Identifier: BUSL-1.1

package api

import (
	"github.com/gopacket/gopacket"

	"github.com/MustardSeedNetworks/niac-go/internal/api/sse"
)

// BroadcastCapturePacket ships a standalone-captured gopacket onto the SSE
// packets stream. Used by the daemon's standalone capture loop, which has no
// protocols.Stack and therefore no PacketObserver. The wire shape matches the
// observer's so the UI stream reader needs no branch for the two sources.
//
// Direction is always "rx" — standalone capture is read-only.
func (s *Server) BroadcastCapturePacket(pkt gopacket.Packet) {
	if s == nil || s.sseHub == nil || pkt == nil {
		return
	}
	s.sseHub.BroadcastPacket(sse.GopacketToWire(pkt))
}
