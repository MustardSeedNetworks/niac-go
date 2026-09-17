package capture

import (
	"errors"
	"testing"

	"github.com/gopacket/gopacket/pcap"
)

// TestEngineStatsNegativeCountersClampToZero covers the one way libpcap can
// hand back a value uint64 cannot hold: pcap.Stats is signed, and a platform
// that does not implement a counter reports -1 rather than 0.
func TestEngineStatsNegativeCountersClampToZero(t *testing.T) {
	engine, handle := newFakeEngine(0)
	handle.stats = pcap.Stats{PacketsReceived: 5, PacketsDropped: -1, PacketsIfDropped: -1}

	stats, err := engine.Stats()
	if err != nil {
		t.Fatalf("Stats() error = %v", err)
	}

	if stats.PacketsDropped != 0 || stats.PacketsIfDropped != 0 {
		t.Errorf("negative counters = (%d, %d), want (0, 0)",
			stats.PacketsDropped, stats.PacketsIfDropped)
	}
}

// TestEngineStatsAfterCloseIsRefused: gopacket hands the raw handle pointer
// to C without a nil check and Close nils it, so a stats read racing
// simulation teardown would call libpcap with NULL. Simulation stop closes
// the engine while the polled read surfaces are still live, so this is a
// reachable path, not a hypothetical one.
func TestEngineStatsAfterCloseIsRefused(t *testing.T) {
	engine, handle := newFakeEngine(0)
	handle.stats = pcap.Stats{PacketsReceived: 10, PacketsDropped: 2}

	engine.Close()

	if _, err := engine.Stats(); !errors.Is(err, ErrEngineClosed) {
		t.Fatalf("Stats() after Close error = %v, want %v", err, ErrEngineClosed)
	}
}
