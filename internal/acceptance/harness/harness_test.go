//go:build acceptance

// The acceptance tag exists so this suite never runs against the working
// tree's own build. It drives the binary named by NIAC_ACCEPTANCE_BINARY,
// which is a release artifact; running it any other way would report a pass
// the release had not earned.
package harness_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/MustardSeedNetworks/niac-go/internal/acceptance/harness"
	"github.com/MustardSeedNetworks/niac-go/internal/cliclient"
)

func startDaemon(t *testing.T) *harness.Daemon {
	t.Helper()
	daemon, err := harness.Start(t.Context(), harness.Options{Root: t.TempDir()})
	if err != nil {
		t.Fatalf("start the released daemon: %v", err)
	}
	t.Cleanup(func() {
		if stopErr := daemon.Stop(); stopErr != nil {
			t.Errorf("stop the daemon: %v", stopErr)
		}
	})
	return daemon
}

// The binary has to be the release, not a bare `go build`: an empty
// uiBuildHash means the UI was never embedded and the ldflags never ran.
func TestReleasedBinaryReportsItsBuild(t *testing.T) {
	daemon := startDaemon(t)

	version, err := daemon.Client.Version(t.Context())
	if err != nil {
		t.Fatalf("read /__version: %v\n%s", err, daemon.Log())
	}
	for field, value := range map[string]string{
		"version":     version.Version,
		"commitFull":  version.CommitFull,
		"uiBuildHash": version.UIBuildHash,
	} {
		if value == "" || value == "unknown" {
			t.Errorf("%s = %q; the binary was not built through the make pipeline", field, value)
		}
	}

	// The operator reads the banner, not /__version, when a daemon starts.
	// The two naming the same build is the only thing that makes the banner
	// worth printing.
	banner := "Starting NIAC Daemon " + version.Version
	if !strings.Contains(daemon.Log(), banner) {
		t.Errorf("no %q in the daemon log:\n%s", banner, daemon.Log())
	}
}

// Every mutation the harness makes goes through the bearer and CSRF
// middleware. A client without the token must be refused, or the acceptance
// run proves nothing about the shipped auth.
func TestReleasedBinaryRefusesAnUnauthenticatedMutation(t *testing.T) {
	daemon := startDaemon(t)

	anonymous, err := cliclient.New(cliclient.Config{BaseURL: daemon.BaseURL, Insecure: true})
	if err != nil {
		t.Fatal(err)
	}
	err = anonymous.SetDeviceFault(t.Context(), cliclient.DeviceFaultRequest{
		Device: "edge-1", Type: "latency", Value: 250,
	})
	if err == nil {
		t.Fatal("an unauthenticated fault injection succeeded")
	}
	if !errors.Is(err, cliclient.ErrUnauthorized) && !errors.Is(err, cliclient.ErrForbidden) {
		t.Fatalf("error = %v, want unauthorized or forbidden", err)
	}
}

// Checkpoints are session-scoped, so naming a session that is not running has
// to answer 404 rather than acting on whichever session happens to be
// selected.
func TestReleasedBinaryRefusesACheckpointOnAnAbsentSession(t *testing.T) {
	daemon := startDaemon(t)

	_, err := daemon.Client.SaveCheckpoint(t.Context(), "not-running", "healthy")
	if err == nil {
		t.Fatal("saving a checkpoint on an absent session succeeded")
	}
	if !errors.Is(err, cliclient.ErrRequestFailed) {
		t.Fatalf("error = %v, want a request failure", err)
	}
}

// A preflight that cannot be satisfied must come back as diagnostics, not as
// an opaque 500 (P1-13): the harness reads that report to decide whether a
// scenario is worth starting.
func TestReleasedBinaryPreflightsWithoutStarting(t *testing.T) {
	daemon := startDaemon(t)

	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()
	report, err := daemon.Client.PreflightSimulation(ctx, cliclient.SimulationRequest{
		Interface: "nic-absent0",
		ConfigData: "devices:\n" +
			"  - name: edge-1\n" +
			"    type: switch\n" +
			"    ip_addresses: [192.0.2.1]\n",
	})
	if err != nil {
		t.Fatalf("preflight: %v\n%s", err, daemon.Log())
	}
	if report.Safe {
		t.Fatal("preflight called an unknown interface safe")
	}
	if len(report.Diagnostics) == 0 {
		t.Fatal("preflight refused the scenario without saying why")
	}

	sessions, err := daemon.Client.Sessions(ctx)
	if err != nil {
		t.Fatalf("list sessions: %v", err)
	}
	if len(sessions) != 0 {
		t.Fatalf("preflight started %d session(s)", len(sessions))
	}
}
