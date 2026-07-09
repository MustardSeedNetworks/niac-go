package capture

import (
	"os"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
)

// collectPlayback drains the streaming reader into a slice for assertions that
// predate streaming (they asserted against the old loadPCAP slice). Production
// replay never materializes the file; this is a test-only convenience over
// forEachPacket.
func collectPlayback(pb *PlaybackEngine) ([]PlaybackPacket, error) {
	var pkts []PlaybackPacket
	err := pb.forEachPacket(func(p PlaybackPacket) bool {
		pkts = append(pkts, p)
		return false
	})

	return pkts, err
}

// TestPlaybackEngine_ForEachPacket_StreamsLargeFile drives a 50k-packet capture
// through the streaming reader and asserts every packet is delivered one at a
// time — the memory-bound win over the old whole-file slice load.
func TestPlaybackEngine_ForEachPacket_StreamsLargeFile(t *testing.T) {
	const want = 50000
	path := createTestPCAPFile(t, want)

	pb := &PlaybackEngine{
		config:   &config.CapturePlayback{FileName: path},
		stopChan: make(chan struct{}),
	}

	var (
		read  int
		bytes uint64
	)
	err := pb.forEachPacket(func(p PlaybackPacket) bool {
		read++
		bytes += uint64(len(p.Data))

		return false
	})
	if err != nil {
		t.Fatalf("forEachPacket: %v", err)
	}
	if read != want {
		t.Fatalf("streamed %d packets, want %d", read, want)
	}
	if bytes == 0 {
		t.Fatal("streamed zero bytes")
	}
}

// TestPlaybackEngine_ForEachPacket_EarlyStop verifies the callback can halt the
// stream before EOF — the path the stop signal takes mid-replay.
func TestPlaybackEngine_ForEachPacket_EarlyStop(t *testing.T) {
	path := createTestPCAPFile(t, 10)

	pb := &PlaybackEngine{
		config:   &config.CapturePlayback{FileName: path},
		stopChan: make(chan struct{}),
	}

	var read int
	err := pb.forEachPacket(func(PlaybackPacket) bool {
		read++

		return read == 3 // stop after the third packet
	})
	if err != nil {
		t.Fatalf("forEachPacket: %v", err)
	}
	if read != 3 {
		t.Fatalf("stopped after %d packets, want 3", read)
	}
}

// TestPlaybackEngine_ForEachPacket_TruncatedTrailer confirms R4: a capture whose
// final packet record is truncated replays the valid prefix and stops cleanly
// (log-and-break) rather than erroring or panicking.
func TestPlaybackEngine_ForEachPacket_TruncatedTrailer(t *testing.T) {
	const valid = 5
	path := createTestPCAPFile(t, valid)

	// Append a partial per-packet record header so the reader hits an
	// unexpected EOF after the valid packets.
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = f.Write([]byte{0x01, 0x02, 0x03, 0x04, 0x05}); err != nil {
		t.Fatal(err)
	}
	_ = f.Close()

	pb := &PlaybackEngine{
		config:   &config.CapturePlayback{FileName: path},
		stopChan: make(chan struct{}),
	}

	packets, err := collectPlayback(pb)
	if err != nil {
		t.Fatalf("collectPlayback on truncated file: %v", err)
	}
	if len(packets) != valid {
		t.Fatalf("replayed %d packets from truncated capture, want %d (valid prefix)", len(packets), valid)
	}
}
