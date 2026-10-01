package scenario_test

import (
	"strings"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/scenario"
)

const uplinkBundleMembers = 2

// P5-9: a distribution switch reaches each core switch over a two-port
// bundle, and both ends describe the same bundle. One end naming a
// port-channel the other lacks would publish a LAG whose neighbour rows point
// at a port that does not exist.
func TestPackCoreDistributionUplinksAreBundles(t *testing.T) {
	total := 0
	for _, pack := range scenario.Packs() {
		t.Run(pack.ID, func(t *testing.T) {
			total += assertUplinkBundles(t, packConfig(t, pack))
		})
	}
	if total == 0 {
		t.Fatal("no pack has a distribution tier, so nothing was checked")
	}
	t.Logf("%d core-distribution bundles across the packs", total)
}

// assertUplinkBundles checks every distribution switch's core uplinks and
// returns how many bundles it found.
func assertUplinkBundles(t *testing.T, cfg *config.Config) int {
	t.Helper()
	bundles, distributions := 0, 0
	for index := range cfg.Devices {
		distribution := &cfg.Devices[index]
		if !strings.Contains(distribution.Name, "-DIST-SW") {
			continue
		}
		distributions++
		for _, trunk := range distribution.TrunkPorts {
			if !strings.Contains(trunk.RemoteDevice, "-CORE-SW") {
				continue
			}
			core := findDevice(cfg, trunk.RemoteDevice)
			assertBundleEnd(t, distribution, trunk.Interface)
			assertBundleEnd(t, core, trunk.RemoteInterface)
			back := findRemotePort(core, distribution.Name)
			if back == nil || back.Interface != trunk.RemoteInterface ||
				back.RemoteInterface != trunk.Interface {
				t.Errorf("%s %s -> %s %s is not mirrored: %+v",
					distribution.Name, trunk.Interface, core.Name, trunk.RemoteInterface, back)
			}
			bundles++
		}
	}
	if bundles < distributions {
		t.Fatalf("%d core-distribution bundles for %d distribution switches", bundles, distributions)
	}
	return bundles
}

func assertBundleEnd(t *testing.T, device *config.Device, name string) {
	t.Helper()
	channel, ok := config.PortChannelFor(device, name)
	if !ok {
		t.Errorf("%s uplink %s is not a port-channel", device.Name, name)
		return
	}
	if len(channel.Members) != uplinkBundleMembers {
		t.Errorf("%s %s has members %v, want %d", device.Name, name, channel.Members, uplinkBundleMembers)
	}
	// An authored type would override the ieee8023adLag the name implies, and
	// the aggregate's speed is its members'.
	aggregate := findInterface(device, name)
	if aggregate == nil || aggregate.Type != "" {
		t.Errorf("%s %s is not authored as an untyped aggregate: %+v", device.Name, name, aggregate)
		return
	}
	speed := 0
	for _, member := range channel.Members {
		authored := findInterface(device, member)
		if authored == nil {
			t.Errorf("%s %s member %s is not an authored interface", device.Name, name, member)
			continue
		}
		speed += authored.Speed
		if isTrunked(device, member) {
			t.Errorf("%s %s member %s is also a trunk of its own", device.Name, name, member)
		}
	}
	if aggregate.Speed != speed {
		t.Errorf("%s %s speed = %d, want its members' %d", device.Name, name, aggregate.Speed, speed)
	}
}

func isTrunked(device *config.Device, name string) bool {
	for _, trunk := range device.TrunkPorts {
		if trunk.Interface == name {
			return true
		}
	}
	return false
}
