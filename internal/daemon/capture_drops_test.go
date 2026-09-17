package daemon

import (
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/capture"
)

// TestGetCaptureStatusReportsRingDrops is D-NIAC-3's acceptance on
// /api/v1/capture: the status endpoint counted only frames libpcap had
// already delivered, so a capture that overflowed its ring read as complete.
func TestGetCaptureStatusReportsRingDrops(t *testing.T) {
	engine := &fakeCaptureEngine{stats: capture.Stats{
		PacketsReceived:  100,
		PacketsDropped:   7,
		PacketsIfDropped: 3,
	}}
	daemon := newTestDaemon(t)
	daemon.capture = &standaloneCapture{iface: "eth0", engine: engine}

	status := daemon.GetCaptureStatus()

	if status.PacketsDropped != 7 {
		t.Errorf("PacketsDropped = %d, want 7", status.PacketsDropped)
	}
	if status.PacketsIfDropped != 3 {
		t.Errorf("PacketsIfDropped = %d, want 3", status.PacketsIfDropped)
	}
}

// TestGetCaptureStatusSurvivesStatsError: a handle that cannot report its
// counters must not break the status endpoint.
func TestGetCaptureStatusSurvivesStatsError(t *testing.T) {
	engine := &fakeCaptureEngine{statsErr: errFakeCaptureStats}
	daemon := newTestDaemon(t)
	daemon.capture = &standaloneCapture{iface: "eth0", engine: engine}

	status := daemon.GetCaptureStatus()

	if !status.Running {
		t.Error("Running = false, want true")
	}
	if status.PacketsDropped != 0 || status.PacketsIfDropped != 0 {
		t.Errorf("drops = (%d, %d), want (0, 0)",
			status.PacketsDropped, status.PacketsIfDropped)
	}
}

// TestTrunkSessionTransportReportsPhysicalDrops keeps D-NIAC-3's fix alive on
// the VLAN trunk path. A trunk session's stack is built on a session
// transport, not on the capture engine, so without this delegation every
// trunked simulation would report a lossy trunk as complete.
func TestTrunkSessionTransportReportsPhysicalDrops(t *testing.T) {
	physical := &fakeTrunkPhysical{stats: capture.Stats{
		PacketsReceived:  100,
		PacketsDropped:   7,
		PacketsIfDropped: 3,
	}}
	trunk := newTrunkCapture(physical)

	session, err := trunk.register(10)
	if err != nil {
		t.Fatalf("register(10) error = %v", err)
	}

	stats, err := session.Stats()
	if err != nil {
		t.Fatalf("Stats() error = %v", err)
	}
	if stats.PacketsDropped != 7 || stats.PacketsIfDropped != 3 {
		t.Errorf("drops = (%d, %d), want (7, 3)",
			stats.PacketsDropped, stats.PacketsIfDropped)
	}
}
