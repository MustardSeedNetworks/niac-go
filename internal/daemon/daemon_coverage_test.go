package daemon

import (
	"context"
	"errors"
	"net"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/MustardSeedNetworks/niac-go/internal/api"
	"github.com/MustardSeedNetworks/niac-go/internal/config"
)

// createMinimalConfig creates a minimal config with one device for testing.
func createMinimalConfig(t *testing.T) *config.Config {
	t.Helper()

	return &config.Config{
		Devices: []config.Device{
			{
				Name:        "test-device",
				Type:        "router",
				MACAddress:  net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, 0x55},
				IPAddresses: []net.IP{net.ParseIP("192.168.1.1")},
			},
		},
	}
}

// TestExpandPath_TildeSlash tests expandPath with ~/subdir.
func TestExpandPath_TildeSlash(t *testing.T) {
	result := expandPath("~/data")
	if strings.Contains(result, "~") {
		t.Errorf("Tilde not expanded: %s", result)
	}

	if !filepath.IsAbs(result) {
		t.Errorf("Expected absolute path, got: %s", result)
	}
}

// TestExpandPath_NoTilde tests expandPath with absolute and relative paths.
func TestExpandPath_NoTilde(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"/tmp/data", "/tmp/data"},
		{"relative/path", "relative/path"},
		{"/a/b/../c", "/a/c"},
	}

	for _, tt := range tests {
		got := expandPath(tt.input)
		if got != tt.want {
			t.Errorf("expandPath(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

// TestNewDaemon_PathTraversal verifies that storage paths containing '..'
// are rejected before [filepath.Clean] strips the traversal components.
func TestNewDaemon_PathTraversal(t *testing.T) {
	cfg := Config{
		ListenAddr:  "127.0.0.1:0",
		StoragePath: "/tmp/../tmp/niac-test-path-traversal.db",
		Version:     "test",
	}

	_, err := NewDaemon(cfg)
	if err == nil {
		t.Fatal("NewDaemon should reject storage path containing '..'")
	}
	if !strings.Contains(err.Error(), "..") {
		t.Errorf("error should mention '..' traversal, got: %v", err)
	}
}

// TestNewDaemon_ValidStoragePath tests NewDaemon with a valid storage path.
func TestNewDaemon_ValidStoragePath(t *testing.T) {
	tmpDir := t.TempDir()
	storagePath := filepath.Join(tmpDir, "test.db")

	cfg := Config{
		ListenAddr:  "127.0.0.1:0",
		StoragePath: storagePath,
		Version:     "test",
	}

	daemon, err := NewDaemon(cfg)
	if err != nil {
		t.Fatalf("NewDaemon failed: %v", err)
	}

	if daemon.storage == nil {
		t.Error("Expected storage to be initialized")
	}

	// Clean up
	if daemon.storage != nil {
		_ = daemon.storage.Close()
	}
}

// TestNewDaemon_InvalidStoragePath tests NewDaemon with an unreachable path.
func TestNewDaemon_InvalidStoragePath(t *testing.T) {
	cfg := Config{
		ListenAddr:  "127.0.0.1:0",
		StoragePath: "/nonexistent/dir/that/should/fail/test.db",
		Version:     "test",
	}

	_, err := NewDaemon(cfg)
	if err == nil {
		t.Error("Expected error for invalid storage path")
	}
}

// TestGetStatus_WithSimulation tests GetStatus when a simulation object is set.
func TestGetStatus_WithSimulation(t *testing.T) {
	cfg := Config{
		ListenAddr:  "127.0.0.1:0",
		StoragePath: "disabled",
		Version:     "test",
	}

	daemon, err := NewDaemon(cfg)
	if err != nil {
		t.Fatalf("NewDaemon failed: %v", err)
	}

	// Manually inject a simulation to test GetStatus branches
	startTime := time.Now().Add(-10 * time.Second)
	daemon.simulation = &Simulation{
		Interface:  "eth0",
		ConfigPath: "/etc/niac/test.yaml",
		ConfigName: "test.yaml",
		StartedAt:  startTime,
		cfg:        nil, // nil cfg path
	}

	status := daemon.GetStatus()

	if !status.Running {
		t.Error("Expected Running to be true")
	}

	if status.Interface != "eth0" {
		t.Errorf("Expected interface 'eth0', got %q", status.Interface)
	}

	if status.ConfigPath != "/etc/niac/test.yaml" {
		t.Errorf("Expected config path, got %q", status.ConfigPath)
	}

	if status.ConfigName != "test.yaml" {
		t.Errorf("Expected config name 'test.yaml', got %q", status.ConfigName)
	}

	if status.UptimeSeconds < 9.0 {
		t.Errorf("Expected uptime >= 9s, got %f", status.UptimeSeconds)
	}

	// DeviceCount should be 0 when cfg is nil
	if status.DeviceCount != 0 {
		t.Errorf("Expected DeviceCount=0 with nil cfg, got %d", status.DeviceCount)
	}
}

// TestGetStatus_WithSimulationAndConfig tests GetStatus with a non-nil config.
func TestGetStatus_WithSimulationAndConfig(t *testing.T) {
	cfg := Config{
		ListenAddr:  "127.0.0.1:0",
		StoragePath: "disabled",
		Version:     "test",
	}

	daemon, err := NewDaemon(cfg)
	if err != nil {
		t.Fatalf("NewDaemon failed: %v", err)
	}

	// Use a config import to avoid circular deps; simulate a config with devices
	// We just need cfg.Devices to be non-nil
	configPkg := createMinimalConfig(t)

	daemon.simulation = &Simulation{
		Interface:  "lo0",
		ConfigPath: "inline",
		ConfigName: "inline",
		StartedAt:  time.Now(),
		cfg:        configPkg,
	}

	status := daemon.GetStatus()

	if !status.Running {
		t.Error("Expected Running=true")
	}

	if status.DeviceCount != 1 {
		t.Errorf("Expected DeviceCount=1, got %d", status.DeviceCount)
	}
}

// TestStopSimulationLocked_NilFields tests stopSimulationLocked with partial simulation.
func TestStopSimulationLocked_NilFields(t *testing.T) {
	tmpDir := t.TempDir()
	storagePath := filepath.Join(tmpDir, "test.db")

	cfg := Config{
		ListenAddr:  "127.0.0.1:0",
		StoragePath: storagePath,
		Version:     "test",
	}

	daemon, err := NewDaemon(cfg)
	if err != nil {
		t.Fatalf("NewDaemon failed: %v", err)
	}

	err = daemon.Start()
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = daemon.Shutdown(ctx)
	}()

	// Inject a simulation with nil fields to test nil-safe branches in stopSimulationLocked
	daemon.mu.Lock()
	daemon.simulation = &Simulation{
		Interface:  "test",
		ConfigPath: "test",
		ConfigName: "test",
		StartedAt:  time.Now(),
		engine:     nil,
		stack:      nil,
		cfg:        nil,
		replay:     nil,
		cancel:     nil,
	}
	daemon.mu.Unlock()

	err = daemon.StopSimulation()
	if err != nil {
		t.Errorf("StopSimulation with nil fields should succeed, got: %v", err)
	}

	// Should now be nil
	status := daemon.GetStatus()
	if status.Running {
		t.Error("Expected simulation to be stopped")
	}
}

// TestStartSimulation_NonExistentConfigPath tests StartSimulation with a config file that doesn't exist.
func TestStartSimulation_NonExistentConfigPath(t *testing.T) {
	cfg := Config{
		ListenAddr:  "127.0.0.1:0",
		StoragePath: "disabled",
		Version:     "test",
	}

	daemon, err := NewDaemon(cfg)
	if err != nil {
		t.Fatalf("NewDaemon failed: %v", err)
	}

	err = daemon.Start()
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = daemon.Shutdown(ctx)
	}()

	req := api.SimulationRequest{
		Interface:  loopbackInterface(),
		ConfigPath: "/nonexistent/path/config.yaml",
	}

	err = daemon.StartSimulation(req, fullSimulationEntitlements())
	if err == nil {
		t.Error("Expected error for nonexistent config file")
	}

	if !strings.Contains(err.Error(), "load configuration") {
		t.Errorf("Expected 'load configuration' error, got: %v", err)
	}
}

// TestStartSimulation_ConfigDataExceedsSize tests the size limit check.
func TestStartSimulation_ConfigDataExceedsSize(t *testing.T) {
	cfg := Config{
		ListenAddr:  "127.0.0.1:0",
		StoragePath: "disabled",
		Version:     "test",
	}

	daemon, err := NewDaemon(cfg)
	if err != nil {
		t.Fatalf("NewDaemon failed: %v", err)
	}

	err = daemon.Start()
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = daemon.Shutdown(ctx)
	}()

	// Create config data that exceeds 10MB
	req := api.SimulationRequest{
		Interface:  loopbackInterface(),
		ConfigData: strings.Repeat("a", 11*1024*1024),
	}

	err = daemon.StartSimulation(req, fullSimulationEntitlements())
	if err == nil {
		t.Error("Expected error for oversized config data")
	}

	if !errors.Is(err, ErrConfigDataExceedsMaxSize) {
		t.Errorf("Expected ErrConfigDataExceedsMaxSize, got: %v", err)
	}
}

// TestStartSimulation_MissingConfigAndPath tests the default case.
func TestStartSimulation_MissingConfigAndPath(t *testing.T) {
	cfg := Config{
		ListenAddr:  "127.0.0.1:0",
		StoragePath: "disabled",
		Version:     "test",
	}

	daemon, err := NewDaemon(cfg)
	if err != nil {
		t.Fatalf("NewDaemon failed: %v", err)
	}

	err = daemon.Start()
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = daemon.Shutdown(ctx)
	}()

	req := api.SimulationRequest{
		Interface: loopbackInterface(),
		// Both ConfigData and ConfigPath empty
	}

	err = daemon.StartSimulation(req, fullSimulationEntitlements())
	if err == nil {
		t.Error("Expected error when both config_path and config_data are empty")
	}

	if !errors.Is(err, ErrConfigPathOrDataRequired) {
		t.Errorf("Expected ErrConfigPathOrDataRequired, got: %v", err)
	}
}

// TestStartSimulation_InvalidInterface tests starting with a non-existent interface.
func TestStartSimulation_InvalidInterface(t *testing.T) {
	cfg := Config{
		ListenAddr:  "127.0.0.1:0",
		StoragePath: "disabled",
		Version:     "test",
	}

	daemon, err := NewDaemon(cfg)
	if err != nil {
		t.Fatalf("NewDaemon failed: %v", err)
	}

	err = daemon.Start()
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = daemon.Shutdown(ctx)
	}()

	req := api.SimulationRequest{
		Interface:  "nonexistent-iface-xyz-123",
		ConfigData: "devices: []",
	}

	err = daemon.StartSimulation(req, fullSimulationEntitlements())
	if err == nil {
		t.Error("Expected error for nonexistent interface")
	}

	if !errors.Is(err, ErrInterfaceNotExist) {
		t.Errorf("Expected ErrInterfaceNotExist, got: %v", err)
	}
}

// TestStartSimulation_InvalidConfigData tests starting with malformed YAML.
func TestStartSimulation_InvalidConfigData(t *testing.T) {
	cfg := Config{
		ListenAddr:  "127.0.0.1:0",
		StoragePath: "disabled",
		Version:     "test",
	}

	daemon, err := NewDaemon(cfg)
	if err != nil {
		t.Fatalf("NewDaemon failed: %v", err)
	}

	err = daemon.Start()
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = daemon.Shutdown(ctx)
	}()

	req := api.SimulationRequest{
		Interface:  loopbackInterface(),
		ConfigData: "{{{{invalid yaml",
	}

	err = daemon.StartSimulation(req, fullSimulationEntitlements())
	if err == nil {
		t.Error("Expected error for invalid YAML config")
	}
}

// TestShutdown_NoAPIServer tests Shutdown when apiServer is nil.
func TestShutdown_NoAPIServer(t *testing.T) {
	cfg := Config{
		ListenAddr:  "127.0.0.1:0",
		StoragePath: "disabled",
		Version:     "test",
	}

	daemon, err := NewDaemon(cfg)
	if err != nil {
		t.Fatalf("NewDaemon failed: %v", err)
	}

	// Don't call Start, so apiServer stays nil
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = daemon.Shutdown(ctx)
	if err != nil {
		t.Errorf("Shutdown with nil apiServer should succeed, got: %v", err)
	}
}

// TestShutdown_WithStorage tests Shutdown properly closes storage.
func TestShutdown_WithStorage(t *testing.T) {
	tmpDir := t.TempDir()
	storagePath := filepath.Join(tmpDir, "test.db")

	cfg := Config{
		ListenAddr:  "127.0.0.1:0",
		StoragePath: storagePath,
		Version:     "test",
	}

	daemon, err := NewDaemon(cfg)
	if err != nil {
		t.Fatalf("NewDaemon failed: %v", err)
	}

	if daemon.storage == nil {
		t.Fatal("Storage should be initialized")
	}

	err = daemon.Start()
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = daemon.Shutdown(ctx)
	if err != nil {
		t.Errorf("Shutdown failed: %v", err)
	}
}

// TestReplayController_StopIdempotent tests calling Stop multiple times.
func TestReplayController_StopIdempotent(t *testing.T) {
	rc := newReplayController(nil, 0)

	for range 3 {
		state, err := rc.Stop()
		if err != nil {
			t.Errorf("Stop returned error: %v", err)
		}

		if state.Running {
			t.Error("State should not be running after Stop")
		}
	}
}

// TestReplayController_NewWithDebugLevel exercises the constructor at every
// supported debug level. We can't observe the level directly any more (it's
// internal/replay-private), so the assertion is the looser-but-still-useful
// "constructor doesn't panic and Status() returns a non-running default".
func TestReplayController_NewWithDebugLevel(t *testing.T) {
	for _, level := range []int{0, 1, 2, 3} {
		rc := newReplayController(nil, level)
		if rc == nil {
			t.Errorf("newReplayController(nil, %d) returned nil", level)
			continue
		}
		if rc.Status().Running {
			t.Errorf("level %d: fresh controller should not report Running", level)
		}
	}
}

// TestSentinelErrors tests daemon sentinel error messages.
func TestSentinelErrors(t *testing.T) {
	tests := []struct {
		err  error
		want string
	}{
		{ErrInterfaceNotExist, "interface does not exist"},
		{ErrConfigDataExceedsMaxSize, "config data exceeds maximum size"},
		{ErrConfigPathOrDataRequired, "either config_path, config_data, or template_name must be provided"},
		{ErrNoSimulationRunning, "no simulation running"},
	}

	for _, tt := range tests {
		if tt.err.Error() != tt.want {
			t.Errorf("Error %v: got %q, want %q", tt.err, tt.err.Error(), tt.want)
		}
	}
}
