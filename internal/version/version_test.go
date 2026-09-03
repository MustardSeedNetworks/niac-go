package version_test

import (
	"runtime/debug"
	"slices"
	"strings"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/version"
)

func TestGetVersionNonEmpty(t *testing.T) {
	if version.GetVersion() == "" {
		t.Error("GetVersion returned empty string")
	}
}

func TestGetCommitNonEmpty(t *testing.T) {
	if version.GetCommit() == "" {
		t.Error("GetCommit returned empty string")
	}
}

func TestGetBuildTimeNonEmpty(t *testing.T) {
	if version.GetBuildTime() == "" {
		t.Error("GetBuildTime returned empty string")
	}
}

func TestGetUIBuildHashNonEmpty(t *testing.T) {
	if version.GetUIBuildHash() == "" {
		t.Error("GetUIBuildHash returned empty string (should default to 'unknown')")
	}
}

func TestInfoShape(t *testing.T) {
	info := version.Info()

	expectedKeys := []string{"version", "commit", "buildTime", "releaseTrain", "uiBuildHash"}
	if len(info) != len(expectedKeys) {
		t.Errorf("Info() returned %d keys, want %d", len(info), len(expectedKeys))
	}

	for _, key := range expectedKeys {
		v, ok := info[key]
		if !ok {
			t.Errorf("Info() missing key %q", key)
		}
		if v == "" {
			t.Errorf("Info()[%q] is empty", key)
		}
	}

	for key := range info {
		if !slices.Contains(expectedKeys, key) {
			t.Errorf("Info() has unexpected key %q", key)
		}
	}
}

func TestInfoReturnsFreshMap(t *testing.T) {
	a := version.Info()
	b := version.Info()
	a["version"] = "mutated"
	if b["version"] == "mutated" {
		t.Error("Info() returns shared map; each call must return a fresh copy")
	}
}

func TestInfoMatchesGetters(t *testing.T) {
	info := version.Info()
	if info["version"] != version.GetVersion() {
		t.Errorf("Info()[version]=%q, GetVersion()=%q", info["version"], version.GetVersion())
	}
	if info["commit"] != version.GetCommit() {
		t.Errorf("Info()[commit]=%q, GetCommit()=%q", info["commit"], version.GetCommit())
	}
	if info["buildTime"] != version.GetBuildTime() {
		t.Errorf("Info()[buildTime]=%q, GetBuildTime()=%q", info["buildTime"], version.GetBuildTime())
	}
	if info["releaseTrain"] != version.GetReleaseTrain() {
		t.Errorf("Info()[releaseTrain]=%q, GetReleaseTrain()=%q", info["releaseTrain"], version.GetReleaseTrain())
	}
	if info["uiBuildHash"] != version.GetUIBuildHash() {
		t.Errorf("Info()[uiBuildHash]=%q, GetUIBuildHash()=%q", info["uiBuildHash"], version.GetUIBuildHash())
	}
}

// Go stamps "+dirty" onto Main.Version itself when the working tree is
// modified, so the version must not carry a second marker of its own.
func TestBuildInfoVersionKeepsOneDirtyMarker(t *testing.T) {
	info := &debug.BuildInfo{
		Main: debug.Module{Version: "v0.94.91-0.20260902214000-d755fdbbda0c+dirty"},
		Settings: []debug.BuildSetting{
			{Key: "vcs.revision", Value: "d755fdbbda0cea6de351c3af3c745448f3bfe195"},
			{Key: "vcs.modified", Value: "true"},
		},
	}

	ver, _, _ := version.ExtractVersionFromBuildInfo(info)

	if got := strings.Count(ver, "dirty"); got != 1 {
		t.Errorf("version %q carries %d dirty markers, want exactly 1", ver, got)
	}
}
