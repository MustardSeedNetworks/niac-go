package daemon

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/krisarmstrong/niac-go/internal/api"
)

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
		Interface:  "lo0", // Loopback interface should exist
		ConfigData: configData,
	}

	err = daemon.StartSimulation(req)
	if err != nil {
		t.Fatalf("Failed to start simulation: %v", err)
	}

	// Verify simulation is running
	status := daemon.GetStatus()
	if status.Running != true {
		t.Error("Simulation should be running")
	}

	if status.Interface != "lo0" {
		t.Errorf("Expected interface lo0, got: %s", status.Interface)
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

	err = daemon.StartSimulation(req)
	if err == nil {
		t.Error("Expected error for nonexistent interface")
	}

	if !strings.Contains(err.Error(), "does not exist") {
		t.Errorf("Expected 'does not exist' error, got: %v", err)
	}

	// Test 2: Invalid config
	req = api.SimulationRequest{
		Interface:  "lo0",
		ConfigData: "invalid: yaml: syntax: [[[",
	}

	err = daemon.StartSimulation(req)
	if err == nil {
		t.Error("Expected error for invalid config")
	}

	// Test 3: Missing config
	req = api.SimulationRequest{
		Interface: "lo0",
		// No ConfigData or ConfigPath
	}

	err = daemon.StartSimulation(req)
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
		Interface:  "lo0",
		ConfigData: largeConfig,
	}

	err = daemon.StartSimulation(req)
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
			Interface:  "lo0",
			ConfigData: configData,
		}

		startErr := daemon.StartSimulation(req)
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
