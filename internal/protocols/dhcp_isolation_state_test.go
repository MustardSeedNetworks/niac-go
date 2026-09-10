package protocols

import (
	"net"
	"net/netip"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/logging"
)

func TestDHCPDerivedIdentityRemainsOwned(t *testing.T) {
	_, cfg := isolationPair(t)
	cfg.Devices[1].DHCPConfig.ServerIdentifier = net.ParseIP("10.0.0.9")
	stack := NewStack(nil, cfg, logging.NewDebugConfig(0))
	for index := range cfg.Devices {
		store := stack.deviceStates[&cfg.Devices[index]]
		network := store.Snapshot().Network
		network.Interfaces[0].Address = netip.MustParsePrefix("10.0.0.8/24")
		store.ReplaceNetwork(network)
	}
	if got := stack.dhcpHandlers[&cfg.Devices[0]].configuredServerIP(); !got.Equal(net.ParseIP("10.0.0.8")) {
		t.Fatalf("derived server address = %s", got)
	}
	if got := stack.dhcpHandlers[&cfg.Devices[1]].configuredServerIP(); !got.Equal(net.ParseIP("10.0.0.9")) {
		t.Fatalf("explicit server identifier was overwritten: %s", got)
	}
}

func TestDHCPFDBLearningStaysInIngressSegment(t *testing.T) {
	cfg := &config.Config{Segments: []config.Segment{
		{Tag: 200, Devices: []config.Device{isolationDHCPServer("a", "10.0.0.2", "10.0.0.100", "10.0.0.109", 2)}},
		{Tag: 300, Devices: []config.Device{isolationDHCPServer("b", "10.0.0.2", "10.0.0.100", "10.0.0.109", 3)}},
	}}
	for index := range cfg.Segments {
		cfg.Segments[index].Devices[0].SNMPConfig = config.SNMPConfig{
			Community: "test", Dot1DFdbTable: &config.FdbTableConfig{Port: 7},
		}
	}
	stack := NewStack(nil, cfg, logging.NewDebugConfig(0))
	mac := net.HardwareAddr{2, 0, 0, 0, 1, 1}
	request := dhcpRequest(mac, net.ParseIP("10.0.0.100"), 200)
	selectIsolationServer(request, "10.0.0.2")
	sendIsolationDHCP(t, stack, request)
	if got := dhcpMsgType(dhcpOf(t, drainDHCP(t, stack))); got != DHCPAck {
		t.Fatalf("message = %d", got)
	}
	macOID, _ := formatMACForFDB(mac)
	for index := range cfg.Segments {
		agent := stack.snmpAgents[&cfg.Segments[index].Devices[0]].Get("test")
		value, err := agent.HandleGet(".1.3.6.1.2.1.17.4.3.1.2" + macOID)
		if index == 0 && (err != nil || value == nil) {
			t.Fatalf("ingress FDB missing learned client: %v", err)
		}
		if index == 1 && err == nil {
			t.Fatal("client FDB leaked into other segment")
		}
	}
}
