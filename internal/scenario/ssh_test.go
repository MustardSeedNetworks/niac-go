package scenario_test

import (
	"strings"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/scenario"
)

// Packs serve SSH on their IOS XE gear with the banner a scanner expects and no
// account, so a pack starts with no daemon secret and every login is refused
// (D-NIAC-43). An authored username or password_env would either need a secret
// or invite a default credential.
func TestPacksServeSSHWithoutAnAccount(t *testing.T) {
	t.Parallel()

	for _, pack := range scenario.Packs() {
		cfg := generatePack(t, pack.ID)
		if err := config.ValidateRuntimeRequirements(cfg); err != nil {
			t.Errorf("%s: %v, want a pack to start with no secret set", pack.ID, err)
		}
		serving := 0
		for index := range cfg.Devices {
			if checkPackSSH(t, pack.ID, &cfg.Devices[index]) {
				serving++
			}
		}
		if serving == 0 {
			t.Errorf("%s: no device serves SSH", pack.ID)
		}
	}
}

// checkPackSSH reports whether device serves SSH, failing t if it serves it
// where it should not, or serves it with an account or the wrong banner.
func checkPackSSH(t *testing.T, packID string, device *config.Device) bool {
	t.Helper()
	software := device.Properties["software"]
	wantSSH := strings.HasPrefix(software, "IOS XE") && device.Properties["role"] != "ap"
	ssh := device.SSHConfig
	if ssh == nil || !ssh.Enabled {
		if wantSSH {
			t.Errorf("%s: %s runs %s and serves no SSH", packID, device.Name, software)
		}
		return false
	}
	if !wantSSH {
		t.Errorf("%s: %s runs %s, whose banner is not taken from hardware, and serves SSH",
			packID, device.Name, software)
	}
	if ssh.Username != "" || ssh.PasswordEnv != "" {
		t.Errorf("%s: %s authors an SSH account %q/%q", packID, device.Name, ssh.Username, ssh.PasswordEnv)
	}
	if device.OSFingerprintConfig == nil || device.OSFingerprintConfig.SSHBanner != "SSH-2.0-Cisco-1.25" {
		t.Errorf("%s: %s serves SSH without the IOS XE banner", packID, device.Name)
	}
	return true
}
