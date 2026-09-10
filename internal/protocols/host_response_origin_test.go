package protocols

import (
	"net"
	"testing"

	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/layers"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
)

func TestGeneratedResponsesUseHostMaskRoute(t *testing.T) {
	for _, protocol := range []string{"udp", "dns", "health-syn", "health-payload"} {
		t.Run(protocol, func(t *testing.T) {
			stack, template := hostEgressFixture(t, "192.0.3.20")
			device := template.generatedHost
			armHostMask(t, stack, template)
			capture := &recordingCapture{}
			stack.capture = capture
			queueHostResponse(t, stack, template, protocol)
			response := receiveRoutedReply(t, stack)
			if response.generatedHost != device || response.Device != device {
				t.Fatal("response lost host identity")
			}
			stack.sendPacket(response)
			if len(capture.frame) != 0 {
				t.Fatal("response bypassed unresolved host route")
			}
			probe := receiveRoutedReply(t, stack)
			packet := gopacket.NewPacket(probe.Buffer, layers.LayerTypeEthernet, gopacket.Default)
			arp, ok := packet.Layer(layers.LayerTypeARP).(*layers.ARP)
			if !ok || !net.IP(arp.DstProtAddress).Equal(net.IPv4(192, 0, 3, 20)) || probe.VLAN != 200 {
				t.Fatalf("wrong response neighbor probe: %v", packet)
			}
		})
	}
}

func queueHostResponse(t *testing.T, stack *Stack, template *Packet, protocol string) {
	t.Helper()
	device := template.generatedHost
	source, destination := net.IPv4(192, 0, 2, 10), net.IPv4(192, 0, 3, 20)
	var err error
	switch protocol {
	case "udp":
		err = stack.udpHandler.SendUDP(device, source, destination, 161, 50000,
			[]byte("response"), device.MACAddress, template.GetDestMAC(), 200)
	case "dns":
		err = NewDNSHandler(stack).SendDNSResponse(device, &layers.DNS{ID: 77, QR: true},
			source, destination, device.MACAddress, template.GetDestMAC(), 50000, 200)
	case "health-syn", "health-payload":
		request := template.Clone()
		request.PutSourceMAC(template.GetDestMAC())
		ip := &layers.IPv4{SrcIP: destination, DstIP: source}
		tcp := &layers.TCP{SrcPort: 50000, DstPort: 104, SYN: true}
		handler := NewHealthCheckHandler(stack)
		if protocol == "health-syn" {
			handler.sendSYNACK(ip, tcp, []*config.Device{device}, request)
		} else {
			handler.sendTCPResponse(ip, tcp, []byte("response"), []*config.Device{device}, request)
		}
	default:
		t.Fatalf("unknown response protocol %q", protocol)
	}
	if err != nil {
		t.Fatal(err)
	}
}
