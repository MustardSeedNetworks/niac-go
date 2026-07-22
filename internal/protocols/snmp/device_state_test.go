package snmp

import (
	"net/netip"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
)

func TestAgentSysNameReadsSharedDeviceState(t *testing.T) {
	device := &config.Device{Name: "configured-name"}
	state := devicestate.NewStore(devicestate.Identity{Hostname: "edge-1"})
	agent := NewAgentWithState(device, state, AgentOptions{})

	assertSysName(t, agent, "edge-1")
	err := state.UpdateIdentity(func(identity devicestate.Identity) (devicestate.Identity, error) {
		identity.Hostname = "edge-2"
		return identity, nil
	})
	if err != nil {
		t.Fatalf("UpdateIdentity() error = %v", err)
	}
	assertSysName(t, agent, "edge-2")
}

func TestAgentInterfaceStatusReadsSharedDeviceState(t *testing.T) {
	device := &config.Device{
		Name: "edge-1",
		TrunkPorts: []config.TrunkPort{
			{Interface: "GigabitEthernet0/1"},
		},
	}
	state := devicestate.NewStore(devicestate.Identity{Hostname: device.Name})
	state.ReplaceNetwork(devicestate.Network{Interfaces: []devicestate.Interface{
		{Name: "GigabitEthernet0/1", AdminUp: true, OperUp: true},
	}})
	agent := NewAgentWithState(device, state, AgentOptions{})

	assertMIBValue(t, agent, ifAdminStatus+".1", interfaceStatusUp)
	assertMIBValue(t, agent, ifOperStatus+".1", interfaceStatusUp)
	err := state.UpdateInterface(
		"GigabitEthernet0/1",
		func(iface devicestate.Interface) (devicestate.Interface, error) {
			iface.AdminUp = false
			iface.OperUp = false
			iface.Description = "Branch uplink"
			return iface, nil
		},
	)
	if err != nil {
		t.Fatalf("UpdateInterface() error = %v", err)
	}
	assertMIBValue(t, agent, ifAdminStatus+".1", interfaceStatusDown)
	assertMIBValue(t, agent, ifOperStatus+".1", interfaceStatusDown)
	assertMIBValue(t, agent, ifAlias+".1", "Branch uplink")
}

func TestAgentDiscoveryIdentityReadsSharedDeviceState(t *testing.T) {
	device := &config.Device{
		Name: "configured-name", LLDPConfig: &config.LLDPConfig{Enabled: true},
		CDPConfig: &config.CDPConfig{Enabled: true},
	}
	state := devicestate.NewStore(devicestate.Identity{Hostname: "edge-1"})
	agent := NewAgentWithState(device, state, AgentOptions{})
	if err := state.UpdateIdentity(func(identity devicestate.Identity) (devicestate.Identity, error) {
		identity.Hostname = "branch-1"
		return identity, nil
	}); err != nil {
		t.Fatalf("UpdateIdentity() error = %v", err)
	}

	assertMIBValue(t, agent, lldpLocSysName, "branch-1")
	assertMIBValue(t, agent, cdpGlobalDeviceID, "branch-1")
}

func TestAgentIPAddressTableTracksSharedDeviceState(t *testing.T) {
	device := &config.Device{
		Name:       "edge-1",
		TrunkPorts: []config.TrunkPort{{Interface: "GigabitEthernet0/1"}},
	}
	state := devicestate.NewStore(devicestate.Identity{Hostname: device.Name})
	state.ReplaceNetwork(devicestate.Network{
		Interfaces: []devicestate.Interface{{
			Name: "GigabitEthernet0/1", Address: mustStatePrefix("10.0.0.1/24"),
			AdminUp: true, OperUp: true,
		}},
		Routes: []devicestate.Route{{
			Destination: mustStatePrefix("10.2.0.0/16"), Via: "GigabitEthernet0/1",
			NextHop: netip.MustParseAddr("10.0.0.2"),
		}},
	})
	agent := NewAgentWithState(device, state, AgentOptions{})
	assertMIBValue(t, agent, ipAdEntAddr+".10.0.0.1", "10.0.0.1")
	assertMIBValue(t, agent, ipRouteNextHop+".10.2.0.0", "10.0.0.2")
	if err := state.UpdateInterface(
		"GigabitEthernet0/1",
		func(iface devicestate.Interface) (devicestate.Interface, error) {
			iface.Address = mustStatePrefix("10.0.1.1/24")
			return iface, nil
		},
	); err != nil {
		t.Fatalf("UpdateInterface() error = %v", err)
	}
	if got := state.Snapshot().Network.Interfaces[0].Address.String(); got != "10.0.1.1/24" {
		t.Fatalf("state address = %q", got)
	}
	if _, ok := agent.InterfaceIndex("GigabitEthernet0/1"); !ok {
		t.Fatal("interface index disappeared")
	}
	value, err := agent.HandleGet(ipAdEntAddr + ".10.0.1.1")
	if err != nil || value.Value != "10.0.1.1" {
		t.Fatalf("new IP address value = %#v error = %v", value, err)
	}
	if _, oldErr := agent.HandleGet(ipAdEntAddr + ".10.0.0.1"); oldErr == nil {
		t.Fatal("old IP address remains in ipAddrTable")
	}
	state.UpsertRoute(devicestate.Route{
		Destination: mustStatePrefix("10.2.0.0/16"), Via: "GigabitEthernet0/1",
		NextHop: netip.MustParseAddr("10.0.1.2"),
	})
	value, err = agent.HandleGet(ipRouteNextHop + ".10.2.0.0")
	if err != nil || value.Value != "10.0.1.2" {
		t.Fatalf("route next hop value = %#v error = %v", value, err)
	}
}

func mustStatePrefix(value string) netip.Prefix {
	return netip.MustParsePrefix(value)
}

func assertSysName(t *testing.T, agent *Agent, want string) {
	t.Helper()
	value, err := agent.HandleGet(sysNameOID)
	if err != nil {
		t.Fatalf("HandleGet(sysName) error = %v", err)
	}
	if got, ok := value.Value.(string); !ok || got != want {
		t.Fatalf("sysName = %#v, want %q", value.Value, want)
	}
}
