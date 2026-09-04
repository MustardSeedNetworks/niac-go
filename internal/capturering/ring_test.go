package capturering_test

import (
	"runtime"
	"testing"
	"time"

	"github.com/MustardSeedNetworks/niac-go/internal/capturering"
	"github.com/MustardSeedNetworks/niac-go/internal/protocols"
)

func frame(n byte, size int) *protocols.Packet {
	buf := make([]byte, size)
	buf[0] = n
	return &protocols.Packet{Buffer: buf, Length: size, VLAN: -1, Timestamp: time.Unix(0, int64(n))}
}

func TestRingKeepsTheMostRecentFrames(t *testing.T) {
	r := capturering.New(capturering.Limits{Frames: 3, Bytes: 1 << 20})
	for i := range byte(5) {
		r.OnPacket("rx", frame(i, 64))
	}

	got := r.Snapshot(0)
	if len(got) != 3 {
		t.Fatalf("frames = %d, want 3", len(got))
	}
	for i, f := range got {
		if want := byte(i + 2); f.Data[0] != want {
			t.Errorf("frame %d starts with %d, want %d", i, f.Data[0], want)
		}
	}
}

func TestRingCopiesTheFrameSoAReusedBufferCannotCorruptIt(t *testing.T) {
	r := capturering.New(capturering.Limits{Frames: 4, Bytes: 1 << 20})
	pkt := frame(7, 64)
	r.OnPacket("tx", pkt)
	pkt.Buffer[0] = 0xff

	got := r.Snapshot(0)
	if len(got) != 1 {
		t.Fatalf("frames = %d, want 1", len(got))
	}
	if got[0].Data[0] != 7 {
		t.Errorf("stored frame aliased the stack's buffer: first byte = %#x", got[0].Data[0])
	}
}

func TestRingEvictsOldestOnceTheByteBudgetIsSpent(t *testing.T) {
	r := capturering.New(capturering.Limits{Frames: 100, Bytes: 3 * 1000})
	for i := range byte(5) {
		r.OnPacket("rx", frame(i, 1000))
	}

	got := r.Snapshot(0)
	if len(got) != 3 {
		t.Fatalf("frames = %d, want 3 (byte budget holds three 1000-byte frames)", len(got))
	}
	if got[0].Data[0] != 2 {
		t.Errorf("oldest retained frame = %d, want 2", got[0].Data[0])
	}
}

func TestSnapshotLastReturnsTheNewestNFramesOldestFirst(t *testing.T) {
	r := capturering.New(capturering.Limits{Frames: 10, Bytes: 1 << 20})
	for i := range byte(6) {
		r.OnPacket("rx", frame(i, 64))
	}

	got := r.Snapshot(2)
	if len(got) != 2 {
		t.Fatalf("frames = %d, want 2", len(got))
	}
	if got[0].Data[0] != 4 || got[1].Data[0] != 5 {
		t.Errorf("Snapshot(2) = %d,%d; want 4,5", got[0].Data[0], got[1].Data[0])
	}
}

func TestOnPacketIgnoresAnEmptyFrame(t *testing.T) {
	r := capturering.New(capturering.Limits{Frames: 4, Bytes: 1 << 20})
	r.OnPacket("rx", &protocols.Packet{Buffer: nil, Length: 0, VLAN: -1})
	if n := len(r.Snapshot(0)); n != 0 {
		t.Errorf("frames = %d, want 0", n)
	}
}

func TestOnPacketStampsObservationTimeWhenTheEmitterLeftItZero(t *testing.T) {
	r := capturering.New(capturering.Limits{Frames: 4, Bytes: 1 << 20})
	pkt := frame(1, 64)
	pkt.Timestamp = time.Time{}
	before := time.Now()
	r.OnPacket("tx", pkt)

	got := r.Snapshot(0)
	if got[0].Timestamp.Before(before) {
		t.Errorf("timestamp %v predates the observation", got[0].Timestamp)
	}
}

func TestRingCostPerPacketDoesNotScaleWithItsCapacity(t *testing.T) {
	// The receive path holds the stack's observer lock, so OnPacket must cost
	// the same on a 4096-frame ring as on a 64-frame one. An implementation
	// that rebuilt its backing array on eviction allocated and copied every
	// retained frame header on every packet once full — the same allocation
	// *count*, but ~700 KB of it, which an AllocsPerRun check would miss.
	const frameSize = 64
	measure := func(capacity int) uint64 {
		ring := capturering.New(capturering.Limits{Frames: capacity, Bytes: 1 << 30})
		for i := range capacity {
			ring.OnPacket("rx", frame(byte(i), frameSize))
		}
		pkt := frame(1, frameSize)
		runtime.GC()
		var before, after runtime.MemStats
		runtime.ReadMemStats(&before)
		for range 2000 {
			ring.OnPacket("rx", pkt)
		}
		runtime.ReadMemStats(&after)

		return (after.TotalAlloc - before.TotalAlloc) / 2000
	}

	small, large := measure(64), measure(4096)
	// Both should be about one frame copy. Allow generous slack for the
	// allocator's size classes; the defect this pins was three orders of
	// magnitude, not a few bytes.
	if large > 4*frameSize {
		t.Errorf("a 4096-frame ring costs %d bytes per packet, want ~%d", large, frameSize)
	}
	if large > 4*small {
		t.Errorf("per-packet cost scales with capacity: %d bytes at 4096 vs %d at 64", large, small)
	}
}

func TestRingKeepsTheNewestFramesAfterManyWraps(t *testing.T) {
	r := capturering.New(capturering.Limits{Frames: 4, Bytes: 1 << 20})
	for i := range byte(50) {
		r.OnPacket("rx", frame(i, 64))
	}

	got := r.Snapshot(0)
	if len(got) != 4 {
		t.Fatalf("frames = %d, want 4", len(got))
	}
	for i, f := range got {
		if want := byte(46 + i); f.Data[0] != want {
			t.Errorf("frame %d starts with %d, want %d", i, f.Data[0], want)
		}
	}
}

func TestEvictedSlotsDoNotPinTheirFrameBytes(t *testing.T) {
	// A dropped frame's Data must be released, not left reachable from the
	// backing array for as long as the ring lives.
	r := capturering.New(capturering.Limits{Frames: 2, Bytes: 1 << 20})
	for i := range byte(6) {
		r.OnPacket("rx", frame(i, 1024))
	}

	var held int
	for _, f := range r.Snapshot(0) {
		held += len(f.Data)
	}
	if held != 2*1024 {
		t.Errorf("retained bytes = %d, want %d", held, 2*1024)
	}
}
