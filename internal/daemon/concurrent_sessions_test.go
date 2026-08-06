package daemon

import (
	"errors"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/api"
	"github.com/MustardSeedNetworks/niac-go/internal/fabric"
)

func TestDaemonRunsDistinctTrunkSessionsConcurrently(t *testing.T) {
	t.Setenv(e2eDryRunEnv, "true")
	t.Setenv("NIAC_CONFIGS_DIR", t.TempDir())
	d, err := NewDaemon(Config{AttachmentPolicies: []fabric.PhysicalAttachmentPolicy{{
		Interface: "eth0", Mode: fabric.ModeTrunk, AllowedVLANs: []uint16{200, 201},
	}}})
	if err != nil {
		t.Fatalf("NewDaemon(): %v", err)
	}

	if err = d.StartSimulation(trunkSessionRequest("hospital", 200), fullSimulationEntitlements()); err != nil {
		t.Fatalf("StartSimulation(hospital): %v", err)
	}
	if err = d.StartSimulation(trunkSessionRequest("warehouse", 201), fullSimulationEntitlements()); err != nil {
		t.Fatalf("StartSimulation(warehouse): %v", err)
	}
	if got := d.sessions.len(); got != 2 {
		t.Fatalf("active sessions = %d, want 2", got)
	}
	if d.sessions.get("hospital") == nil || d.sessions.get("warehouse") == nil {
		t.Fatalf("sessions = %#v", d.sessions.sessions)
	}
	if d.sessions.get("hospital").ConfigPath == d.sessions.get("warehouse").ConfigPath {
		t.Fatalf("concurrent sessions share config path %q", d.sessions.get("hospital").ConfigPath)
	}
	status := d.GetStatus()
	// Starting warehouse must not take the default away from hospital.
	// Adopting every new session repointed anyone reading the unscoped
	// surface away from the scenario they were already watching.
	if status.SessionID != "hospital" || len(status.Sessions) != 2 {
		t.Fatalf("GetStatus() session = %q, sessions = %#v", status.SessionID, status.Sessions)
	}
	if status.Sessions[0].SessionID != "hospital" || status.Sessions[0].PhysicalVLAN != 200 ||
		status.Sessions[1].SessionID != "warehouse" || status.Sessions[1].PhysicalVLAN != 201 {
		t.Fatalf("GetStatus().Sessions = %#v", status.Sessions)
	}

	if err = d.StopSimulation(""); err != nil {
		t.Fatalf("StopSimulation(): %v", err)
	}
	if got := d.sessions.len(); got != 1 {
		t.Fatalf("active sessions after stop = %d, want 1", got)
	}
}

func TestDaemonRejectsDuplicateTrunkVLAN(t *testing.T) {
	t.Setenv(e2eDryRunEnv, "true")
	t.Setenv("NIAC_CONFIGS_DIR", t.TempDir())
	d, err := NewDaemon(Config{AttachmentPolicies: []fabric.PhysicalAttachmentPolicy{{
		Interface: "eth0", Mode: fabric.ModeTrunk, AllowedVLANs: []uint16{200, 201},
	}}})
	if err != nil {
		t.Fatalf("NewDaemon(): %v", err)
	}
	if err = d.StartSimulation(trunkSessionRequest("hospital", 200), fullSimulationEntitlements()); err != nil {
		t.Fatalf("StartSimulation(hospital): %v", err)
	}

	err = d.StartSimulation(trunkSessionRequest("warehouse", 200), fullSimulationEntitlements())
	if !errors.Is(err, ErrPhysicalVLANInUse) {
		t.Fatalf("StartSimulation(warehouse) error = %v, want ErrPhysicalVLANInUse", err)
	}
	if got := d.sessions.len(); got != 1 {
		t.Fatalf("active sessions = %d, want 1", got)
	}
}

func TestDaemonRejectsUnapprovedFlatTrunkVLAN(t *testing.T) {
	t.Setenv(e2eDryRunEnv, "true")
	t.Setenv("NIAC_CONFIGS_DIR", t.TempDir())
	d, err := NewDaemon(Config{AttachmentPolicies: []fabric.PhysicalAttachmentPolicy{{
		Interface: "eth0", Mode: fabric.ModeTrunk, AllowedVLANs: []uint16{200},
	}}})
	if err != nil {
		t.Fatalf("NewDaemon(): %v", err)
	}

	err = d.StartSimulation(trunkSessionRequest("warehouse", 201), fullSimulationEntitlements())
	if !errors.Is(err, ErrUnsafeTopology) {
		t.Fatalf("StartSimulation() error = %v, want ErrUnsafeTopology", err)
	}
}

func TestDaemonConfiguresFlatTrunkPhysicalVLAN(t *testing.T) {
	t.Setenv(e2eDryRunEnv, "true")
	t.Setenv("NIAC_CONFIGS_DIR", t.TempDir())
	d, err := NewDaemon(Config{AttachmentPolicies: []fabric.PhysicalAttachmentPolicy{{
		Interface: "eth0", Mode: fabric.ModeTrunk, AllowedVLANs: []uint16{200},
	}}})
	if err != nil {
		t.Fatalf("NewDaemon(): %v", err)
	}
	if err = d.StartSimulation(
		trunkSessionRequest("hospital", 200), fullSimulationEntitlements(),
	); err != nil {
		t.Fatalf("StartSimulation(): %v", err)
	}
	topology := d.sessions.get("hospital").fabric
	if topology == nil || !topology.Binding.WireTagged || topology.Binding.AccessVLAN != 200 {
		t.Fatalf("flat trunk topology = %#v", topology)
	}
}

func trunkSessionRequest(sessionID string, vlan uint16) api.SimulationRequest {
	return api.SimulationRequest{
		SessionID:      sessionID,
		Interface:      "eth0",
		AttachmentMode: fabric.ModeTrunk,
		AccessVLAN:     vlan,
		ConfigData:     "devices:\n  - name: demo-device\n    type: host\n    mac: 02:00:00:00:00:01\n",
	}
}

func TestStartingASessionDoesNotStealTheDefaultFromAnother(t *testing.T) {
	// Whoever reads the unscoped surface picked their scenario. Launching a
	// different one must not silently move them onto it — that is what made
	// "start the warehouse demo" change what the hospital demo was showing.
	t.Setenv(e2eDryRunEnv, "true")
	t.Setenv("NIAC_CONFIGS_DIR", t.TempDir())
	d, err := NewDaemon(Config{AttachmentPolicies: []fabric.PhysicalAttachmentPolicy{{
		Interface: "eth0", Mode: fabric.ModeTrunk, AllowedVLANs: []uint16{200, 201},
	}}})
	if err != nil {
		t.Fatalf("NewDaemon(): %v", err)
	}
	for _, session := range []struct {
		id   string
		vlan uint16
	}{{"hospital", 200}, {"warehouse", 201}} {
		if err = d.StartSimulation(
			trunkSessionRequest(session.id, session.vlan), fullSimulationEntitlements(),
		); err != nil {
			t.Fatalf("StartSimulation(%s): %v", session.id, err)
		}
	}

	if got := d.GetStatus().SessionID; got != "hospital" {
		t.Fatalf("default session = %q, want hospital to keep it", got)
	}

	// Restarting the session that holds the default keeps it, rather than
	// leaving the daemon with no default at all.
	if err = d.StartSimulation(
		trunkSessionRequest("hospital", 200), fullSimulationEntitlements(),
	); err != nil {
		t.Fatalf("StartSimulation(hospital) restart: %v", err)
	}
	if got := d.GetStatus().SessionID; got != "hospital" {
		t.Errorf("after restart default session = %q, want hospital", got)
	}

	// Selecting explicitly is still honoured — this is a default, not a lock.
	if err = d.SelectSimulation("warehouse"); err != nil {
		t.Fatal(err)
	}
	if got := d.GetStatus().SessionID; got != "warehouse" {
		t.Errorf("after explicit select = %q, want warehouse", got)
	}
}
