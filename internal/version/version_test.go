package version_test

import (
	"slices"
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

	expectedKeys := []string{"version", "commit", "buildTime", "uiBuildHash"}
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
	if info["uiBuildHash"] != version.GetUIBuildHash() {
		t.Errorf("Info()[uiBuildHash]=%q, GetUIBuildHash()=%q", info["uiBuildHash"], version.GetUIBuildHash())
	}
}
