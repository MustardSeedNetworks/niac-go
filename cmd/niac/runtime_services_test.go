package main

import (
	"os"
	"testing"

	"github.com/krisarmstrong/niac-go/internal/api"
)

func TestResolveAPIToken(t *testing.T) {
	t.Run("env var takes precedence", func(t *testing.T) {
		t.Setenv("NIAC_API_TOKEN", "env-token")
		result := resolveAPIToken("cli-token")
		if result != "env-token" {
			t.Errorf("resolveAPIToken() = %q, want %q", result, "env-token")
		}
	})

	t.Run("cli fallback when no env", func(t *testing.T) {
		t.Setenv("NIAC_API_TOKEN", "")
		result := resolveAPIToken("cli-token")
		if result != "cli-token" {
			t.Errorf("resolveAPIToken() = %q, want %q", result, "cli-token")
		}
	})

	t.Run("empty when neither set", func(t *testing.T) {
		t.Setenv("NIAC_API_TOKEN", "")
		result := resolveAPIToken("")
		if result != "" {
			t.Errorf("resolveAPIToken() = %q, want empty", result)
		}
	})
}

func TestReplayControllerStatus(t *testing.T) {
	rc := newReplayController(nil, 1)
	state := rc.Status()
	if state.Running {
		t.Error("Expected new controller to not be running")
	}
	if state.File != "" {
		t.Errorf("Expected empty file, got %q", state.File)
	}
}

func TestReplayControllerStartNoEngine(t *testing.T) {
	rc := newReplayController(nil, 1)
	req := api.ReplayRequest{File: "test.pcap"}
	_, err := rc.Start(req)
	if err == nil {
		t.Error("Expected error when engine is nil")
	}
}

func TestReplayControllerStartEmptyFile(t *testing.T) {
	rc := newReplayController(nil, 1)
	req := api.ReplayRequest{File: ""}
	_, err := rc.Start(req)
	if err == nil {
		t.Error("Expected error for empty file path")
	}
}

func TestReplayControllerStop(t *testing.T) {
	rc := newReplayController(nil, 1)
	state, err := rc.Stop()
	if err != nil {
		t.Errorf("Stop() error = %v", err)
	}
	if state.Running {
		t.Error("Expected Running=false after stop")
	}
}

func TestReplayControllerCleanupTempFile(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := tmpDir + "/test.pcap"
	if err := os.WriteFile(tmpFile, []byte("test"), 0o644); err != nil {
		t.Fatal(err)
	}

	rc := newReplayController(nil, 1)
	rc.cleanup = tmpFile
	rc.cleanupTempFile()

	if rc.cleanup != "" {
		t.Error("Expected cleanup to be cleared")
	}
	if _, err := os.Stat(tmpFile); !os.IsNotExist(err) {
		t.Error("Expected temp file to be removed")
	}
}

func TestReplayControllerCleanupNoFile(_ *testing.T) {
	rc := newReplayController(nil, 1)
	rc.cleanup = ""
	// Should not panic
	rc.cleanupTempFile()
}

func TestRuntimeServicesApplyConfigNil(t *testing.T) {
	// nil receiver
	var rs *runtimeServices
	err := rs.applyConfig(nil)
	if err == nil {
		t.Error("Expected error for nil runtime services")
	}

	// nil config
	rs2 := &runtimeServices{}
	err = rs2.applyConfig(nil)
	if err == nil {
		t.Error("Expected error for nil config")
	}
}
