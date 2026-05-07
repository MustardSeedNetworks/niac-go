package main

import (
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

// Cleanup-temp-file behaviour now lives in internal/replay (see PR #494) and
// is verified there at the package level. The legacy CLI's wrapper just calls
// replay.New so there's nothing CLI-specific left to assert; we keep
// TestReplayControllerStop above to confirm the wrapper still wires Stop()
// through correctly.

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
