package protocols

import (
	"bytes"
	"net"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/logging"
)

func TestSNMPTopologyUsesAuthoredCDPPeerAddress(t *testing.T) {
	cfg := &config.Config{Devices: []config.Device{
		{
			Name:        "LAB-EDGE-R1",
			Type:        "router",
			MACAddress:  mustTestMAC(t, "00:00:0c:ff:01:01"),
			IPAddresses: []net.IP{net.ParseIP("10.254.200.1")},
			Interfaces: []config.Interface{
				{Name: "GigabitEthernet1/0/48", Address: "10.254.200.1/24"},
				{Name: "GigabitEthernet1/0/1", Address: "203.0.113.1/30"},
			},
			SNMPConfig: config.SNMPConfig{Community: "NetAllyDemo"},
			CDPConfig:  &config.CDPConfig{Enabled: true},
			TrunkPorts: []config.TrunkPort{{
				Interface:       "GigabitEthernet1/0/1",
				RemoteDevice:    "INET-R1",
				RemoteInterface: "GigabitEthernet1/0/1",
			}},
		},
		{
			Name:        "INET-R1",
			Type:        "router",
			MACAddress:  mustTestMAC(t, "00:00:0c:00:01:01"),
			IPAddresses: []net.IP{net.ParseIP("203.0.113.2"), net.ParseIP("8.8.8.8")},
			Interfaces: []config.Interface{{
				Name: "GigabitEthernet1/0/1", Address: "203.0.113.2/30",
			}},
		},
	}}

	stack := NewStack(nil, cfg, logging.NewDebugConfig(0))
	agent := stack.snmpAgents[&cfg.Devices[0]].baseAgent
	value, err := agent.HandleGet("1.3.6.1.4.1.9.9.23.1.2.1.1.4.1.1")
	if err != nil {
		t.Fatalf("get cdpCacheAddress: %v", err)
	}

	want := net.ParseIP("203.0.113.2").To4()
	got, ok := value.Value.([]byte)
	if !ok || !bytes.Equal(got, want) {
		t.Fatalf("cdpCacheAddress = %v, want %v", value.Value, want)
	}
}

func TestSNMPTopologyPublishesRouterARPForConnectedHost(t *testing.T) {
	hostMAC := mustTestMAC(t, "00:1a:2b:00:00:21")
	cfg := &config.Config{Devices: []config.Device{
		{
			Name: "COS-CORE-SW01", Type: "router",
			MACAddress: mustTestMAC(t, "00:00:0c:00:01:01"),
			Interfaces: []config.Interface{{
				Name: "Vlan210", Address: "10.240.210.2/24",
			}},
			SNMPConfig: config.SNMPConfig{Community: "NetAllyDemo"},
		},
		{
			Name: "COS-WS-B01-F01-01", Type: "host", MACAddress: hostMAC,
			Interfaces: []config.Interface{{
				Name: "eth0", Address: "10.240.210.21/24",
			}},
		},
		{
			Name: "REMOTE-WS01", Type: "host",
			MACAddress: mustTestMAC(t, "00:1a:2b:00:01:21"),
			Interfaces: []config.Interface{{
				Name: "eth0", Address: "10.241.210.21/24",
			}},
		},
	}}

	stack := NewStack(nil, cfg, logging.NewDebugConfig(0))
	agent := stack.snmpAgents[&cfg.Devices[0]].baseAgent
	const arpMAC = "1.3.6.1.2.1.4.22.1.2.1.10.240.210.21"
	value, err := agent.HandleGet(arpMAC)
	if err != nil {
		t.Fatalf("get connected host ARP entry: %v", err)
	}
	got, ok := value.Value.([]byte)
	if !ok || !bytes.Equal(got, hostMAC) {
		t.Fatalf("connected host ARP MAC = %v, want %v", value.Value, hostMAC)
	}
	const remoteARP = "1.3.6.1.2.1.4.22.1.2.1.10.241.210.21"
	if _, remoteErr := agent.HandleGet(remoteARP); remoteErr == nil {
		t.Fatal("router published ARP entry for a host outside its connected subnet")
	}
}

func mustTestMAC(t *testing.T, raw string) net.HardwareAddr {
	t.Helper()
	mac, err := net.ParseMAC(raw)
	if err != nil {
		t.Fatalf("parse MAC %q: %v", raw, err)
	}
	return mac
}
