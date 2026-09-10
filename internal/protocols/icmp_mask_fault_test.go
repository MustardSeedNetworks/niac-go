package protocols

import (
	"bytes"
	"net"
	"testing"

	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/layers"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
)

func TestICMPMaskReplyPreservesVLANAndProjectsFault(t *testing.T) {
	stack, reply := hostEgressFixture(t, "192.0.2.20")
	device := reply.generatedHost
	device.ICMPConfig = &config.ICMPConfig{AddressMaskReply: net.ParseIP("255.255.255.0")}
	request := reply.Clone()
	request.PutSourceMAC(reply.GetDestMAC())
	request.PutDestMAC(device.MACAddress)
	store := stack.deviceStates[device]
	for _, bits := range []int{24, 16, 24} {
		if bits == 16 {
			armHostMask(t, stack, reply)
		} else {
			if err := store.ClearInterfacePrefixFault("eth0", devicestate.FaultBadMask); err != nil {
				t.Fatal(err)
			}
		}
		stack.icmpHandler.handleAddressMaskRequest(request,
			&layers.IPv4{SrcIP: net.ParseIP("192.0.2.20"), DstIP: net.ParseIP("192.0.2.10")},
			&layers.ICMPv4{Id: 12, Seq: 1}, []*config.Device{device})
		packet := receiveRoutedReply(t, stack)
		if packet.VLAN != 200 {
			t.Fatalf("mask reply VLAN=%d, want200", packet.VLAN)
		}
		decoded := gopacket.NewPacket(packet.Buffer, layers.LayerTypeEthernet, gopacket.Default)
		icmp, _ := decoded.Layer(layers.LayerTypeICMPv4).(*layers.ICMPv4)
		if icmp == nil || !bytes.Equal(icmp.Payload, net.CIDRMask(bits, 32)) {
			t.Fatalf("mask reply=%+v want /%d", icmp, bits)
		}
		if packet.generatedHost != device {
			t.Fatal("missing host response provenance")
		}
	}
}
