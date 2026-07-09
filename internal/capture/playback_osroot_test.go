package capture

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
)

// TestPlaybackEngine_LoadPCAP_RoundTrip verifies the pcapgo-based read path
// loads every packet with its timestamp — the behaviour the previous
// libpcap pcap.OpenOffline path provided.
func TestPlaybackEngine_LoadPCAP_RoundTrip(t *testing.T) {
	path := createTestPCAPFile(t, 3)

	pb := &PlaybackEngine{
		config:   &config.CapturePlayback{FileName: path},
		stopChan: make(chan struct{}),
	}

	packets, err := pb.loadPCAP()
	if err != nil {
		t.Fatalf("loadPCAP: %v", err)
	}
	if len(packets) != 3 {
		t.Fatalf("got %d packets, want 3", len(packets))
	}
	for i := range packets {
		if len(packets[i].Data) == 0 {
			t.Errorf("packet %d has empty data", i)
		}
		if i > 0 && !packets[i].Timestamp.After(packets[i-1].Timestamp) {
			t.Errorf("packet %d timestamp %v not after previous %v",
				i, packets[i].Timestamp, packets[i-1].Timestamp)
		}
	}
}

// TestPlaybackEngine_LoadPCAP_RejectsSymlinkEscape proves the os.Root
// containment: a playback file that is a symlink escaping its own directory
// is rejected at open. The previous pcap.OpenOffline path followed the
// symlink and would have replayed a file outside the validated directory —
// a validate→open TOCTOU escape.
func TestPlaybackEngine_LoadPCAP_RejectsSymlinkEscape(t *testing.T) {
	// A real, loadable pcap living outside the playback directory.
	outside := createTestPCAPFile(t, 1)

	// A playback directory whose entry is a symlink to that outside file.
	dir := t.TempDir()
	link := filepath.Join(dir, "evil.pcap")
	if err := os.Symlink(outside, link); err != nil {
		t.Skipf("symlinks unsupported on this platform: %v", err)
	}

	pb := &PlaybackEngine{
		config:   &config.CapturePlayback{FileName: link},
		stopChan: make(chan struct{}),
	}

	if _, err := pb.loadPCAP(); err == nil {
		t.Fatal("expected loadPCAP to reject a symlink escaping its directory, got nil")
	}
}
