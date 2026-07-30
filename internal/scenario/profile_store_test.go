package scenario_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/scenario"
)

func TestCustomProfileStoreRoundTrip(t *testing.T) {
	root := t.TempDir()
	profile := scenario.DeviceProfile{
		Role: "captured-access", DeviceType: "switch", Vendor: "cisco", Model: "Catalyst 9300",
		Platform: "Cisco IOS XE", Software: "17.15", SysObjectID: "1.3.6.1.4.1.9.1.2584",
		WalkName: "captured/captured-access.walk", InterfaceCount: 48,
		SupportedSNMPData: []string{"interfaces", "system"}, Source: "captured",
	}
	if err := scenario.SaveCustomProfile(root, profile); err != nil {
		t.Fatalf("SaveCustomProfile() error = %v", err)
	}
	profiles, err := scenario.CustomProfiles(root)
	if err != nil {
		t.Fatalf("CustomProfiles() error = %v", err)
	}
	if len(profiles) != 1 || profiles[0].Role != profile.Role || profiles[0].InterfaceCount != 48 {
		t.Fatalf("CustomProfiles() = %+v", profiles)
	}
	info, err := os.Stat(filepath.Join(root, "profiles", "captured-access.json"))
	if err != nil {
		t.Fatalf("stat saved profile: %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("profile mode = %o, want 600", info.Mode().Perm())
	}
	if err = scenario.SaveCustomProfile(root, profile); !errors.Is(err, scenario.ErrProfileExists) {
		t.Fatalf("duplicate error = %v, want ErrProfileExists", err)
	}
}

func TestSaveCustomProfileRejectsUnsafeAndBuiltInRoles(t *testing.T) {
	base := scenario.DeviceProfile{
		Role: "captured-switch", DeviceType: "switch", Vendor: "generic", Model: "Switch",
		Platform: "Network switch", WalkName: "captured/switch.walk", Source: "captured",
	}
	for _, role := range []string{"../escape", "access", "UPPER"} {
		profile := base
		profile.Role = role
		err := scenario.SaveCustomProfile(t.TempDir(), profile)
		if role == "access" {
			if !errors.Is(err, scenario.ErrProfileExists) {
				t.Errorf("role %q error = %v, want ErrProfileExists", role, err)
			}
		} else if !errors.Is(err, scenario.ErrInvalidProfile) {
			t.Errorf("role %q error = %v, want ErrInvalidProfile", role, err)
		}
	}
}

func TestSaveCustomProfileRejectsSymlinkedProfileDirectory(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(root, "profiles")); err != nil {
		t.Fatalf("create profile directory symlink: %v", err)
	}
	profile := scenario.DeviceProfile{
		Role: "captured-switch", DeviceType: "switch", Vendor: "generic", Model: "Switch",
		Platform: "Network switch", WalkName: "captured/switch.walk", Source: "captured",
	}
	if err := scenario.SaveCustomProfile(root, profile); err == nil {
		t.Fatal("SaveCustomProfile() error = nil, want symlink rejection")
	}
	if _, err := os.Stat(filepath.Join(outside, "captured-switch.json")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("outside profile error = %v, want os.ErrNotExist", err)
	}
}

func TestSaveCustomProfileRejectsUnknownDeviceType(t *testing.T) {
	profile := scenario.DeviceProfile{
		Role: "captured-unknown", DeviceType: "made-up", Vendor: "generic", Model: "Device",
		Platform: "Network device", WalkName: "captured/device.walk", Source: "captured",
	}
	if err := scenario.SaveCustomProfile(t.TempDir(), profile); !errors.Is(
		err,
		scenario.ErrInvalidProfile,
	) {
		t.Fatalf("SaveCustomProfile() error = %v, want ErrInvalidProfile", err)
	}
}

func TestSaveCustomProfileRejectsInterfaceOutsideDraftSchema(t *testing.T) {
	profile := scenario.DeviceProfile{
		Role: "captured-switch", DeviceType: "switch", Vendor: "generic", Model: "Switch",
		Platform: "Network switch", WalkName: "captured/switch.walk", Source: "captured",
		InterfaceCount: 1,
		Interfaces:     []scenario.ProfileInterface{{Name: "eth0", Type: "unsupported", MTU: 1500}},
	}
	if err := scenario.SaveCustomProfile(t.TempDir(), profile); !errors.Is(
		err,
		scenario.ErrInvalidProfile,
	) {
		t.Fatalf("SaveCustomProfile() error = %v, want ErrInvalidProfile", err)
	}
}
