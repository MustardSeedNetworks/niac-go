package protocols

import (
	"net"
	"testing"

	"github.com/gopacket/gopacket/layers"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
	"github.com/MustardSeedNetworks/niac-go/internal/logging"
)

func isolationPair(t *testing.T) (*Stack, *config.Config) {
	t.Helper()
	cfg := &config.Config{Devices: []config.Device{
		isolationDHCPServer("a", "10.0.0.2", "10.0.0.100", "10.0.0.109", 2),
		isolationDHCPServer("b", "10.0.0.3", "10.0.0.110", "10.0.0.119", 3),
	}}
	return NewStack(nil, cfg, logging.NewDebugConfig(0)), cfg
}

func selectIsolationServer(info *dhcpPacketInfo, server string) {
	info.dhcp.Options = append(info.dhcp.Options, layers.DHCPOption{
		Type: layers.DHCPOptServerID, Length: 4, Data: net.ParseIP(server).To4(),
	})
}

func TestDHCPSelectedServerAloneCommits(t *testing.T) {
	stack, cfg := isolationPair(t)
	mac := net.HardwareAddr{2, 0, 0, 0, 1, 1}
	sendIsolationDHCP(t, stack, dhcpDiscover(mac))
	if got := len(isolationReplies(stack)); got != 2 {
		t.Fatalf("offers = %d", got)
	}
	peer := stack.dhcpHandlers[&cfg.Devices[1]]
	before := *peer.leases[mac.String()]
	request := dhcpRequest(mac, net.ParseIP("10.0.0.100"), 0)
	selectIsolationServer(request, "10.0.0.2")
	sendIsolationDHCP(t, stack, request)
	replies := isolationReplies(stack)
	if len(replies) != 1 || dhcpMsgType(dhcpOf(t, replies[0])) != DHCPAck {
		t.Fatalf("selected request produced %d replies, want one ACK", len(replies))
	}
	if after := peer.leases[mac.String()]; !after.Expiry.Equal(before.Expiry) || !after.IP.Equal(before.IP) {
		t.Fatalf("unselected server mutated lease: before=%+v after=%+v", before, after)
	}
}

func TestDHCPUnknownSelectedServerIsSilent(t *testing.T) {
	stack, cfg := isolationPair(t)
	request := dhcpRequest(net.HardwareAddr{2, 0, 0, 0, 1, 1}, net.ParseIP("10.0.0.100"), 0)
	selectIsolationServer(request, "10.0.0.99")
	sendIsolationDHCP(t, stack, request)
	if len(isolationReplies(stack)) != 0 {
		t.Fatal("request selecting an unknown server received a response")
	}
	for index := range cfg.Devices {
		if len(stack.dhcpHandlers[&cfg.Devices[index]].leases) != 0 {
			t.Fatal("unknown selection mutated lease inventory")
		}
	}
}

func TestDHCPNoOfferDoesNotSilencePeer(t *testing.T) {
	stack, cfg := isolationPair(t)
	if err := stack.SetDeviceFault("a", devicestate.FaultDHCPNoOffer, 1); err != nil {
		t.Fatal(err)
	}
	sendIsolationDHCP(t, stack, dhcpDiscover(net.HardwareAddr{2, 0, 0, 0, 1, 1}))
	replies := isolationReplies(stack)
	if len(replies) != 1 || !dhcpOf(t, replies[0]).YourClientIP.Equal(net.ParseIP("10.0.0.110")) {
		t.Fatalf("no-offer peer response count = %d", len(replies))
	}
	if len(stack.dhcpHandlers[&cfg.Devices[0]].leases) != 0 {
		t.Fatal("silent server allocated a lease")
	}
}

func TestDHCPSegmentsIsolateIdenticalClients(t *testing.T) {
	cfg := &config.Config{Segments: []config.Segment{
		{Tag: 200, Devices: []config.Device{isolationDHCPServer("a", "10.0.0.2", "10.0.0.100", "10.0.0.109", 2)}},
		{Tag: 300, Devices: []config.Device{isolationDHCPServer("a", "10.0.0.2", "10.0.0.100", "10.0.0.109", 2)}},
	}}
	stack := NewStack(nil, cfg, logging.NewDebugConfig(0))
	for _, vlan := range []int{200, 300} {
		info := dhcpDiscover(net.HardwareAddr{2, 0, 0, 0, 1, 1})
		info.vlan = vlan
		sendIsolationDHCP(t, stack, info)
		replies := isolationReplies(stack)
		if len(replies) != 1 || replies[0].VLAN != vlan {
			t.Fatalf("VLAN %d replies = %v", vlan, replies)
		}
	}
	first, second := stack.dhcpHandlers[&cfg.Segments[0].Devices[0]], stack.dhcpHandlers[&cfg.Segments[1].Devices[0]]
	if first == second || len(first.leases) != 1 || len(second.leases) != 1 {
		t.Fatal("identical VLAN client did not acquire independent leases")
	}
	decline := dhcpRequest(net.HardwareAddr{2, 0, 0, 0, 1, 1}, net.ParseIP("10.0.0.100"), 200)
	decline.dhcp.Options[0].Data[0] = DHCPDecline
	selectIsolationServer(decline, "10.0.0.2")
	sendIsolationDHCP(t, stack, decline)
	if len(first.declined) != 1 || len(first.leases) != 0 || len(second.declined) != 0 || len(second.leases) != 1 {
		t.Fatal("decline crossed segment ownership")
	}
}
