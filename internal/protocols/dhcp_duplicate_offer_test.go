package protocols

import (
	"net"
	"net/netip"
	"testing"

	"github.com/gopacket/gopacket/layers"

	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
	"github.com/MustardSeedNetworks/niac-go/internal/logging"
)

func TestDuplicateDHCPOfferPreservesLeaseOwnership(t *testing.T) {
	stack, cfg := isolationPair(t)
	server := &cfg.Devices[0]
	if err := stack.SetDeviceAddressFault(
		server.Name,
		devicestate.FaultDuplicateDHCPOffer,
		netip.MustParseAddr("10.0.0.3"),
	); err != nil {
		t.Fatal(err)
	}
	client := net.HardwareAddr{2, 0, 0, 0, 1, 1}
	sendIsolationDHCP(t, stack, dhcpDiscover(client))
	assertDuplicateOffers(t, stack, map[string]string{"10.0.0.2": "10.0.0.3", "10.0.0.3": "10.0.0.110"})
	if len(stack.dhcpHandlers[server].leases) != 0 {
		t.Fatal("conflicting offer changed ordinary lease ownership")
	}
	if err := stack.ClearDeviceFault(server.Name, devicestate.FaultDuplicateDHCPOffer); err != nil {
		t.Fatal(err)
	}
	sendIsolationDHCP(t, stack, dhcpDiscover(client))
	assertDuplicateOffers(t, stack, map[string]string{"10.0.0.2": "10.0.0.100", "10.0.0.3": "10.0.0.110"})
}

func TestDuplicateDHCPOfferCannotCommitConflict(t *testing.T) {
	_, cfg := isolationPair(t)
	cfg.Devices[1].IPAddresses[0] = net.ParseIP("10.0.0.105")
	stack := NewStack(nil, cfg, logging.NewDebugConfig(0))
	if err := stack.SetDeviceAddressFault(
		"a",
		devicestate.FaultDuplicateDHCPOffer,
		netip.MustParseAddr("10.0.0.105"),
	); err != nil {
		t.Fatal(err)
	}
	client := net.HardwareAddr{2, 0, 0, 0, 1, 1}
	sendIsolationDHCP(t, stack, dhcpDiscover(client))
	assertDuplicateOffers(t, stack, map[string]string{"10.0.0.2": "10.0.0.105", "10.0.0.105": "10.0.0.110"})
	request := dhcpRequest(client, net.ParseIP("10.0.0.105"), 0)
	selectIsolationServer(request, "10.0.0.2")
	sendIsolationDHCP(t, stack, request)
	replies := isolationReplies(stack)
	if len(replies) != 1 || dhcpMsgType(dhcpOf(t, replies[0])) != DHCPNak {
		t.Fatal("offer-only fault committed the conflicting address")
	}
	if len(stack.dhcpHandlers[&cfg.Devices[0]].leases) != 0 {
		t.Fatal("conflicting REQUEST mutated ordinary leases")
	}
}

func assertDuplicateOffers(t *testing.T, stack *Stack, expected map[string]string) {
	t.Helper()
	for _, reply := range isolationReplies(stack) {
		dhcp := dhcpOf(t, reply)
		if dhcpMsgType(dhcp) != DHCPOffer {
			t.Fatalf("unexpected DHCP message %d", dhcpMsgType(dhcp))
		}
		identifier := ""
		for _, option := range dhcp.Options {
			if option.Type == layers.DHCPOptServerID {
				identifier = net.IP(option.Data).String()
			}
		}
		address, exists := expected[identifier]
		if !exists || dhcp.YourClientIP.String() != address {
			t.Fatalf("server %s offered %s, expected %v", identifier, dhcp.YourClientIP, expected)
		}
		delete(expected, identifier)
	}
	if len(expected) != 0 {
		t.Fatalf("missing offers: %v", expected)
	}
}
