package protocols

import (
	"net"
	"testing"

	"github.com/gopacket/gopacket/layers"
)

// TestHandleDHCPInformAckHasNoLease verifies a DHCPINFORM gets a DHCPACK with
// configuration options but no address or lease (RFC 2131 §4.3.5).
func TestHandleDHCPInformAckHasNoLease(t *testing.T) {
	stack, h, device := newDHCPTestHandler(t)

	info := &dhcpPacketInfo{
		dhcp: &layers.DHCPv4{
			Operation:    layers.DHCPOpRequest,
			Xid:          0xABCD,
			ClientHWAddr: net.HardwareAddr{0xaa, 0xaa, 0xaa, 0x00, 0x00, 0x07},
			Options: []layers.DHCPOption{
				{Type: layers.DHCPOptMessageType, Length: 1, Data: []byte{DHCPInform}},
			},
		},
		messageType: DHCPInform,
		vlan:        200,
	}
	h.handleDHCPInform(info, device, 1, 0)

	d := dhcpOf(t, drainDHCP(t, stack))

	if mt := dhcpMsgType(d); mt != DHCPAck {
		t.Errorf("Inform reply message type = %d, want %d (ACK)", mt, DHCPAck)
	}

	if !d.YourClientIP.Equal(net.IPv4zero) {
		t.Errorf("Inform ACK yiaddr = %s, want 0.0.0.0 (no address assigned)", d.YourClientIP)
	}

	present := map[layers.DHCPOpt]bool{}
	for _, o := range d.Options {
		present[o.Type] = true
	}

	// No lease parameters (RFC 2131 §4.3.5).
	for _, forbidden := range []layers.DHCPOpt{layers.DHCPOptLeaseTime, layers.DHCPOptT1, layers.DHCPOptT2} {
		if present[forbidden] {
			t.Errorf("Inform ACK must not carry lease option %v", forbidden)
		}
	}

	// But config options are still delivered.
	if !present[layers.DHCPOptSubnetMask] {
		t.Error("Inform ACK should carry the subnet mask")
	}
}

// TestHandleDHCPDeclineQuarantines verifies a DHCPDECLINE quarantines the
// address so it is never granted again (RFC 2131 §4.3.3).
func TestHandleDHCPDeclineQuarantines(t *testing.T) {
	_, h, _ := newDHCPTestHandler(t)

	mac := net.HardwareAddr{0xaa, 0xaa, 0xaa, 0x00, 0x00, 0x09}
	ip := net.ParseIP("10.20.200.150")

	if !h.canGrantRequestedIP(mac, ip) {
		t.Fatal("in-pool address should be grantable before a decline")
	}

	info := &dhcpPacketInfo{
		dhcp: &layers.DHCPv4{
			ClientHWAddr: mac,
			Options: []layers.DHCPOption{
				{Type: layers.DHCPOptMessageType, Length: 1, Data: []byte{DHCPDecline}},
				{Type: layers.DHCPOptRequestIP, Length: 4, Data: ip.To4()},
			},
		},
		messageType: DHCPDecline,
	}
	h.handleDHCPDecline(info, 1, 0)

	if h.canGrantRequestedIP(mac, ip) {
		t.Error("a declined address must not be grantable")
	}

	if _, quarantined := h.declined[ip.String()]; !quarantined {
		t.Error("declined address should be recorded in the quarantine set")
	}
}
