package protocols

import (
	"bytes"
	"net"
	"net/netip"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
	"github.com/MustardSeedNetworks/niac-go/internal/fabric"
	"github.com/MustardSeedNetworks/niac-go/internal/logging"
)

func TestStackOwnsStateForEveryDevice(t *testing.T) {
	cfg := &config.Config{Devices: []config.Device{{Name: "edge-1"}}}
	stack := NewStack(nil, cfg, logging.NewDebugConfig(0))
	device := &cfg.Devices[0]

	state := stack.deviceStates[device]
	if state == nil {
		t.Fatal("device state was not registered")
	}
	if got := state.Snapshot().Identity.Hostname; got != "edge-1" {
		t.Fatalf("hostname = %q, want edge-1", got)
	}
}

func TestDiscoveryAdvertisementsReadAuthoritativeHostname(t *testing.T) {
	cfg := &config.Config{Devices: []config.Device{{
		Name: "edge-1", LLDPConfig: &config.LLDPConfig{Enabled: true},
		CDPConfig: &config.CDPConfig{Enabled: true},
	}}}
	stack := NewStack(nil, cfg, logging.NewDebugConfig(0))
	device := &cfg.Devices[0]
	state := stack.deviceStates[device]
	if err := state.UpdateIdentity(func(identity devicestate.Identity) (devicestate.Identity, error) {
		identity.Hostname = "branch-1"
		return identity, nil
	}); err != nil {
		t.Fatalf("UpdateIdentity() error = %v", err)
	}

	lldp := stack.lldpHandler.buildSystemNameTLV(device)
	cdp := stack.cdpHandler.buildDeviceIDTLV(device)
	if !bytes.Equal(lldp[2:], []byte("branch-1")) || !bytes.Equal(cdp[4:], []byte("branch-1")) {
		t.Fatalf("LLDP name = %q CDP name = %q", lldp[2:], cdp[4:])
	}
}

func TestSNMPAgentConsumesStackDeviceState(t *testing.T) {
	cfg := &config.Config{Devices: []config.Device{{
		Name:       "edge-1",
		SNMPConfig: config.SNMPConfig{Community: "public"},
	}}}
	stack := NewStack(nil, cfg, logging.NewDebugConfig(0))
	device := &cfg.Devices[0]
	group := stack.snmpAgents[device]

	if group == nil || group.state != stack.deviceStates[device] {
		t.Fatal("SNMP agent does not share the stack-owned device state")
	}
}

func TestConfigureFabricSeedsAuthoritativeNetworkState(t *testing.T) {
	cfg := &config.Config{Devices: []config.Device{{
		Name: "edge-1",
		Interfaces: []config.Interface{{
			Name: "Gi0/1", AdminStatus: "down", Description: "WAN", VLANs: []int{200},
		}},
	}}}
	stack := NewStack(nil, cfg, logging.NewDebugConfig(0))
	stack.ConfigureFabric(testDeviceStateTopology())

	snapshot := stack.deviceStates[&cfg.Devices[0]].Snapshot()
	if len(snapshot.Network.Interfaces) != 1 || len(snapshot.Network.Routes) != 1 {
		t.Fatalf("network state = %#v", snapshot.Network)
	}
	iface := snapshot.Network.Interfaces[0]
	if iface.AdminUp || iface.OperUp || iface.Description != "WAN" || iface.VLANs[0] != 200 {
		t.Fatalf("interface state = %#v", iface)
	}
}

func TestFlatConfigSeedsAuthoritativeNetworkState(t *testing.T) {
	cfg := &config.Config{Devices: []config.Device{{
		Name: "edge-1", IPAddresses: []net.IP{net.ParseIP("192.0.2.10")},
		Routes: []config.Route{{Destination: "198.51.100.0/24", Via: "Management", NextHop: "192.0.2.1"}},
	}}}
	stack := NewStack(nil, cfg, logging.NewDebugConfig(0))

	snapshot := stack.deviceStates[&cfg.Devices[0]].Snapshot()
	if len(snapshot.Network.Interfaces) != 1 || len(snapshot.Network.Routes) != 2 {
		t.Fatalf("network state = %#v", snapshot.Network)
	}
	if got := snapshot.Network.Interfaces[0]; got.Name != "Management" || got.Address.String() != "192.0.2.10/32" {
		t.Fatalf("interface state = %#v", got)
	}
	if got := snapshot.Network.Routes[1]; got.Destination.String() != "198.51.100.0/24" ||
		got.NextHop.String() != "192.0.2.1" {
		t.Fatalf("static route = %#v", got)
	}
}

func TestFlatConfigReloadReseedsAuthoritativeNetworkState(t *testing.T) {
	cfg := &config.Config{Devices: []config.Device{{
		Name: "edge-1", IPAddresses: []net.IP{net.ParseIP("192.0.2.10")},
	}}}
	stack := NewStack(nil, cfg, logging.NewDebugConfig(0))
	replacement := &config.Config{Devices: []config.Device{{
		Name: "edge-2", IPAddresses: []net.IP{net.ParseIP("198.51.100.20")},
	}}}

	if err := stack.ReloadConfig(replacement); err != nil {
		t.Fatalf("ReloadConfig() error = %v", err)
	}
	snapshot := stack.deviceStates[&replacement.Devices[0]].Snapshot()
	if len(snapshot.Network.Interfaces) != 1 ||
		snapshot.Network.Interfaces[0].Address.String() != "198.51.100.20/32" {
		t.Fatalf("network state = %#v", snapshot.Network)
	}
}

func testDeviceStateTopology() *fabric.Topology {
	return &fabric.Topology{
		Interfaces: []fabric.Interface{{
			Device: "edge-1", Name: "Gi0/1", Network: "wan", Address: netip.MustParsePrefix("10.0.0.1/24"),
		}},
		Routes: []fabric.Route{{
			Device: "edge-1", Destination: netip.MustParsePrefix("10.0.0.0/24"), Via: "Gi0/1", Connected: true,
		}},
	}
}
