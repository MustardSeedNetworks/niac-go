//go:build linux && integration

package wiretest_test

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/layers"
	"github.com/gopacket/gopacket/pcap"
	"github.com/gopacket/gopacket/pcapgo"

	"github.com/MustardSeedNetworks/niac-go/internal/api"
	"github.com/MustardSeedNetworks/niac-go/internal/capture"
	"github.com/MustardSeedNetworks/niac-go/internal/replay"
)

// PCAP playback is the one part of the simulation that claims to reproduce a
// recording rather than answer a request, and until now only its pacing
// arithmetic was tested — in process, against a recording sender. What that
// cannot see is the wire: whether the bytes libpcap writes are the bytes the
// far end reads, whether a bounded loop sends the passes it promised, and
// whether "original timing" survives contact with a real scheduler.
//
// These tests replay a synthetic capture through the production controller
// (internal/replay, the copy both the daemon and the CLI use) onto one end of
// the veth and read the frames back off the other.
//
// Note, not a defect these tests can fix: the config-authored path cannot
// reach most of this. daemon.startConfiguredReplay (internal/daemon/daemon.go)
// forwards only file, LoopMs and Scale, and yaml_load.go copies only those
// three fields out of capture_playbacks, so RateMode, LoopCount and BPFFilter
// are API-only today. The tests drive the controller directly, which is what
// both callers do.

const (
	// A local-experimental EtherType (IEEE 802a) rather than IP, for two
	// reasons. The kernel offloads (GRO/GSO) coalesce and re-segment IP on a
	// veth, which would make a frame count meaningless; and an EtherType
	// nothing else on this wire speaks is what lets the capture filter
	// separate replayed frames from simulation traffic.
	fidelityEtherType = 0x88b5

	// A second local-experimental EtherType carries the warm-up probe, so a
	// probe frame is never mistaken for a replayed one. See primeCapture.
	fidelityProbeEtherType = 0x88b6

	// Frames are padded past the 60-byte Ethernet minimum so the NIC's own
	// padding can never appear as a byte difference, and stay well under the
	// 1500-byte MTU so nothing fragments.
	fidelityFrameLen = 128

	// Ethernet II header geometry, used to size payloads and to read a
	// captured frame's EtherType.
	ethernetHeaderLen  = 14
	ethernetTypeOffset = 12

	fidelityFrames = 12 // packets in the synthetic capture
	fidelityLoops  = 3  // LoopCount for the byte/count assertions

	// Source spacing for the timing test, and the tolerance allowed on each
	// arrival offset. 150 ms is far wider than the millisecond-scale jitter of
	// time.After plus libpcap delivery, and topspeed — the failure mode this
	// is meant to catch — arrives at ~0 ms, roughly four tolerances early by
	// the second frame. 40 ms leaves room for a loaded build host (this box
	// routinely runs several Go suites at once) without letting an unpaced
	// replay pass.
	fidelitySourceGap  = 150 * time.Millisecond
	fidelityTolerance  = 40 * time.Millisecond
	fidelityReadWindow = 30 * time.Second

	// Bounds for the warm-up probe: how long to keep probing before giving up
	// on the wire entirely, and how long to wait for each probe to come back.
	fidelityPrimeWindow = 5 * time.Second
	fidelityPrimeRetry  = 100 * time.Millisecond

	// The reading end needs the same generous kernel ring the product's own
	// capture engine sets (internal/capture/capture.go). Measured, not
	// guessed: with libpcap's default ring, one run in eight of the topspeed
	// case lost frames — always to ps_drop, never in transit, the sender's
	// count intact — because a 36-frame burst arrives faster than the reader
	// drains it. That is the reader's ring overflowing, which is why the
	// shortfall message below reports the drop counters.
	fidelityRingBytes = 16 << 20
)

// TestPcapFidelityReplaysEveryFrameByteIdenticalInOrder asserts the three
// things a replay is for: the same bytes, the promised number of passes, and
// the source order preserved across pass boundaries.
func TestPcapFidelityReplaysEveryFrameByteIdenticalInOrder(t *testing.T) {
	requireWire(t)

	source := fidelityFixtureFrames(t)
	dir, file := writeFidelityPCAP(t, source, fidelitySourceGap)
	engine, handle := openFidelityWire(t)
	controller := startFidelityReplay(t, engine, api.ReplayRequest{
		File:      file,
		RootDir:   dir,
		RateMode:  "topspeed",
		LoopCount: fidelityLoops,
	})

	want := len(source) * fidelityLoops
	got, _ := readFidelityFrames(t, handle, want, fidelityReadWindow)
	if len(got) != want {
		t.Fatalf(
			"read %d replayed frames, want %d (%d packets × %d passes); "+
				"controller reports %d passes and %d packets sent; %s",
			len(got), want, len(source), fidelityLoops,
			controller.Status().Passes, controller.Status().PacketsSent,
			captureLosses(handle),
		)
	}
	for i, frame := range got {
		expected := source[i%len(source)]
		if !bytes.Equal(frame, expected) {
			t.Fatalf(
				"frame %d (pass %d, source packet %d) differs from the source:\n got %d bytes % x\nwant %d bytes % x",
				i, i/len(source), i%len(source), len(frame), frame, len(expected), expected,
			)
		}
	}

	// A bounded LoopCount must also stop. Anything still arriving after the
	// promised passes is a replay that ignored its own bound.
	extra, _ := readFidelityFrames(t, handle, 1, 2*time.Second)
	if len(extra) != 0 {
		t.Errorf("replay sent %d frame(s) after its %d passes finished", len(extra), fidelityLoops)
	}
}

// TestPcapFidelityHonoursSourceInterArrivalTiming asserts that timing mode
// reproduces the capture's own spacing on the wire.
//
// The comparison is each frame's offset from the FIRST arrival, not the gap
// from the previous one, because that is what the engine actually schedules:
// pacingDelay targets startTime+(ts-firstTs) absolutely, so a frame delayed by
// the scheduler does not push the next one out. Gap-to-gap differencing would
// therefore report a legitimate catch-up as an error.
func TestPcapFidelityHonoursSourceInterArrivalTiming(t *testing.T) {
	requireWire(t)

	source := fidelityFixtureFrames(t)
	dir, file := writeFidelityPCAP(t, source, fidelitySourceGap)
	engine, handle := openFidelityWire(t)
	startFidelityReplay(t, engine, api.ReplayRequest{
		File:     file,
		RootDir:  dir,
		RateMode: "timing",
		Scale:    1,
	})

	got, at := readFidelityFrames(t, handle, len(source), fidelityReadWindow)
	if len(got) != len(source) {
		t.Fatalf("read %d frames in timing mode, want %d", len(got), len(source))
	}
	for i := 1; i < len(at); i++ {
		wantOffset := time.Duration(i) * fidelitySourceGap
		gotOffset := at[i].Sub(at[0])
		if drift := gotOffset - wantOffset; drift < -fidelityTolerance || drift > fidelityTolerance {
			t.Errorf(
				"frame %d arrived %v after the first frame, want %v (±%v): off by %v",
				i, gotOffset.Round(time.Millisecond), wantOffset, fidelityTolerance,
				drift.Round(time.Millisecond),
			)
		}
	}
}

// fidelityFixtureFrames builds the synthetic capture's frames. Each carries
// its own index in the payload, so "arrived in order" is an assertion about
// identity rather than about a run of interchangeable bytes.
func fidelityFixtureFrames(t *testing.T) [][]byte {
	t.Helper()

	src := net.HardwareAddr{0x02, 0x00, 0x00, 0x0f, 0x1d, 0x01}
	dst := net.HardwareAddr{0x02, 0x00, 0x00, 0x0f, 0x1d, 0x02}
	frames := make([][]byte, 0, fidelityFrames)
	for i := range fidelityFrames {
		payload := []byte(fmt.Sprintf("niac pcap fidelity frame %03d ", i))
		for len(payload) < fidelityFrameLen-ethernetHeaderLen {
			payload = append(payload, byte(i))
		}
		frame := serialize(t, &layers.Ethernet{
			SrcMAC:       src,
			DstMAC:       dst,
			EthernetType: layers.EthernetType(fidelityEtherType),
		}, gopacket.Payload(payload[:fidelityFrameLen-ethernetHeaderLen]))
		if len(frame) != fidelityFrameLen {
			t.Fatalf("fixture frame %d is %d bytes, want %d", i, len(frame), fidelityFrameLen)
		}
		frames = append(frames, frame)
	}
	return frames
}

// writeFidelityPCAP writes the frames to a classic pcap in a fresh temp
// directory, spaced by gap, and returns the directory (the replay root) and
// the file path.
func writeFidelityPCAP(t *testing.T, frames [][]byte, gap time.Duration) (string, string) {
	t.Helper()

	dir := t.TempDir()
	path := filepath.Join(dir, "fidelity.pcap")
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("creating %s: %v", path, err)
	}
	defer func() { _ = f.Close() }()

	w := pcapgo.NewWriter(f)
	if headerErr := w.WriteFileHeader(65536, layers.LinkTypeEthernet); headerErr != nil {
		t.Fatalf("writing pcap header: %v", headerErr)
	}
	base := time.Now().Add(-time.Hour)
	for i, frame := range frames {
		info := gopacket.CaptureInfo{
			Timestamp:     base.Add(time.Duration(i) * gap),
			CaptureLength: len(frame),
			Length:        len(frame),
		}
		if writeErr := w.WritePacket(info, frame); writeErr != nil {
			t.Fatalf("writing pcap packet %d: %v", i, writeErr)
		}
	}
	return dir, path
}

// startFidelityReplay replays through the production controller — the copy
// internal/daemon and cmd/niac both use — and stops it when the test ends.
func startFidelityReplay(
	t *testing.T,
	engine *capture.Engine,
	req api.ReplayRequest,
) *replay.Controller {
	t.Helper()

	controller := replay.New(engine, 0)
	t.Cleanup(func() {
		if _, stopErr := controller.Stop(); stopErr != nil {
			t.Errorf("stopping replay: %v", stopErr)
		}
	})
	if _, startErr := controller.Start(req); startErr != nil {
		t.Fatalf("starting replay of %s: %v", req.File, startErr)
	}
	return controller
}

// openFidelityWire opens both ends of the veth and returns them only once a
// frame has made the crossing, so no test starts a replay into a capture path
// that is not carrying yet.
func openFidelityWire(t *testing.T) (*capture.Engine, *pcap.Handle) {
	t.Helper()

	engine, err := capture.New(simIface, 0)
	if err != nil {
		t.Fatalf("capture.New(%s): %v", simIface, err)
	}
	t.Cleanup(engine.Close)

	handle := openFidelityClient(t)
	primeCapture(t, engine, handle)

	return engine, handle
}

// primeCapture sends probe frames until one is read back.
//
// This is not a sleep in disguise: an activated libpcap handle on a freshly
// created veth does not necessarily see the very next frame, and the first
// cold run of these tests lost the opening four frames of a topspeed replay to
// exactly that. Waiting for a *fixed* interval would be a guess; waiting for a
// probe to complete the round trip proves the property the replay depends on.
// The probe rides its own EtherType so it can never be counted as a replayed
// frame, and readFidelityFrames drops any probe still sitting in the ring.
func primeCapture(t *testing.T, engine *capture.Engine, handle *pcap.Handle) {
	t.Helper()

	probe := serialize(t, &layers.Ethernet{
		SrcMAC:       net.HardwareAddr{0x02, 0x00, 0x00, 0x0f, 0x1d, 0x01},
		DstMAC:       net.HardwareAddr{0x02, 0x00, 0x00, 0x0f, 0x1d, 0x02},
		EthernetType: layers.EthernetType(fidelityProbeEtherType),
	}, gopacket.Payload(bytes.Repeat([]byte("niac fidelity probe "), 3)))

	deadline := time.Now().Add(fidelityPrimeWindow)
	for time.Now().Before(deadline) {
		if err := engine.SendPacket(probe); err != nil {
			t.Fatalf("sending the warm-up probe on %s: %v", simIface, err)
		}
		seen, _, err := readFrames(handle, fidelityProbeEtherType, 1, fidelityPrimeRetry)
		if err != nil {
			t.Fatalf("reading the warm-up probe on %s: %v", testIface, err)
		}
		if len(seen) > 0 {
			return
		}
	}
	t.Fatalf("no probe frame crossed %s → %s within %v", simIface, testIface, fidelityPrimeWindow)
}

// openFidelityClient opens the test end filtered to the fixture's two
// EtherTypes, so simulation traffic on the same wire cannot enter the
// comparison. The read timeout is short rather than BlockForever because the
// read loop enforces its own deadline in the same goroutine.
func openFidelityClient(t *testing.T) *pcap.Handle {
	t.Helper()

	inactive, err := pcap.NewInactiveHandle(testIface)
	if err != nil {
		t.Fatalf("pcap.NewInactiveHandle(%s): %v", testIface, err)
	}
	defer inactive.CleanUp()
	_ = inactive.SetSnapLen(65536)
	_ = inactive.SetPromisc(true)
	_ = inactive.SetTimeout(20 * time.Millisecond)
	_ = inactive.SetImmediateMode(true)
	_ = inactive.SetBufferSize(fidelityRingBytes)

	handle, err := inactive.Activate()
	if err != nil {
		t.Fatalf("activating pcap on %s: %v", testIface, err)
	}
	t.Cleanup(handle.Close)
	filter := fmt.Sprintf(
		"ether proto 0x%04x or ether proto 0x%04x",
		fidelityEtherType, fidelityProbeEtherType,
	)
	if filterErr := handle.SetBPFFilter(filter); filterErr != nil {
		t.Fatalf("setting the fidelity capture filter: %v", filterErr)
	}
	return handle
}

// readFidelityFrames reads replayed frames, failing the test on a read error.
func readFidelityFrames(
	t *testing.T,
	handle *pcap.Handle,
	want int,
	window time.Duration,
) ([][]byte, []time.Time) {
	t.Helper()

	frames, at, err := readFrames(handle, fidelityEtherType, want, window)
	if err != nil {
		t.Fatalf("reading from %s: %v", testIface, err)
	}
	return frames, at
}

// captureLosses renders libpcap's own counters, which separate the two
// explanations for a short read that otherwise look identical: frames the
// replay never sent, and frames this reader's kernel ring dropped.
func captureLosses(handle *pcap.Handle) string {
	stats, err := handle.Stats()
	if err != nil {
		return fmt.Sprintf("capture stats unavailable: %v", err)
	}
	return fmt.Sprintf(
		"libpcap received %d, dropped %d, interface-dropped %d",
		stats.PacketsReceived, stats.PacketsDropped, stats.PacketsIfDropped,
	)
}

// readFrames reads up to want frames of the given EtherType, or until the
// window closes, returning what arrived and libpcap's own timestamp for each —
// the kernel's receive time, not this loop's, so a slow reader cannot be
// mistaken for a slow sender. Frames of any other EtherType (a warm-up probe
// left in the ring) are discarded. Returning short is normal: the caller
// decides whether that is a failure or the point of the assertion.
func readFrames(
	handle *pcap.Handle,
	etherType uint16,
	want int,
	window time.Duration,
) ([][]byte, []time.Time, error) {
	frames := make([][]byte, 0, want)
	at := make([]time.Time, 0, want)
	deadline := time.Now().Add(window)
	for len(frames) < want && time.Now().Before(deadline) {
		data, info, err := handle.ReadPacketData()
		if err != nil {
			if errors.Is(err, pcap.NextErrorTimeoutExpired) {
				continue
			}
			return frames, at, err
		}
		if len(data) < ethernetHeaderLen ||
			binary.BigEndian.Uint16(data[ethernetTypeOffset:ethernetHeaderLen]) != etherType {
			continue
		}
		frames = append(frames, bytes.Clone(data))
		at = append(at, info.Timestamp)
	}
	return frames, at, nil
}
