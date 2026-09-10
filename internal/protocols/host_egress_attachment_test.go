package protocols

import (
	"bytes"
	"net/netip"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
	"github.com/MustardSeedNetworks/niac-go/internal/logging"
)

func TestHostEgressRoutedDevicesPreserveBytes(t *testing.T) {
	for _, role := range []string{"router", "layer3-switch", "firewall"} {
		t.Run(role, func(t *testing.T) {
			stack, packet := hostEgressFixture(t, "192.0.2.20")
			packet.generatedHost.Type = role
			store := stack.deviceStates[packet.generatedHost]
			state := store.ExportState()
			state.PrefixFaults = []devicestate.InterfacePrefixFault{{
				Interface: "eth0", Type: devicestate.FaultBadMask, PrefixBits: 30,
			}}
			if err := store.RestoreState(state); err != nil {
				t.Fatal(err)
			}
			before := bytes.Clone(packet.Buffer)
			prepared, err := stack.prepareHostEgress(packet)
			if err != nil || prepared != packet || !bytes.Equal(before, packet.Buffer) {
				t.Fatalf("routed device egress changed: prepared=%p original=%p error=%v", prepared, packet, err)
			}
		})
	}
}

func TestHostNeighborResolutionRejectsRemoteModeledInterfaces(t *testing.T) {
	cfg, topology, routerMAC := forwardingFixture(t)
	stack := NewStack(nil, cfg, logging.NewDebugConfig(0))
	stack.ConfigureFabric(topology)
	stack.running.Store(true)
	t.Cleanup(stack.Stop)
	sender := stack.notifications.sender.(*stackDatagramSender)
	for _, address := range []string{"10.20.0.10", "10.20.0.1"} {
		if mac := sender.hostNeighborMAC(0, netip.MustParseAddr(address)); mac != nil {
			t.Fatalf("resolved remote %s to %s", address, mac)
		}
	}
	address := netip.MustParseAddr("10.10.200.1")
	if mac := sender.hostNeighborMAC(0, address); !bytes.Equal(mac, routerMAC) {
		t.Fatalf("local gateway = %s", mac)
	}
	if err := stack.deviceStates[&cfg.Devices[0]].SetInterfaceFault(
		"outside",
		devicestate.FaultLinkDown,
		100,
	); err != nil {
		t.Fatal(err)
	}
	if mac := sender.hostNeighborMAC(0, address); mac != nil {
		t.Fatalf("resolved down gateway to %s", mac)
	}
	physicalMAC := mustParseMAC(t, "02:00:00:00:00:30")
	physical := netip.MustParseAddr("10.10.200.30")
	sender.observeNeighbor(0, physical, physicalMAC)
	if mac := sender.hostNeighborMAC(201, physical); mac != nil {
		t.Fatalf("learned neighbor leaked VLAN: %s", mac)
	}
	if mac := sender.hostNeighborMAC(0, physical); !bytes.Equal(mac, physicalMAC) {
		t.Fatalf("physical neighbor = %s", mac)
	}
}
