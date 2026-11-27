//go:build realdaemon && !windows

package integration

import (
	"bytes"
	"context"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/MustardSeedNetworks/niac-go/internal/cliclient"
)

// TestRealDaemon_GeneratedPackRunsOnLoopback is the AUD-2 acceptance: a pack
// goes from `scenario generate --save` through `simulation start` on a real
// interface to a running simulation, and stops, with a real daemon process and
// no UI. The httptest-backed CLI tests prove each command alone; this proves
// the hand-off between them, including the attachment policy the start
// depends on.
//
// NIAC_BIN names a prebuilt niac binary (CI grants it raw-socket capabilities
// first); without it the test builds one.
func TestRealDaemon_GeneratedPackRunsOnLoopback(t *testing.T) {
	loopback := "lo"
	if runtime.GOOS == "darwin" {
		loopback = "lo0"
	}

	niac := os.Getenv("NIAC_BIN")
	if niac == "" {
		niac = filepath.Join(t.TempDir(), "niac")
		build := exec.Command("go", "build", "-o", niac, "../../cmd/niac")
		if out, err := build.CombinedOutput(); err != nil {
			t.Fatalf("build niac: %v\n%s", err, out)
		}
	}

	// Home and library are private so the run never reads or writes the
	// invoking user's ~/.niac, and takes its own instance lock.
	home := t.TempDir()
	env := append(os.Environ(),
		"HOME="+home,
		"NIAC_LIBRARY_ROOT="+filepath.Join(home, "library"),
		"NIAC_CONFIGS_DIR="+filepath.Join(home, "configs"),
		"NIAC_API_URL=",
		"NIAC_API_TOKEN=",
	)
	certDir := filepath.Join(home, "certs")
	api := startRealDaemon(t, niac, env, certDir, loopback)
	connect := []string{"--api", api, "--cacert", filepath.Join(certDir, "server.crt")}

	const saved = "pack-e2e-hospital"
	runNiac(t, niac, env, append([]string{"scenario", "generate", "hospital", "--save", saved}, connect...)...)
	savedFile := filepath.Join(home, "library", "networks", saved+".yaml")
	if _, err := os.Stat(savedFile); err != nil {
		t.Fatalf("--save did not land in the daemon library: %v", err)
	}

	const session = "pack-e2e"
	started := runNiac(t, niac, env, append([]string{
		"simulation", "start", "-i", loopback, "--config", saved + ".yaml",
		"--attachment", "cyberscope", "--attachment-mode", "direct", "--session", session,
	}, connect...)...)
	if !strings.Contains(started, `"running": true`) {
		t.Fatalf("simulation start did not report running:\n%s", started)
	}

	client, err := cliclient.New(cliclient.Config{BaseURL: api, CertPath: filepath.Join(certDir, "server.crt")})
	if err != nil {
		t.Fatalf("api client: %v", err)
	}
	ctx, cancel := context.WithTimeout(t.Context(), 20*time.Second)
	defer cancel()

	sessions, err := client.Sessions(ctx)
	if err != nil {
		t.Fatalf("list sessions: %v", err)
	}
	if len(sessions) != 1 || sessions[0].SessionID != session || !sessions[0].Running || sessions[0].DeviceCount == 0 {
		t.Fatalf("daemon sessions = %+v, want %q running with devices", sessions, session)
	}
	state, err := client.Runtime(ctx)
	if err != nil {
		t.Fatalf("read runtime: %v", err)
	}
	if !state.Running || state.Interface != loopback {
		t.Fatalf("runtime = %+v, want running on %s", state, loopback)
	}

	runNiac(t, niac, env, append([]string{"simulation", "stop", session}, connect...)...)
	sessions, err = client.Sessions(ctx)
	if err != nil {
		t.Fatalf("list sessions after stop: %v", err)
	}
	for _, s := range sessions {
		if s.Running {
			t.Fatalf("session %q still running after stop", s.SessionID)
		}
	}
}

// startRealDaemon launches `niac daemon` on a free loopback port with the
// interface approved by flag, waits until it accepts connections, and stops it
// when the test ends. It returns the API base URL.
func startRealDaemon(t *testing.T, niac string, env []string, certDir, iface string) string {
	t.Helper()

	probe, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("reserve a port: %v", err)
	}
	addr := probe.Addr().String()
	if err = probe.Close(); err != nil {
		t.Fatalf("release the port: %v", err)
	}

	var logs bytes.Buffer
	daemon := exec.Command(niac, "daemon", "--listen", addr, "--cert-dir", certDir,
		"--attachment-policy", iface+"=direct", "--storage", "disabled")
	daemon.Env = env
	daemon.Stdout, daemon.Stderr = &logs, &logs
	if err = daemon.Start(); err != nil {
		t.Fatalf("start daemon: %v", err)
	}
	exited := make(chan struct{})
	go func() { _ = daemon.Wait(); close(exited) }()
	t.Cleanup(func() {
		_ = daemon.Process.Signal(os.Interrupt)
		select {
		case <-exited:
		case <-time.After(10 * time.Second):
			_ = daemon.Process.Kill()
			<-exited
		}
		if t.Failed() {
			t.Logf("daemon log:\n%s", logs.String())
		}
	})

	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		select {
		case <-exited:
			t.Fatalf("daemon exited during start:\n%s", logs.String())
		default:
		}
		if conn, dialErr := net.DialTimeout("tcp", addr, time.Second); dialErr == nil {
			_ = conn.Close()
			return "https://" + addr
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("daemon never listened on %s:\n%s", addr, logs.String())

	return ""
}

// runNiac runs one CLI command and returns its stdout, failing the test with
// both streams when it exits non-zero.
func runNiac(t *testing.T, niac string, env []string, args ...string) string {
	t.Helper()

	var stdout, stderr bytes.Buffer
	cmd := exec.Command(niac, args...)
	cmd.Env = env
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("niac %s: %v\n%s", strings.Join(args, " "), err, stdout.String()+stderr.String())
	}

	return stdout.String()
}
