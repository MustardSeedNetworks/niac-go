package daemon

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/MustardSeedNetworks/niac-go/internal/api"
)

func fullSimulationEntitlements() api.SimulationEntitlements {
	return api.SimulationEntitlements{RoutedLabs: true, UnlimitedDevices: true}
}

// loopbackInterface returns the loopback interface name for the current OS.
func loopbackInterface() string {
	if runtime.GOOS == "darwin" {
		return "lo0"
	}
	return "lo" // Linux
}

// TestDaemon_StartupShutdown verifies daemon can start and shutdown cleanly.
func TestDaemon_StartupShutdown(t *testing.T) {
	// Create temporary storage
	tmpDir := t.TempDir()
	storagePath := filepath.Join(tmpDir, "test.db")

	cfg := Config{
		ListenAddr:  "127.0.0.1:0", // Random port
		Token:       "test-token-123",
		StoragePath: storagePath,
		Version:     "test",
	}

	daemon, err := NewDaemon(cfg)
	if err != nil {
		t.Fatalf("Failed to create daemon: %v", err)
	}

	// Start daemon
	err = daemon.Start()
	if err != nil {
		t.Fatalf("Failed to start daemon: %v", err)
	}

	// Verify API server is running
	if daemon.apiServer == nil {
		t.Fatal("API server not initialized")
	}

	// Verify storage is open
	if daemon.storage == nil {
		t.Fatal("Storage not initialized")
	}

	// Shutdown daemon
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = daemon.Shutdown(ctx)
	if err != nil {
		t.Fatalf("Failed to shutdown daemon: %v", err)
	}
}

// TestDaemon_SimulationLifecycle verifies simulation can be started and stopped.
func TestDaemon_SimulationLifecycle(t *testing.T) {
	if os.Getenv("CI") == "true" || os.Geteuid() != 0 {
		t.Skip("Skipping simulation test (requires root privileges for packet capture)")
	}

	tmpDir := t.TempDir()
	storagePath := filepath.Join(tmpDir, "test.db")

	cfg := Config{
		ListenAddr:  "127.0.0.1:0",
		Token:       "test-token-123",
		StoragePath: storagePath,
		Version:     "test",
	}

	daemon, err := NewDaemon(cfg)
	if err != nil {
		t.Fatalf("Failed to create daemon: %v", err)
	}

	err = daemon.Start()
	if err != nil {
		t.Fatalf("Failed to start daemon: %v", err)
	}

	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		_ = daemon.Shutdown(ctx)
	}()

	// Create inline config for simulation
	configData := `
devices:
  - name: test-router
    ip: 192.168.1.1
    mac: 00:11:22:33:44:55
    type: router
    protocols:
      arp:
        enabled: true
      icmp:
        enabled: true
`

	// Start simulation
	req := api.SimulationRequest{
		Interface:  loopbackInterface(), // Loopback interface should exist
		ConfigData: configData,
	}

	err = daemon.StartSimulation(req, fullSimulationEntitlements())
	if err != nil {
		t.Fatalf("Failed to start simulation: %v", err)
	}

	// Verify simulation is running
	status := daemon.GetStatus()
	if status.Running != true {
		t.Error("Simulation should be running")
	}

	if status.Interface != loopbackInterface() {
		t.Errorf("Expected interface %s, got: %s", loopbackInterface(), status.Interface)
	}

	// Stop simulation
	err = daemon.StopSimulation()
	if err != nil {
		t.Fatalf("Failed to stop simulation: %v", err)
	}

	// Verify simulation stopped
	status = daemon.GetStatus()
	if status.Running != false {
		t.Error("Simulation should be stopped")
	}
}

// TestDaemon_ErrorRecovery verifies daemon handles errors gracefully.
func TestDaemon_ErrorRecovery(t *testing.T) {
	tmpDir := t.TempDir()
	storagePath := filepath.Join(tmpDir, "test.db")

	cfg := Config{
		ListenAddr:  "127.0.0.1:0",
		Token:       "test-token-123",
		StoragePath: storagePath,
		Version:     "test",
	}

	daemon, err := NewDaemon(cfg)
	if err != nil {
		t.Fatalf("Failed to create daemon: %v", err)
	}

	err = daemon.Start()
	if err != nil {
		t.Fatalf("Failed to start daemon: %v", err)
	}

	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		_ = daemon.Shutdown(ctx)
	}()

	// Test 1: Invalid interface
	req := api.SimulationRequest{
		Interface:  "nonexistent-interface-xyz",
		ConfigData: "devices: []",
	}

	err = daemon.StartSimulation(req, fullSimulationEntitlements())
	if err == nil {
		t.Error("Expected error for nonexistent interface")
	}

	if !strings.Contains(err.Error(), "does not exist") {
		t.Errorf("Expected 'does not exist' error, got: %v", err)
	}

	// Test 2: Invalid config
	req = api.SimulationRequest{
		Interface:  loopbackInterface(),
		ConfigData: "invalid: yaml: syntax: [[[",
	}

	err = daemon.StartSimulation(req, fullSimulationEntitlements())
	if err == nil {
		t.Error("Expected error for invalid config")
	}

	// Test 3: Missing config
	req = api.SimulationRequest{
		Interface: loopbackInterface(),
		// No ConfigData or ConfigPath
	}

	err = daemon.StartSimulation(req, fullSimulationEntitlements())
	if err == nil {
		t.Error("Expected error for missing config")
	}

	if !strings.Contains(err.Error(), "must be provided") {
		t.Errorf("Expected 'must be provided' error, got: %v", err)
	}
}

// TestDaemon_ResourceCleanup verifies resources are cleaned up properly.
func TestDaemon_ResourceCleanup(t *testing.T) {
	tmpDir := t.TempDir()
	storagePath := filepath.Join(tmpDir, "test.db")

	cfg := Config{
		ListenAddr:  "127.0.0.1:0",
		Token:       "test-token-123",
		StoragePath: storagePath,
		Version:     "test",
	}

	daemon, err := NewDaemon(cfg)
	if err != nil {
		t.Fatalf("Failed to create daemon: %v", err)
	}

	err = daemon.Start()
	if err != nil {
		t.Fatalf("Failed to start daemon: %v", err)
	}

	// Verify storage file exists
	_, err = os.Stat(storagePath)
	if os.IsNotExist(err) {
		t.Errorf("Storage file should exist: %s", storagePath)
	}

	// Shutdown daemon
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = daemon.Shutdown(ctx)
	if err != nil {
		t.Fatalf("Failed to shutdown daemon: %v", err)
	}

	// Note: Storage file should still exist after shutdown (persistent data)
	// but it should be closed properly
	_, err = os.Stat(storagePath)
	if os.IsNotExist(err) {
		t.Errorf("Storage file should still exist after shutdown: %s", storagePath)
	}
}

// TestDaemon_ConfigSizeValidation verifies config size limits (SECURITY FIX #2.8.1).
func TestDaemon_ConfigSizeValidation(t *testing.T) {
	tmpDir := t.TempDir()
	storagePath := filepath.Join(tmpDir, "test.db")

	cfg := Config{
		ListenAddr:  "127.0.0.1:0",
		Token:       "test-token-123",
		StoragePath: storagePath,
		Version:     "test",
	}

	daemon, err := NewDaemon(cfg)
	if err != nil {
		t.Fatalf("Failed to create daemon: %v", err)
	}

	err = daemon.Start()
	if err != nil {
		t.Fatalf("Failed to start daemon: %v", err)
	}

	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		_ = daemon.Shutdown(ctx)
	}()

	// Test config that exceeds 10MB limit
	largeConfig := strings.Repeat("x", 11*1024*1024) // 11MB

	req := api.SimulationRequest{
		Interface:  loopbackInterface(),
		ConfigData: largeConfig,
	}

	err = daemon.StartSimulation(req, fullSimulationEntitlements())
	if err == nil {
		t.Error("Expected error for config exceeding size limit")
	}

	if !strings.Contains(err.Error(), "exceeds maximum size") {
		t.Errorf("Expected 'exceeds maximum size' error, got: %v", err)
	}
}

// TestDaemon_StorageDisabled verifies daemon works without storage.
func TestDaemon_StorageDisabled(t *testing.T) {
	cfg := Config{
		ListenAddr:  "127.0.0.1:0",
		Token:       "test-token-123",
		StoragePath: "disabled",
		Version:     "test",
	}

	daemon, err := NewDaemon(cfg)
	if err != nil {
		t.Fatalf("Failed to create daemon: %v", err)
	}

	// Verify storage is nil when disabled
	if daemon.storage != nil {
		t.Error("Storage should be nil when disabled")
	}

	err = daemon.Start()
	if err != nil {
		t.Fatalf("Failed to start daemon: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = daemon.Shutdown(ctx)
	if err != nil {
		t.Fatalf("Failed to shutdown daemon: %v", err)
	}
}

// TestExpandPath verifies path expansion handles tilde and relative paths.
func TestExpandPath(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("Cannot get user home directory")
	}

	tests := []struct {
		name string
		path string
		want string
	}{
		{
			name: "tilde path",
			path: "~/test/path",
			want: filepath.Join(home, "test/path"),
		},
		{
			name: "tilde only",
			path: "~",
			want: home,
		},
		{
			name: "absolute path unchanged",
			path: "/var/log/test.log",
			want: "/var/log/test.log",
		},
		{
			name: "relative path cleaned",
			path: "./test/../data/file.txt",
			want: "data/file.txt",
		},
		{
			name: "empty path",
			path: "",
			want: ".",
		},
		{
			name: "dot dot path cleaned",
			path: "/foo/bar/../baz",
			want: "/foo/baz",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := expandPath(tt.path)
			if got != tt.want {
				t.Errorf("expandPath(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}

// TestReplayController_Status verifies replay controller status method.
func TestReplayController_Status(t *testing.T) {
	rc := newReplayController(nil, 0)

	// Initial status should have Running = false
	status := rc.Status()
	if status.Running {
		t.Error("Initial replay state should not be running")
	}
}

// TestReplayController_Start_NoEngine verifies Start refuses to run without
// a capture engine (the daemon constructs a controller per-simulation, but a
// nil engine should still bail cleanly rather than panic).
func TestReplayController_Start_NoEngine(t *testing.T) {
	rc := newReplayController(nil, 0)

	req := api.ReplayRequest{
		File: "/tmp/test.pcap",
	}

	state, err := rc.Start(req)
	if err == nil {
		t.Fatal("expected error when engine is nil")
	}
	if state.Running {
		t.Errorf("state should not be running after a failed Start, got %+v", state)
	}
}

// TestReplayController_Start_EmptyFile verifies an empty file path is rejected.
func TestReplayController_Start_EmptyFile(t *testing.T) {
	rc := newReplayController(nil, 0)

	_, err := rc.Start(api.ReplayRequest{File: "   "})
	if err == nil {
		t.Fatal("expected error for empty file path")
	}
}

// TestReplayController_Stop verifies replay stop method.
func TestReplayController_Stop(t *testing.T) {
	// Stop on a freshly-constructed controller (no engine, no Start) is a
	// no-op and must never error — exercises the idempotence path that the
	// daemon's shutdown sequence relies on.
	rc := newReplayController(nil, 0)
	state, err := rc.Stop()
	if err != nil {
		t.Errorf("Stop returned error on fresh controller: %v", err)
	}
	if state.Running {
		t.Errorf("state.Running should be false after Stop on fresh controller, got %+v", state)
	}
}

// TestGetStatus_NoSimulation verifies GetStatus when no simulation is running.
func TestGetStatus_NoSimulation(t *testing.T) {
	cfg := Config{
		ListenAddr:  "127.0.0.1:0",
		StoragePath: "disabled",
		Version:     "test",
	}

	daemon, err := NewDaemon(cfg)
	if err != nil {
		t.Fatalf("Failed to create daemon: %v", err)
	}

	status := daemon.GetStatus()
	if status.Running {
		t.Error("Status should show not running")
	}
	if status.Interface != "" {
		t.Errorf("Interface should be empty, got: %s", status.Interface)
	}
	if status.DeviceCount != 0 {
		t.Errorf("DeviceCount should be 0, got: %d", status.DeviceCount)
	}
}

// TestStopSimulation_NoSimulation verifies StopSimulation error when none running.
func TestStopSimulation_NoSimulation(t *testing.T) {
	cfg := Config{
		ListenAddr:  "127.0.0.1:0",
		StoragePath: "disabled",
		Version:     "test",
	}

	daemon, err := NewDaemon(cfg)
	if err != nil {
		t.Fatalf("Failed to create daemon: %v", err)
	}

	err = daemon.StopSimulation()
	if err == nil {
		t.Error("Expected error when stopping non-existent simulation")
	}
	if !errors.Is(err, ErrNoSimulationRunning) {
		t.Errorf("Expected ErrNoSimulationRunning, got: %v", err)
	}
}

// TestNewDaemon_EmptyStoragePath verifies daemon with empty storage path.
func TestNewDaemon_EmptyStoragePath(t *testing.T) {
	cfg := Config{
		ListenAddr:  "127.0.0.1:0",
		StoragePath: "",
		Version:     "test",
	}

	daemon, err := NewDaemon(cfg)
	if err != nil {
		t.Fatalf("Failed to create daemon: %v", err)
	}

	if daemon.storage != nil {
		t.Error("Storage should be nil when path is empty")
	}
}

// TestDaemon_MultipleStartStop verifies daemon handles multiple start/stop cycles.
func TestDaemon_MultipleStartStop(t *testing.T) {
	if os.Getenv("CI") == "true" || os.Geteuid() != 0 {
		t.Skip("Skipping simulation test (requires root privileges for packet capture)")
	}

	tmpDir := t.TempDir()
	storagePath := filepath.Join(tmpDir, "test.db")

	cfg := Config{
		ListenAddr:  "127.0.0.1:0",
		Token:       "test-token-123",
		StoragePath: storagePath,
		Version:     "test",
	}

	daemon, err := NewDaemon(cfg)
	if err != nil {
		t.Fatalf("Failed to create daemon: %v", err)
	}

	err = daemon.Start()
	if err != nil {
		t.Fatalf("Failed to start daemon: %v", err)
	}

	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		_ = daemon.Shutdown(ctx)
	}()

	configData := `
devices:
  - name: test-device
    ip: 192.168.1.1
    mac: 00:11:22:33:44:55
    type: router
    protocols:
      arp:
        enabled: true
`

	// Start/stop simulation 3 times
	for i := range 3 {
		req := api.SimulationRequest{
			Interface:  loopbackInterface(),
			ConfigData: configData,
		}

		startErr := daemon.StartSimulation(req, fullSimulationEntitlements())
		if startErr != nil {
			t.Fatalf("Cycle %d: Failed to start simulation: %v", i, startErr)
		}

		status := daemon.GetStatus()
		if !status.Running {
			t.Errorf("Cycle %d: Simulation should be running", i)
		}

		stopErr := daemon.StopSimulation()
		if stopErr != nil {
			t.Fatalf("Cycle %d: Failed to stop simulation: %v", i, stopErr)
		}

		status = daemon.GetStatus()
		if status.Running {
			t.Errorf("Cycle %d: Simulation should be stopped", i)
		}
	}
}
