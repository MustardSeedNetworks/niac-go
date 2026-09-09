package protocols

import (
	"time"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
)

// sendEchoPacket sends one built echo reply, held back by the device's armed
// latency fault. The reply is serialized before the delay starts, so nothing
// that outlives the handler points into the capture buffer the request was
// read from -- that buffer is reused on the next read.
func (h *ICMPHandler) sendEchoPacket(device *config.Device, pkt *Packet) {
	if delay := h.echoReplyDelay(device); delay > 0 {
		time.AfterFunc(delay, func() { h.stack.Send(pkt) })

		return
	}

	h.stack.Send(pkt)
}

// echoReplyDelay returns how long an armed latency fault holds this device's
// ICMP echo reply. Latency is the one device fault with no service to
// suppress: the device still answers, just late, which is what a tester
// measures as round-trip time.
func (h *ICMPHandler) echoReplyDelay(device *config.Device) time.Duration {
	milliseconds := h.stack.deviceFaultValue(device, devicestate.FaultLatency)

	return time.Duration(milliseconds) * time.Millisecond
}
