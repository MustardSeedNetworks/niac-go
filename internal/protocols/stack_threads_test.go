package protocols

import (
	"bytes"
	"testing"
)

// TestParseReceivedPacketOwnsBuffer verifies a received packet does not alias
// the reusable capture buffer. ReadPacket hands back a slice of a buffer that
// the receive loop overwrites on the next read; the packet is decoded on
// another goroutine, so it must own its bytes or in-flight packets get
// corrupted (a data race across every handler).
func TestParseReceivedPacketOwnsBuffer(t *testing.T) {
	s := &Stack{stats: &Statistics{}}

	// A recognizable "first packet" living in a reusable buffer.
	shared := bytes.Repeat([]byte{0xAB}, 64)
	snapshot := bytes.Clone(shared)

	pkt := s.parseReceivedPacket(shared)
	if pkt == nil {
		t.Fatal("parseReceivedPacket returned nil")
	}

	// Simulate the receive loop reading the next packet into the same buffer.
	for i := range shared {
		shared[i] = 0xCD
	}

	if !bytes.Equal(pkt.Buffer, snapshot) {
		t.Errorf("packet buffer was corrupted by a subsequent read: got %x…, want %x…",
			pkt.Buffer[:4], snapshot[:4])
	}
}
