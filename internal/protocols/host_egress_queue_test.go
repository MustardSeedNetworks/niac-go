package protocols

import (
	"bytes"
	"net/netip"
	"testing"

	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/layers"

	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
)

func armHostMask(t *testing.T, stack *Stack, packet *Packet) {
	t.Helper()
	if err := stack.deviceStates[packet.generatedHost].SetInterfacePrefixFault(devicestate.InterfacePrefixFault{
		Interface: "eth0", Type: devicestate.FaultBadMask, PrefixBits: 16,
	}); err != nil {
		t.Fatal(err)
	}
}

func TestHostEgressPendingNeighborAndClear(t *testing.T) {
	for _, clear := range []bool{false, true} {
		stack, packet := hostEgressFixture(t, "192.0.3.20")
		capture := &recordingCapture{}
		stack.capture = capture
		armHostMask(t, stack, packet)
		stack.sendPacket(packet)
		if len(capture.frame) != 0 {
			t.Fatal("host reply escaped before ARP")
		}
		probe := receiveRoutedReply(t, stack)
		decoded := gopacket.NewPacket(probe.Buffer, layers.LayerTypeEthernet, gopacket.Default)
		arp, _ := decoded.Layer(layers.LayerTypeARP).(*layers.ARP)
		if arp == nil || !bytes.Equal(arp.DstProtAddress, []byte{192, 0, 3, 20}) || probe.VLAN != 200 {
			t.Fatalf("wrong ARP probe: %+v", arp)
		}
		if clear {
			if err := stack.deviceStates[packet.generatedHost].ClearInterfacePrefixFault(
				"eth0",
				devicestate.FaultBadMask,
			); err != nil {
				t.Fatal(err)
			}
		}
		mac := mustParseMAC(t, "02:00:00:00:00:20")
		stack.notifications.observeNeighbor(201, netip.MustParseAddr("192.0.3.20"), mac)
		select {
		case <-stack.sendQueue:
			t.Fatal("other VLAN flushed packet")
		default:
		}
		stack.notifications.observeNeighbor(200, netip.MustParseAddr("192.0.3.20"), mac)
		resolved := receiveRoutedReply(t, stack)
		stack.sendPacket(resolved)
		if clear {
			mac = packet.GetDestMAC()
		}
		decoded = gopacket.NewPacket(capture.frame, layers.LayerTypeEthernet, gopacket.Default)
		eth, _ := decoded.Layer(layers.LayerTypeEthernet).(*layers.Ethernet)
		if eth == nil || !bytes.Equal(eth.DstMAC, mac) {
			t.Fatalf("incorrect resolved destination: %+v", eth)
		}
	}
}

func TestHostEgressPendingInterfaceDownAndStop(t *testing.T) {
	for _, stop := range []bool{false, true} {
		stack, packet := hostEgressFixture(t, "192.0.3.20")
		capture := &recordingCapture{}
		stack.capture = capture
		armHostMask(t, stack, packet)
		stack.sendPacket(packet)

		receiveRoutedReply(t, stack)
		if stop {
			stack.Stop()
		} else {
			if err := stack.deviceStates[packet.generatedHost].SetInterfaceFault(
				"eth0",
				devicestate.FaultLinkDown,
				100,
			); err != nil {
				t.Fatal(err)
			}
		}
		stack.notifications.observeNeighbor(
			200,
			netip.MustParseAddr("192.0.3.20"),
			mustParseMAC(t, "02:00:00:00:00:20"),
		)
		select {
		case queued := <-stack.sendQueue:
			stack.sendPacket(queued)
		default:
		}
		if len(capture.frame) != 0 {
			t.Fatal("inactive host emitted pending reply")
		}
	}
}
