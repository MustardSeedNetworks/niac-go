// Package capturering retains the most recent frames a simulation session
// handled so they can be exported as pcapng.
//
// The SSE bridge in internal/api/sse broadcasts every packet and keeps
// nothing, and it truncates each frame to 256 bytes for the browser's hex
// view. Neither shape can feed Wireshark: a client that connects after the
// interesting exchange has already missed it, and a truncated frame is not a
// capture. This package is the retained, full-frame side of the same seam —
// a second protocols.PacketObserver on the session's stack.
package capturering

import (
	"sync"
	"time"

	"github.com/MustardSeedNetworks/niac-go/internal/protocols"
)

// Limits bounds one session's ring. Both bounds apply: the frame count keeps
// a flood of small frames from costing an unbounded number of allocations,
// and the byte budget keeps a run of jumbo frames from costing unbounded
// memory. A daemon runs several sessions at once, so the budget is per
// session and deliberately modest.
type Limits struct {
	Frames int
	Bytes  int
}

// DefaultLimits retains roughly the last few seconds of a busy pack: enough
// that an operator who notices something and clicks Export still has it,
// small enough that six concurrent sessions cost under 50 MB.
func DefaultLimits() Limits { return Limits{Frames: defaultFrames, Bytes: defaultBytes} }

const (
	defaultFrames = 4096
	defaultBytes  = 8 << 20
)

// Frame is one retained frame together with the fabric annotations the stack
// made while handling it. Data is a private copy — the stack reuses its
// receive buffer.
type Frame struct {
	Timestamp time.Time
	Direction string // "rx" or "tx"
	Serial    int
	VLAN      int // -1 when the frame carried no 802.1Q tag
	Trace     protocols.FabricTrace
	Data      []byte
}

// Ring is a bounded FIFO of frames that implements protocols.PacketObserver.
//
// OnPacket runs on the stack's receive path under its observer RLock, so it
// must not block: it takes one mutex, copies the frame and returns.
type Ring struct {
	mu sync.Mutex
	// frames is a fixed circular buffer: head is the oldest entry, count is
	// how many are live. A slice that were re-sliced on eviction would copy
	// every retained header on every packet once the ring filled — which is
	// the steady state of a busy pack, on the stack's receive path.
	frames []Frame
	head   int
	count  int
	bytes  int
	limits Limits
}

// New returns a ring bounded by limits. A non-positive bound falls back to
// its DefaultLimits value, so a zero Limits is usable.
func New(limits Limits) *Ring {
	if limits.Frames <= 0 {
		limits.Frames = defaultFrames
	}
	if limits.Bytes <= 0 {
		limits.Bytes = defaultBytes
	}

	return &Ring{limits: limits, frames: make([]Frame, limits.Frames)}
}

// OnPacket implements protocols.PacketObserver.
func (r *Ring) OnPacket(direction string, pkt *protocols.Packet) {
	if r == nil || pkt == nil {
		return
	}
	raw := pkt.Buffer
	if pkt.Length > 0 && pkt.Length <= len(raw) {
		raw = raw[:pkt.Length]
	}
	if len(raw) == 0 {
		return
	}

	// Only NewPacket and ParsePacket stamp a Timestamp; the protocol emitters
	// build Packet literals directly and leave it zero, which would write an
	// epoch-zero frame into the capture.
	ts := pkt.Timestamp
	if ts.IsZero() {
		ts = time.Now()
	}

	frame := Frame{
		Timestamp: ts,
		Direction: direction,
		Serial:    pkt.SerialNumber,
		VLAN:      pkt.VLAN,
		Trace:     pkt.FabricTrace(),
		Data:      append([]byte(nil), raw...),
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	if r.count == len(r.frames) {
		r.dropOldestLocked()
	}
	r.frames[(r.head+r.count)%len(r.frames)] = frame
	r.count++
	r.bytes += len(frame.Data)
	for r.bytes > r.limits.Bytes && r.count > 1 {
		r.dropOldestLocked()
	}
}

// dropOldestLocked advances past the oldest frame, clearing the slot so the
// dropped frame's Data is not pinned for as long as the ring lives.
func (r *Ring) dropOldestLocked() {
	r.bytes -= len(r.frames[r.head].Data)
	r.frames[r.head] = Frame{}
	r.head = (r.head + 1) % len(r.frames)
	r.count--
}

// Snapshot returns the retained frames oldest-first. last <= 0 returns
// everything; otherwise the newest last frames. The Frame values share their
// Data with the ring, which never mutates a stored frame.
func (r *Ring) Snapshot(last int) []Frame {
	r.mu.Lock()
	defer r.mu.Unlock()
	take := r.count
	if last > 0 && last < take {
		take = last
	}
	out := make([]Frame, take)
	for i := range take {
		out[i] = r.frames[(r.head+r.count-take+i)%len(r.frames)]
	}

	return out
}
