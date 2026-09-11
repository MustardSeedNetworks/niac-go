//go:build linux && integration

package wiretest_test

import (
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/gosnmp/gosnmp"

	"github.com/MustardSeedNetworks/niac-go/internal/acceptance/harness"
	"github.com/MustardSeedNetworks/niac-go/internal/cliclient"
	"github.com/MustardSeedNetworks/niac-go/internal/fabric"
	"github.com/MustardSeedNetworks/niac-go/internal/templates"
)

// P3-1's sequence -- start, checkpoint, mutate, reset, stop -- against the
// binary a release ships, on a real wire.
//
// The unprivileged half lives in internal/acceptance/harness and can only
// reach the routes that need no running scenario. Starting one needs a NIC and
// raw sockets, so the full sequence belongs here, where the namespace and the
// veth pair already exist.
//
// Every assertion is on the wire, not on the API's own account of itself: the
// daemon reporting a fault cleared is the same daemon that reported it armed,
// and a reset that convinced only the API would be invisible to a consumer.
//
// The scenario is the resource-pressure template, and the device is its
// *healthy* peer: DEMO-APP01 is driven by the template's own timeline, so a
// value read there could not be attributed to this test's fault. DEMO-APP02
// stays at its authored baseline unless something here changes it.
const (
	acceptanceSession = "acceptance"
	acceptanceDevice  = "DEMO-APP02"
	acceptanceAddr    = "10.254.200.12"
	acceptanceCommand = "resource_demo"

	// hrProcessorLoad.1 -- the value the CPU fault drives, and one this
	// device's walk actually advertises.
	acceptanceCPUOID = "1.3.6.1.2.1.25.3.3.1.2.1"
	// resource-pressure.yaml: DEMO-APP02's authored load, and the value a
	// reset has to return exactly.
	acceptanceCPUBaseline = 18
	// Far enough from the baseline that no rounding or drift could produce it.
	acceptanceCPUFaulted = 95
	// POST /api/v1/errors matches the fault's display label, not its type.
	acceptanceCPULabel = "CPU Utilization"

	acceptanceSNMPTimeout = time.Second
	acceptanceSNMPRetries = 2
	// A fault reaches the agent through the MIB, not the request path, so a
	// value can lag the API's answer by a poll.
	acceptanceSettle = 15 * time.Second
)

func startAcceptanceDaemon(t *testing.T) *harness.Daemon {
	t.Helper()
	requireWire(t)

	template, err := templates.Get("resource-pressure")
	if err != nil {
		t.Fatalf("templates.Get(resource-pressure): %v", err)
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
		Attachment:     "tester",
		AttachmentMode: fabric.ModeAccess,
		AccessVLAN:     accessVLAN,
		ConfigData:     template.Content,
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

	return daemon
}

// The whole sequence in one test, because the halves prove nothing apart: a
// device at its baseline says nothing about the reset unless it had actually
// moved, and a moved value says nothing unless it came back.
func TestReleasedBinaryCheckpointsMutatesAndResets(t *testing.T) {
	daemon := startAcceptanceDaemon(t)
	ctx := t.Context()
	client := dialAcceptanceHost(t)

	if load := acceptanceCPU(t, client); load != acceptanceCPUBaseline {
		t.Fatalf("%s load = %d before the checkpoint, want %d\n%s",
			acceptanceDevice, load, acceptanceCPUBaseline, daemon.Log())
	}

	devices, err := daemon.Client.SaveCheckpoint(ctx, acceptanceSession, "healthy")
	if err != nil {
		t.Fatalf("save the checkpoint: %v\n%s", err, daemon.Log())
	}
	if devices == 0 {
		t.Fatal("the checkpoint covered no devices")
	}

	err = daemon.Client.SetDeviceFault(ctx, cliclient.DeviceFaultRequest{
		Device: acceptanceDevice, Type: acceptanceCPULabel, Value: acceptanceCPUFaulted,
	})
	if err != nil {
		t.Fatalf("arm the CPU fault: %v\n%s", err, daemon.Log())
	}
	awaitAcceptanceCPU(t, client, acceptanceCPUFaulted, daemon)

	if err = daemon.Client.RestoreCheckpoint(ctx, acceptanceSession, "healthy"); err != nil {
		t.Fatalf("restore the checkpoint: %v\n%s", err, daemon.Log())
	}
	awaitAcceptanceCPU(t, client, acceptanceCPUBaseline, daemon)

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

func dialAcceptanceHost(t *testing.T) *gosnmp.GoSNMP {
	t.Helper()
	client := &gosnmp.GoSNMP{
		Target: acceptanceAddr, Port: 161, Community: acceptanceCommand,
		Version: gosnmp.Version2c,
		Timeout: acceptanceSNMPTimeout, Retries: acceptanceSNMPRetries,
	}
	if err := client.Connect(); err != nil {
		t.Fatalf("connect to %s: %v", acceptanceAddr, err)
	}
	t.Cleanup(func() { _ = client.Conn.Close() })

	return client
}

func acceptanceCPU(t *testing.T, client *gosnmp.GoSNMP) int64 {
	t.Helper()
	result, err := client.Get([]string{acceptanceCPUOID})
	if err != nil {
		t.Fatalf("SNMP GET %s on %s: %v", acceptanceCPUOID, acceptanceAddr, err)
	}
	if len(result.Variables) != 1 {
		t.Fatalf("SNMP GET %s returned %d variables", acceptanceCPUOID, len(result.Variables))
	}
	value, err := strconv.ParseInt(fmt.Sprint(gosnmp.ToBigInt(result.Variables[0].Value)), 10, 64)
	if err != nil {
		t.Fatalf("%s is not an integer: %#v", acceptanceCPUOID, result.Variables[0].Value)
	}

	return value
}

func awaitAcceptanceCPU(t *testing.T, client *gosnmp.GoSNMP, want int64, daemon *harness.Daemon) {
	t.Helper()
	deadline := time.Now().Add(acceptanceSettle)
	var load int64
	for time.Now().Before(deadline) {
		load = acceptanceCPU(t, client)
		if load == want {
			return
		}
		time.Sleep(acceptanceSNMPTimeout)
	}
	t.Fatalf("%s load = %d after %s, want %d\n%s",
		acceptanceDevice, load, acceptanceSettle, want, daemon.Log())
}
