package protocols_test

import (
	"bytes"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/logging"
	"github.com/MustardSeedNetworks/niac-go/internal/protocols"
	"github.com/MustardSeedNetworks/niac-go/internal/scenario"
)

const (
	dot1dStpDesignatedRoot = "1.3.6.1.2.1.17.2.5.0"
	stpRootPrimaryPriority = 24576
)

// Every switch of a pack site must agree on one root, its primary core. A
// switch the site's trunks do not reach would name itself instead, and a
// consumer polling the site would see two trees.
func TestPackSitesElectTheirPrimaryCore(t *testing.T) {
	for _, pack := range scenario.Packs() {
		t.Run(pack.ID, func(t *testing.T) {
			result, err := scenario.Generate(pack.Request)
			if err != nil {
				t.Fatal(err)
			}
			stack := protocols.NewStack(nil, result.Config, logging.NewDebugConfig(0))
			bridges := stpBridges(result.Config)
			if len(bridges) == 0 {
				t.Fatal("no switch authors spanning tree")
			}
			roots := primaryRoots(bridges)
			for _, bridge := range bridges {
				assertNamesRoot(t, stack, bridge, roots[bridge.Properties["site"]])
			}
		})
	}
}

func assertNamesRoot(t *testing.T, stack *protocols.Stack, bridge, root *config.Device) {
	t.Helper()
	if root == nil {
		t.Fatalf("site %q has no switch at the root priority", bridge.Properties["site"])
	}
	want := append([]byte{stpRootPrimaryPriority >> 8, 0}, root.MACAddress...)
	got, err := stack.SNMPGet(bridge, dot1dStpDesignatedRoot)
	if err != nil {
		t.Fatalf("%s: %v", bridge.Name, err)
	}
	if named, _ := got.Value.([]byte); !bytes.Equal(named, want) {
		t.Errorf("%s names root %x, want its site's primary core %s (%x)", bridge.Name, named, root.Name, want)
	}
}

func stpBridges(cfg *config.Config) []*config.Device {
	var bridges []*config.Device
	for i := range cfg.Devices {
		if cfg.Devices[i].STPConfig != nil && cfg.Devices[i].STPConfig.Enabled {
			bridges = append(bridges, &cfg.Devices[i])
		}
	}
	return bridges
}

func primaryRoots(bridges []*config.Device) map[string]*config.Device {
	roots := make(map[string]*config.Device)
	for _, bridge := range bridges {
		if bridge.STPConfig.BridgePriority == stpRootPrimaryPriority {
			roots[bridge.Properties["site"]] = bridge
		}
	}
	return roots
}
