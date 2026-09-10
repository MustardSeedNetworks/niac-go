package protocols

import (
	"bytes"
	"net/netip"
	"testing"

	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/layers"

	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
)

func TestHostNeighborRetryRejectsStaleSource(t *testing.T) {
	stack, packet := hostEgressFixture(t, "192.0.3.20")
	armHostMask(t, stack, packet)
	stack.capture = &recordingCapture{}
	stack.sendPacket(packet)
	receiveRoutedReply(t, stack)
	if err := stack.deviceStates[packet.generatedHost].UpdateInterface(
		"eth0",
		func(iface devicestate.Interface) (devicestate.Interface, error) {
			iface.Address = netip.MustParsePrefix("198.51.100.10/24")
			return iface, nil
		},
	); err != nil {
		t.Fatal(err)
	}
	sender := stack.notifications.sender.(*stackDatagramSender)
	sender.probeNeighbor(newNotificationNeighborKey(200, netip.MustParseAddr("192.0.3.20")))
	queued := receiveRoutedReply(t, stack)
	decoded := gopacket.NewPacket(queued.Buffer, layers.LayerTypeEthernet, gopacket.Default)
	if arp := decoded.Layer(layers.LayerTypeARP); arp != nil {
		t.Fatal("retry emitted ARP from obsolete canonical address")
	}
	stack.sendPacket(queued)
	sender.mu.Lock()
	pending := len(sender.pending)
	sender.mu.Unlock()
	if pending != 0 {
		t.Fatal("obsolete resolution retained")
	}
}

func TestHostNeighborRetryUsesNotificationInMixedQueue(t *testing.T) {
	stack, packet := hostEgressFixture(t, "192.0.3.20")
	armHostMask(t, stack, packet)
	stack.capture = &recordingCapture{}
	stack.sendPacket(packet)
	receiveRoutedReply(t, stack)
	sender := stack.notifications.sender.(*stackDatagramSender)
	target := netip.MustParseAddr("192.0.3.20")
	source := netip.MustParseAddr("192.0.2.11")
	if err := stack.deviceStates[packet.generatedHost].UpdateInterface(
		"eth0",
		func(iface devicestate.Interface) (devicestate.Interface, error) {
			iface.Address = netip.PrefixFrom(source, 24)
			return iface, nil
		},
	); err != nil {
		t.Fatal(err)
	}
	if err := sender.resolveNeighbor(pendingNotification{
		egressDevice: packet.generatedHost, vlan: 200, neighborSource: source,
		source: source, destination: target, sourcePort: 514, destinationPort: 514, payload: []byte("event"),
	}, target); err != nil {
		t.Fatal(err)
	}
	sender.probeNeighbor(newNotificationNeighborKey(200, target))
	probe := receiveRoutedReply(t, stack)
	decoded := gopacket.NewPacket(probe.Buffer, layers.LayerTypeEthernet, gopacket.Default)
	arp, _ := decoded.Layer(layers.LayerTypeARP).(*layers.ARP)
	if arp == nil || !bytes.Equal(arp.SourceProtAddress, source.AsSlice()) {
		t.Fatalf("mixed queue probe=%+v", arp)
	}
}

func TestHostNeighborOldTimerDoesNotProbeReplacement(t *testing.T) {
	stack, packet := hostEgressFixture(t, "192.0.3.20")
	armHostMask(t, stack, packet)
	stack.capture = &recordingCapture{}
	stack.sendPacket(packet)
	receiveRoutedReply(t, stack)
	sender := stack.notifications.sender.(*stackDatagramSender)
	key := newNotificationNeighborKey(200, netip.MustParseAddr("192.0.3.20"))
	sender.mu.Lock()
	old := sender.pending[key]
	old.timer.Stop()
	replacement := &notificationNeighborResolution{hostPackets: old.hostPackets}
	sender.pending[key] = replacement
	sender.mu.Unlock()
	sender.probeNeighborResolution(key, old)
	sender.mu.Lock()
	attempts := replacement.attempts
	sender.mu.Unlock()
	if attempts != 0 {
		t.Fatal("stale timer consumed replacement retry budget")
	}
	select {
	case <-stack.sendQueue:
		t.Fatal("stale timer queued replacement probe")
	default:
	}
}
