package capture

import (
	"testing"
	"time"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
)

// TestPacingDelay_ModeMath checks the per-mode pacing math. rateDelay only
// computes a duration (it never sleeps), so large target elapsed values are
// cheap and keep the assertions off the wall clock: with startTime == now the
// returned delay is essentially the whole target interval.
func TestPacingDelay_ModeMath(t *testing.T) {
	start := time.Now()
	pkt := PlaybackPacket{Data: make([]byte, 1250), Timestamp: start} // 1250 B = 10000 bits

	tests := []struct {
		name        string
		cfg         config.CapturePlayback
		sentPackets uint64
		sentBytes   uint64
		wantZero    bool          // expect exactly 0
		lo, hi      time.Duration // else expect (lo, hi]
	}{
		{
			name:        "topspeed",
			cfg:         config.CapturePlayback{RateMode: config.RateTopspeed},
			sentPackets: 100,
			sentBytes:   1 << 20,
			wantZero:    true,
		},
		{
			name:        "pps first packet is immediate",
			cfg:         config.CapturePlayback{RateMode: config.RatePPS, PacketsPerSec: 10},
			sentPackets: 0,
			wantZero:    true,
		},
		{
			name:        "pps paces to rate",
			cfg:         config.CapturePlayback{RateMode: config.RatePPS, PacketsPerSec: 10},
			sentPackets: 100,
			lo:          9 * time.Second,
			hi:          10 * time.Second,
		},
		{
			name:      "mbps first byte is immediate",
			cfg:       config.CapturePlayback{RateMode: config.RateMbps, MbpsCap: 1},
			sentBytes: 0,
			wantZero:  true,
		},
		// 1_250_000 B = 10_000_000 bits; at 1 Mbps that is 10s of target elapsed.
		{
			name:      "mbps caps throughput",
			cfg:       config.CapturePlayback{RateMode: config.RateMbps, MbpsCap: 1},
			sentBytes: 1_250_000,
			lo:        9 * time.Second,
			hi:        10 * time.Second,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg := tc.cfg
			pb := &PlaybackEngine{config: &cfg, stopChan: make(chan struct{})}

			got := pb.pacingDelay(pkt, tc.sentPackets, tc.sentBytes, start, start)
			if tc.wantZero {
				if got != 0 {
					t.Fatalf("pacingDelay = %v, want 0", got)
				}

				return
			}
			if got <= tc.lo || got > tc.hi {
				t.Fatalf("pacingDelay = %v, want in (%v, %v]", got, tc.lo, tc.hi)
			}
		})
	}
}

// TestPacingDelay_TimingModeDelegates confirms the default and explicit timing
// modes fall through to calculatePacketDelay (original inter-packet spacing).
func TestPacingDelay_TimingModeDelegates(t *testing.T) {
	start := time.Now()
	first := PlaybackPacket{Timestamp: start}
	later := PlaybackPacket{Timestamp: start.Add(500 * time.Millisecond)}

	for _, mode := range []config.RateMode{"", config.RateTiming} {
		cfg := config.CapturePlayback{RateMode: mode}
		pb := &PlaybackEngine{config: &cfg, stopChan: make(chan struct{})}

		// calculatePacketDelay reads time.Now() internally, so two calls differ
		// by sub-microsecond wall-clock drift — compare within a small tolerance.
		want := pb.calculatePacketDelay(later, start, first.Timestamp)
		got := pb.pacingDelay(later, 1, 0, start, first.Timestamp)
		if diff := got - want; diff < -time.Millisecond || diff > time.Millisecond {
			t.Fatalf("mode %q: pacingDelay = %v, want ≈ calculatePacketDelay = %v", mode, got, want)
		}
	}
}

// TestReplay_Topspeed_IgnoresCapturedTiming proves the governor is wired into
// the real send loop: a capture whose packets are 10ms apart (≈190ms in timing
// mode) replays near-instantly under topspeed.
func TestReplay_Topspeed_IgnoresCapturedTiming(t *testing.T) {
	const count = 20

	pb, sender := newTestPlayback(t, &config.CapturePlayback{
		FileName: createTestPCAPFile(t, count),
		RateMode: config.RateTopspeed,
	})

	if err := pb.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer pb.Stop()

	// Topspeed should send all 20 well under the ~190ms the captured timing
	// would impose. Poll briefly so the test isn't tied to a fixed sleep.
	deadline := time.Now().Add(100 * time.Millisecond)
	for time.Now().Before(deadline) {
		if pb.Progress().PacketsSent == count {
			if got := sender.Count(); got != count {
				t.Fatalf("sender got %d frames, want %d", got, count)
			}

			return
		}
		time.Sleep(2 * time.Millisecond)
	}
	t.Fatalf("topspeed sent %d/%d packets within 100ms", pb.Progress().PacketsSent, count)
}
