package protocols

import (
	"net"
	"testing"

	"github.com/gopacket/gopacket/layers"
)

// TestSendDNSResponseEchoesVLAN verifies a DNS reply is queued on the request's
// VLAN. A tester on an 802.1Q trunk sends tagged queries; an untagged reply is
// dropped before it reaches the sender (the pre-#876 bug, which had never been
// applied to the DNS handler — DNS used SendRawPacket, not SendRawPacketVLAN).
func TestSendDNSResponseEchoesVLAN(t *testing.T) {
	for _, vlan := range []int{0, 200} {
		stack := newTestStackInternal(t)
		h := NewDNSHandler(stack)

		response := &layers.DNS{ID: 1, QR: true, AA: true, ResponseCode: layers.DNSResponseCodeNoErr}

		err := h.SendDNSResponse(response,
			net.ParseIP("10.20.200.2"), net.ParseIP("10.20.200.250"),
			net.HardwareAddr{0x02, 0x00, 0x14, 0x03, 0x00, 0x01},
			net.HardwareAddr{0xaa, 0xaa, 0xaa, 0x00, 0x00, 0x01},
			45000, vlan)
		if err != nil {
			t.Fatalf("SendDNSResponse(vlan=%d): %v", vlan, err)
		}

		select {
		case pkt := <-stack.sendQueue:
			if pkt.VLAN != vlan {
				t.Errorf("DNS reply VLAN = %d, want %d (must echo the request VLAN)", pkt.VLAN, vlan)
			}
		default:
			t.Fatalf("no DNS reply queued for vlan=%d", vlan)
		}
	}
}
