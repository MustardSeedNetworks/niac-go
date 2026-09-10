//go:build linux && integration

package wiretest_test

import (
	"context"
	"errors"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/gosnmp/gosnmp"

	"github.com/MustardSeedNetworks/niac-go/internal/acceptance/harness"
	"github.com/MustardSeedNetworks/niac-go/internal/cliclient"
	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/fabric"
	"github.com/MustardSeedNetworks/niac-go/internal/scenario"
)

// P3-1's sequence -- start, checkpoint, mutate, reset, stop -- against the
// binary a release ships, on a real wire.
//
// The unprivileged half of this lives in internal/acceptance/harness and can
// only reach the routes that need no running scenario. Starting one needs a
// NIC and raw sockets, so the full sequence belongs here, where the namespace
// and the veth pair already exist.
//
// Every assertion is on the wire, not on the API's own account of itself: the
// daemon reporting a fault cleared is the same daemon that reported it armed,
// and a reset that only convinced the API would be invisible to a consumer.
const (
	acceptanceSession   = "acceptance"
	acceptanceDevice    = "e2e-rtr-01"
	acceptanceAddr      = "10.77.0.1"
	acceptanceOID       = ".1.3.6.1.2.1.1.5.0" // sysName.0
	acceptanceCommunity = "e2e_public"
	acceptanceSNMPPort  = 161

	// A latency well past the SNMP timeout below, so a faulted device is
	// silent within the window rather than merely slow.
	acceptanceLatencyMS = 5000
	acceptanceSNMPWait  = 2 * time.Second
	// The reply has to arrive within this after a reset, or the reset did
	// not take.
	acceptanceRecovery = 20 * time.Second
)

func startAcceptanceDaemon(t *testing.T) (*harness.Daemon, *config.Device) {
	t.Helper()
	requireWire(t)

	pack := acceptancePack(t)
	generated, err := scenario.Generate(pack.Request)
	if err != nil {
		t.Fatalf("scenario.Generate(hospital): %v", err)
	}
	authored, err := config.LoadYAMLBytes(generated.YAML)
	if err != nil {
		t.Fatalf("loading the generated hospital YAML: %v", err)
	}

	daemon, err := harness.Start(t.Context(), harness.Options{
		Root: t.TempDir(),
		AttachmentPolicies: []string{
			fmt.Sprintf("%s=access:%d", simIface, accessVLAN),
		},
	})
	if err != nil {
		t.Fatalf("start the released daemon: %v", err)
	}
	t.Cleanup(func() {
		if stopErr := daemon.Stop(); stopErr != nil {
			t.Errorf("stop the daemon: %v", stopErr)
		}
	})

	request := cliclient.SimulationRequest{
		SessionID:      acceptanceSession,
		Interface:      simIface,
		Attachment:     pack.Request.AttachmentName,
		AttachmentMode: fabric.ModeAccess,
		AccessVLAN:     accessVLAN,
		ConfigData:     string(generated.YAML),
	}
	report, err := daemon.Client.PreflightSimulation(t.Context(), request)
	if err != nil {
		t.Fatalf("preflight: %v\n%s", err, daemon.Log())
	}
	if !report.Safe {
		t.Fatalf("preflight refused the scenario: %+v", report.Diagnostics)
	}
	if _, err = daemon.Client.StartSimulation(t.Context(), request); err != nil {
		t.Fatalf("start: %v\n%s", err, daemon.Log())
	}

	return daemon, edgeRouter(t, authored)
}

func acceptancePack(t *testing.T) scenario.Pack {
	t.Helper()
	for _, candidate := range scenario.Packs() {
		if candidate.ID == "hospital" {
			return candidate
		}
	}
	t.Fatal("no pack with id \"hospital\"; scenario.Packs() no longer ships it")

	return scenario.Pack{}
}

// The whole sequence in one test, because the halves prove nothing apart: a
// device that answers before the fault says nothing about the reset, and one
// that answers after it says nothing unless it had actually gone quiet.
func TestReleasedBinaryCheckpointsMutatesAndResets(t *testing.T) {
	daemon, edge := startAcceptanceDaemon(t)
	ctx := t.Context()
	community := edge.SNMPConfig.Community
	if community == "" {
		t.Fatalf("%s has no authored SNMP community", edge.Name)
	}

	if !answersSNMP(t, community, acceptanceSNMPWait) {
		t.Fatalf("the scenario never answered SNMP before the checkpoint\n%s", daemon.Log())
	}

	devices, err := daemon.Client.SaveCheckpoint(ctx, acceptanceSession, "healthy")
	if err != nil {
		t.Fatalf("save the checkpoint: %v\n%s", err, daemon.Log())
	}
	if devices == 0 {
		t.Fatal("the checkpoint covered no devices")
	}

	err = daemon.Client.SetDeviceFault(ctx, cliclient.DeviceFaultRequest{
		Device: edge.Name, Type: "latency", Value: acceptanceLatencyMS,
	})
	if err != nil {
		t.Fatalf("inject latency: %v\n%s", err, daemon.Log())
	}
	if answersSNMP(t, community, acceptanceSNMPWait) {
		t.Fatal("the device answered inside the window while a 5s latency was armed")
	}

	if err = daemon.Client.RestoreCheckpoint(ctx, acceptanceSession, "healthy"); err != nil {
		t.Fatalf("restore the checkpoint: %v\n%s", err, daemon.Log())
	}
	deadline := time.Now().Add(acceptanceRecovery)
	recovered := false
	for time.Now().Before(deadline) && !recovered {
		recovered = answersSNMP(t, community, acceptanceSNMPWait)
	}
	if !recovered {
		t.Fatalf("the device never answered again after the reset\n%s", daemon.Log())
	}

	if err = daemon.Client.StopSimulation(ctx, acceptanceSession); err != nil {
		t.Fatalf("stop the session: %v\n%s", err, daemon.Log())
	}
	sessions, err := daemon.Client.Sessions(ctx)
	if err != nil {
		t.Fatalf("list sessions: %v", err)
	}
	for _, session := range sessions {
		if session.SessionID == acceptanceSession {
			t.Fatalf("session %s survived its stop", acceptanceSession)
		}
	}
}

// answersSNMP reports whether the authored router replies to a sysName GET
// inside the window. A timeout is the expected answer while a fault is armed,
// so it is a result rather than a failure.
func answersSNMP(t *testing.T, community string, within time.Duration) bool {
	t.Helper()
	client := &gosnmp.GoSNMP{
		Target:    transitGateway,
		Port:      acceptanceSNMPPort,
		Community: community,
		Version:   gosnmp.Version2c,
		Timeout:   within,
		Retries:   0,
		Context:   context.Background(),
	}
	if err := client.Connect(); err != nil {
		t.Fatalf("connect to %s: %v", transitGateway, err)
	}
	defer client.Conn.Close()

	result, err := client.Get([]string{acceptanceOID})
	if err != nil {
		var timeout net.Error
		if errors.As(err, &timeout) && timeout.Timeout() {
			return false
		}
		t.Fatalf("SNMP GET %s: %v", acceptanceOID, err)
	}

	return len(result.Variables) == 1 && result.Variables[0].Value != nil
}
