//go:build linux && integration

package wiretest_test

import (
	"net"
	"testing"

	"github.com/gopacket/gopacket/layers"
)

// A client on a network the DHCP server is not attached to leases through the
// gateway's relay: the offer comes from the relayed scope, with the relay as
// giaddr and the central server in option 54.
func TestRelayedDHCPLeasesFromTheCentralServer(t *testing.T) {
	startAuthoredFile(t, "dhcp-relay-wire.yaml", "wiretest-dhcp-relay")
	relay, server := net.ParseIP("10.254.200.7"), net.ParseIP("10.77.10.5")
	poolStart, poolEnd := net.ParseIP("10.254.200.150"), net.ParseIP("10.254.200.159")

	handle := openClient(t)
	src := clientMAC(t)
	xid := uint32(0x9e57d4c9)

	offer := dhcpExchange(t, handle, src, xid, layers.DHCPMsgTypeDiscover, nil)
	offered := append(net.IP(nil), offer.YourClientIP...)
	if !addrWithin(offered, poolStart, poolEnd) {
		t.Fatalf("DHCPOFFER yiaddr = %s, outside the relayed scope %s-%s", offered, poolStart, poolEnd)
	}
	if !offer.RelayAgentIP.Equal(relay) {
		t.Errorf("DHCPOFFER giaddr = %s, want the relay %s", offer.RelayAgentIP, relay)
	}
	if id := dhcpOptionIP(offer, layers.DHCPOptServerID); !id.Equal(server) {
		t.Errorf("DHCPOFFER server identifier = %s, want the central server %s", id, server)
	}

	ack := dhcpExchange(t, handle, src, xid, layers.DHCPMsgTypeRequest, offered)
	if !ack.YourClientIP.Equal(offered) {
		t.Errorf("DHCPACK yiaddr = %s, want the offered %s", ack.YourClientIP, offered)
	}
	if router := dhcpOptionIP(ack, layers.DHCPOptRouter); !router.Equal(relay) {
		t.Errorf("DHCPACK router option = %s, want the scope's %s", router, relay)
	}
	t.Logf("relayed lease %s via %s from %s", offered, relay, server)
}
