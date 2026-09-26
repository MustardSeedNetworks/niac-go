package scenario_test

import (
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
)

// assertPackSNMPv3 pins where the packs author SNMPv3: the network,
// controller and server tier an NMS is configured to poll securely answers as
// the one published authPriv user, and endpoint appliances stay v2c-only. It
// reads the loaded config, which is what the runtime builds its USM engines
// from, and runs inside the per-pack loop that already paid for generating it.
func assertPackSNMPv3(t *testing.T, cfg *config.Config) {
	t.Helper()
	managed := map[string]bool{
		"switch": true, "layer3-switch": true, "router": true,
		"firewall": true, "access-point": true, "server": true,
	}
	want := config.SNMPv3User{
		Username:     "netops",
		AuthProtocol: "sha256", AuthPassword: "NetAllyDemoAuth",
		PrivProtocol: "aes", PrivPassword: "NetAllyDemoPriv",
	}
	served := 0
	for index := range cfg.Devices {
		device := &cfg.Devices[index]
		v3 := config.SNMPv3Enabled(device.SNMPv3Config)
		switch {
		case !managed[device.Type] && v3:
			t.Errorf("%s (%s) serves SNMPv3; endpoints stay v2c-only", device.Name, device.Type)
		case !managed[device.Type]:
		case !v3:
			t.Errorf("managed %s (%s) serves no SNMPv3", device.Name, device.Type)
		default:
			if users := device.SNMPv3Config.Users; len(users) != 1 || users[0] != want {
				t.Errorf("%s SNMPv3 users = %+v, want exactly %+v", device.Name, users, want)
			}
			served++
		}
	}
	if served == 0 {
		t.Error("no device serves SNMPv3")
	}
}
