package api

import (
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/logging"
	"github.com/MustardSeedNetworks/niac-go/internal/protocols"
)

func TestCurrentConfig(t *testing.T) {
	cfg := mustLoadConfig(t, baseConfigYAML)
	server := &Server{
		cfg: ServerConfig{
			Config: cfg,
		},
		logger: slog.Default(),
	}

	got := server.currentConfig()
	if got == nil {
		t.Fatal("currentConfig() returned nil")
	}
	if len(got.Devices) != 1 {
		t.Errorf("device count = %d, want 1", len(got.Devices))
	}
}

func TestCurrentConfigNil(t *testing.T) {
	server := &Server{
		cfg:    ServerConfig{},
		logger: slog.Default(),
	}

	got := server.currentConfig()
	if got != nil {
		t.Error("currentConfig() should return nil when no config loaded")
	}
}

func TestConfigPath(t *testing.T) {
	server := &Server{
		cfg: ServerConfig{
			ConfigPath: "/tmp/test/config.yaml",
		},
		logger: slog.Default(),
	}

	got := server.configPath()
	if got != "/tmp/test/config.yaml" {
		t.Errorf("configPath() = %q, want %q", got, "/tmp/test/config.yaml")
	}
}

func TestConfigPathEmpty(t *testing.T) {
	server := &Server{
		cfg:    ServerConfig{},
		logger: slog.Default(),
	}

	got := server.configPath()
	if got != "" {
		t.Errorf("configPath() = %q, want empty", got)
	}
}

func TestCurrentInterface(t *testing.T) {
	server := &Server{
		cfg: ServerConfig{
			Interface: "eth0",
		},
		logger: slog.Default(),
	}

	got := server.currentInterface()
	if got != "eth0" {
		t.Errorf("currentInterface() = %q, want %q", got, "eth0")
	}
}

func TestCurrentStack(t *testing.T) {
	cfg := mustLoadConfig(t, baseConfigYAML)
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))

	server := &Server{
		cfg: ServerConfig{
			Stack: stack,
		},
		logger: slog.Default(),
	}

	got := server.currentStack()
	if got == nil {
		t.Fatal("currentStack() returned nil")
	}
}

func TestCurrentStackNil(t *testing.T) {
	server := &Server{
		cfg:    ServerConfig{},
		logger: slog.Default(),
	}

	got := server.currentStack()
	if got != nil {
		t.Error("currentStack() should return nil when no stack")
	}
}

func TestCurrentTopology(t *testing.T) {
	cfg := mustLoadConfig(t, baseConfigYAML)
	topo := BuildTopology(cfg)

	server := &Server{
		cfg: ServerConfig{
			Topology: topo,
		},
		logger: slog.Default(),
	}

	got := server.currentTopology()
	if len(got.Nodes) != len(topo.Nodes) {
		t.Errorf("topology nodes = %d, want %d", len(got.Nodes), len(topo.Nodes))
	}
}

func TestReplaceConfig(t *testing.T) {
	server := &Server{
		cfg: ServerConfig{
			Config: mustLoadConfig(t, baseConfigYAML),
		},
		logger: slog.Default(),
	}

	newCfg := mustLoadConfig(t, updatedConfigYAML)
	server.replaceConfig(newCfg)

	got := server.currentConfig()
	if got == nil {
		t.Fatal("config should not be nil after replace")
	}
	if len(got.Devices) == 0 {
		t.Fatal("expected at least one device after replace")
	}
	if got.Devices[0].Name != "core2" {
		t.Errorf("device name = %q, want %q", got.Devices[0].Name, "core2")
	}
}

func TestReadConfigDocument(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	err := os.WriteFile(configPath, []byte(baseConfigYAML), 0o600)
	if err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg := mustLoadConfig(t, baseConfigYAML)
	server := &Server{
		cfg: ServerConfig{
			Config:     cfg,
			ConfigPath: configPath,
		},
		logger: slog.Default(),
	}

	doc, status, err := server.readConfigDocument()
	if err != nil {
		t.Fatalf("readConfigDocument(): %v", err)
	}
	if status != 200 {
		t.Errorf("status = %d, want 200", status)
	}
	if doc.Path != configPath {
		t.Errorf("doc.Path = %q, want %q", doc.Path, configPath)
	}
	if doc.DeviceCount != 1 {
		t.Errorf("doc.DeviceCount = %d, want 1", doc.DeviceCount)
	}
	if doc.Content == "" {
		t.Error("doc.Content should not be empty")
	}
	if doc.Filename != "config.yaml" {
		t.Errorf("doc.Filename = %q, want %q", doc.Filename, "config.yaml")
	}
}

func TestReadConfigDocumentNoPath(t *testing.T) {
	server := &Server{
		cfg: ServerConfig{
			Config:     mustLoadConfig(t, baseConfigYAML),
			ConfigPath: "",
		},
		logger: slog.Default(),
	}

	_, status, err := server.readConfigDocument()
	if err == nil {
		t.Error("readConfigDocument() should fail when no path")
	}
	if status != 400 {
		t.Errorf("status = %d, want 400", status)
	}
}

func TestReadConfigDocumentMissingFile(t *testing.T) {
	server := &Server{
		cfg: ServerConfig{
			Config:     mustLoadConfig(t, baseConfigYAML),
			ConfigPath: "/nonexistent/path/config.yaml",
		},
		logger: slog.Default(),
	}

	_, status, err := server.readConfigDocument()
	if err == nil {
		t.Error("readConfigDocument() should fail for missing file")
	}
	if status != 404 {
		t.Errorf("status = %d, want 404", status)
	}
}

func TestWriteConfigFile(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	server := &Server{
		cfg: ServerConfig{
			ConfigPath: configPath,
		},
		logger: slog.Default(),
	}

	err := server.writeConfigFile("devices: []\n")
	if err != nil {
		t.Fatalf("writeConfigFile(): %v", err)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read written file: %v", err)
	}
	if string(data) != "devices: []\n" {
		t.Errorf("written content = %q, want %q", string(data), "devices: []\n")
	}
}

func TestUpdateSimulation(t *testing.T) {
	server := &Server{
		cfg:    ServerConfig{},
		logger: slog.Default(),
	}

	cfg := &config.Config{
		Devices: []config.Device{{Name: "r1", Type: "router"}},
	}
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))

	server.UpdateSimulation(stack, cfg, "/tmp/config.yaml", "eth0", nil)

	if server.currentConfig() == nil {
		t.Error("config should not be nil after UpdateSimulation")
	}
	if server.currentStack() == nil {
		t.Error("stack should not be nil after UpdateSimulation")
	}
	if server.configPath() != "/tmp/config.yaml" {
		t.Errorf("configPath = %q, want %q", server.configPath(), "/tmp/config.yaml")
	}
	if server.currentInterface() != "eth0" {
		t.Errorf("interface = %q, want %q", server.currentInterface(), "eth0")
	}
}

func TestClearSimulation(t *testing.T) {
	cfg := mustLoadConfig(t, baseConfigYAML)
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))

	server := &Server{
		cfg: ServerConfig{
			Stack:  stack,
			Config: cfg,
		},
		logger: slog.Default(),
	}

	server.ClearSimulation()

	if server.currentConfig() != nil {
		t.Error("config should be nil after ClearSimulation")
	}
	if server.currentStack() != nil {
		t.Error("stack should be nil after ClearSimulation")
	}
}
