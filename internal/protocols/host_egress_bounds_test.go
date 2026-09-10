package protocols

import (
	"errors"
	"net/netip"
	"testing"

	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/layers"

	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
)

func TestHostNeighborQueueBounds(t *testing.T) {
	stack, packet := hostEgressFixture(t, "192.0.3.20")
	armHostMask(t, stack, packet)
	route, _, err := stack.hostPacketRoute(packet)
	if err != nil {
		t.Fatal(err)
	}
	sender := stack.notifications.sender.(*stackDatagramSender)
	for index := range maxPendingNeighborTotal {
		route.target = netip.AddrFrom4([4]byte{192, 0, 3, byte(index/maxPendingNeighborDatagrams + 1)})
		packet = hostPacketDestination(t, packet, route.target)
		if err = sender.resolveHostNeighbor(packet, route, 200); err != nil {
			t.Fatalf("queue %d: %v", index, err)
		}
		if index == maxPendingNeighborDatagrams-1 {
			if err = sender.resolveHostNeighbor(packet, route, 200); !errors.Is(err, errHostNeighborQueueFull) {
				t.Fatalf("per-key bound: %v", err)
			}
		}
	}
	route.target = netip.MustParseAddr("192.0.3.250")
	if err = sender.resolveHostNeighbor(packet, route, 200); !errors.Is(err, errHostNeighborQueueFull) {
		t.Fatalf("global bound: %v", err)
	}
	stack.Stop()
	sender.mu.Lock()
	pending := len(sender.pending)
	sender.mu.Unlock()
	if pending != 0 {
		t.Fatal("stop retained pending queue")
	}
	if err = sender.resolveHostNeighbor(packet, route, 200); !errors.Is(err, ErrStackStopped) {
		t.Fatalf("queued after stop: %v", err)
	}
}

func hostPacketDestination(t *testing.T, packet *Packet, destination netip.Addr) *Packet {
	t.Helper()
	decoded := gopacket.NewPacket(packet.Buffer, layers.LayerTypeEthernet, gopacket.Default)
	ethernet, _ := decoded.Layer(layers.LayerTypeEthernet).(*layers.Ethernet)
	ip, _ := decoded.Layer(layers.LayerTypeIPv4).(*layers.IPv4)
	if ethernet == nil || ip == nil {
		t.Fatal("invalid host fixture")
	}
	ip.DstIP = destination.AsSlice()
	buffer := gopacket.NewSerializeBuffer()
	if err := gopacket.SerializeLayers(
		buffer,
		gopacket.SerializeOptions{FixLengths: true, ComputeChecksums: true},
		ethernet,
		ip,
		gopacket.Payload(ip.Payload),
	); err != nil {
		t.Fatal(err)
	}
	result := packet.Clone()
	result.Buffer, result.Length = buffer.Bytes(), len(buffer.Bytes())
	return result
}

func TestHostPendingReplyRejectsChangedSource(t *testing.T) {
	stack, packet := hostEgressFixture(t, "192.0.3.20")
	armHostMask(t, stack, packet)
	capture := &recordingCapture{}
	stack.capture = capture
	stack.sendPacket(packet)

	receiveRoutedReply(t, stack)
	store := stack.deviceStates[packet.generatedHost]
	if err := store.UpdateInterface("eth0", func(iface devicestate.Interface) (devicestate.Interface, error) {
		iface.Address = netip.MustParsePrefix("198.51.100.10/24")
		return iface, nil
	}); err != nil {
		t.Fatal(err)
	}
	if err := store.ClearInterfacePrefixFault("eth0", devicestate.FaultBadMask); err != nil {
		t.Fatal(err)
	}
	stack.notifications.observeNeighbor(200, netip.MustParseAddr("192.0.3.20"), mustParseMAC(t, "02:00:00:00:00:20"))
	stack.sendPacket(receiveRoutedReply(t, stack))
	if len(capture.frame) != 0 {
		t.Fatal("emitted packet using previous canonical address")
	}
}

func TestHostPendingReplyRejectsMovedOrRemovedDevice(t *testing.T) {
	for _, remove := range []bool{false, true} {
		stack, packet := hostEgressFixture(t, "192.0.3.20")
		armHostMask(t, stack, packet)
		capture := &recordingCapture{}
		stack.capture = capture
		stack.sendPacket(packet)

		receiveRoutedReply(t, stack)
		store := stack.deviceStates[packet.generatedHost]
		if err := store.ClearInterfacePrefixFault("eth0", devicestate.FaultBadMask); err != nil {
			t.Fatal(err)
		}
		if remove {
			delete(stack.deviceStates, packet.generatedHost)
		} else {
			if err := store.UpdateInterface("eth0", func(iface devicestate.Interface) (devicestate.Interface, error) {
				iface.Network = "other-network"
				return iface, nil
			}); err != nil {
				t.Fatal(err)
			}
		}
		stack.notifications.observeNeighbor(
			200,
			netip.MustParseAddr("192.0.3.20"),
			mustParseMAC(t, "02:00:00:00:00:20"),
		)
		stack.sendPacket(receiveRoutedReply(t, stack))
		if len(capture.frame) != 0 {
			t.Fatal("cleared packet escaped after move/removal")
		}
	}
}
