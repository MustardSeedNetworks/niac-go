package protocols

import (
	"testing"
	"time"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
)

func TestStackPrefixFaultScopeAndClear(t *testing.T) {
	for _, role := range []string{"server", "router", "layer3-switch", "firewall"} {
		t.Run(role, func(t *testing.T) {
			stack, packet := hostEgressFixture(t, "192.0.2.20")
			device := packet.generatedHost
			device.Type = role
			fault := devicestate.InterfacePrefixFault{
				Interface:  "eth0",
				Type:       devicestate.FaultBadMask,
				PrefixBits: 0,
			}
			err := stack.SetInterfacePrefixFault(device.Name, fault)
			if role != "server" {
				if err == nil {
					t.Fatal("accepted routed-device mask fault")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			store := stack.deviceStates[device]
			if !store.HasInterfacePrefixFaults() {
				t.Fatal("zero prefix did not arm the fault")
			}
			if err = stack.ClearInterfacePrefixFault(device.Name, fault.Interface, fault.Type); err != nil {
				t.Fatal(err)
			}
			if store.HasInterfacePrefixFaults() {
				t.Fatal("explicit clear retained mask")
			}
		})
	}
}

func TestMaskPreflightRejectsRemoteAttachment(t *testing.T) {
	stack, cfg := isolationRoutedDHCP(t)
	for _, device := range []string{"a", "remote"} {
		cfg.BehaviorTimelines = []config.BehaviorTimeline{{
			Name: "mask", RepeatCount: 1,
			Phases: []config.BehaviorPhase{{
				Name: "fault", Duration: time.Second,
				Faults: []config.BehaviorFault{
					{Device: device, Interface: "eth0", Type: "bad_mask", PrefixBits: 16},
				},
			}},
		}}
		err := ValidateConfiguredBehaviorTargets(cfg, &stack.fabric.topology)
		if (err == nil) != (device == "a") {
			t.Fatalf("device %s: preflight error %v", device, err)
		}
		err = stack.SetInterfacePrefixFault(device, devicestate.InterfacePrefixFault{
			Interface: "eth0", Type: devicestate.FaultBadMask, PrefixBits: 16,
		})
		if (err == nil) != (device == "a") {
			t.Fatalf("device %s: setter error %v", device, err)
		}
	}
}

func TestMaskClearSurvivesLinkDown(t *testing.T) {
	stack, packet := hostEgressFixture(t, "192.0.2.20")
	device := packet.generatedHost
	fault := devicestate.InterfacePrefixFault{
		Interface:  "eth0",
		Type:       devicestate.FaultBadMask,
		PrefixBits: 16,
	}
	if err := stack.SetInterfacePrefixFault(device.Name, fault); err != nil {
		t.Fatal(err)
	}
	store := stack.deviceStates[device]
	if err := store.SetInterfaceFault("eth0", devicestate.FaultLinkDown, 1); err != nil {
		t.Fatal(err)
	}
	if err := stack.ClearInterfacePrefixFault(device.Name, "eth0", devicestate.FaultBadMask); err != nil {
		t.Fatal(err)
	}
	if store.HasInterfacePrefixFaults() || store.Snapshot().Network.Interfaces[0].OperUp {
		t.Fatal("mask clear failed or removed independent link-down")
	}
}
