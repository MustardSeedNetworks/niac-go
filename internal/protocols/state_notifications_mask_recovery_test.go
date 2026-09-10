package protocols

import (
	"net/netip"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
)

func TestNotificationMaskPendingRejectsInvalidOrigin(t *testing.T) {
	for _, change := range []string{"down", "link-fault", "move", "remove"} {
		t.Run(change, func(t *testing.T) {
			stack, packet := hostEgressFixture(t, "192.0.2.20")
			setNotificationMask(t, stack, packet, 30)
			sender := stack.notifications.sender.(*stackDatagramSender)
			if err := sender.Send(packet.generatedHost, 200, "192.0.2.20:514", 514, []byte("event")); err != nil {
				t.Fatal(err)
			}
			receiveRoutedReply(t, stack)
			invalidateNotificationOrigin(t, stack, packet, change)
			sender.observeNeighbor(200, netip.MustParseAddr("192.0.2.1"), mustParseMAC(t, "02:00:00:00:00:99"))
			select {
			case <-stack.sendQueue:
				t.Fatal("invalid notification origin escaped")
			default:
			}
		})
	}
}

func invalidateNotificationOrigin(t *testing.T, stack *Stack, packet *Packet, change string) {
	t.Helper()
	store := stack.deviceStates[packet.generatedHost]
	store.ClearAllFaults()
	if change == "remove" {
		delete(stack.deviceStates, packet.generatedHost)
		return
	}
	if change == "link-fault" {
		if err := store.SetInterfaceFault("eth0", devicestate.FaultLinkDown, 100); err != nil {
			t.Fatal(err)
		}
		return
	}
	if err := store.UpdateInterface("eth0", func(iface devicestate.Interface) (devicestate.Interface, error) {
		if change == "down" {
			iface.OperUp = false
		} else {
			iface.Network = "other"
		}
		return iface, nil
	}); err != nil {
		t.Fatal(err)
	}
}

func TestNotificationMaskRetryRefreshesMixedQueue(t *testing.T) {
	stack, packet := hostEgressFixture(t, "192.0.2.20")
	setNotificationMask(t, stack, packet, 30)
	stack.capture = &recordingCapture{}
	stack.sendPacket(packet)
	receiveRoutedReply(t, stack)
	sender := stack.notifications.sender.(*stackDatagramSender)
	if err := sender.Send(packet.generatedHost, 200, "192.0.2.20:514", 514, []byte("event")); err != nil {
		t.Fatal(err)
	}
	store := stack.deviceStates[packet.generatedHost]
	if err := store.UpdateInterface("eth0", func(iface devicestate.Interface) (devicestate.Interface, error) {
		iface.Address = netip.MustParsePrefix("192.0.2.11/24")
		return iface, nil
	}); err != nil {
		t.Fatal(err)
	}
	key := newNotificationNeighborKey(200, netip.MustParseAddr("192.0.2.1"))
	sender.mu.Lock()
	resolution := sender.pending[key]
	resolution.timer.Stop()
	sender.mu.Unlock()
	sender.retryNeighborResolution(key, resolution)
	assertNotificationProbe(t, receiveRoutedReply(t, stack), "192.0.2.11", "192.0.2.1")
	sender.observeNeighbor(200, key.address, mustParseMAC(t, "02:00:00:00:00:98"))
	staleReply := receiveRoutedReply(t, stack)
	if prepared, err := stack.prepareHostEgress(staleReply); err == nil || prepared != nil {
		t.Fatal("stale host reply survived notification source update")
	}
	assertNotificationDatagram(t, receiveRoutedReply(t, stack), "192.0.2.11")
}

func TestNotificationMaskRetryRetainsBudget(t *testing.T) {
	stack, packet := hostEgressFixture(t, "192.0.2.20")
	setNotificationMask(t, stack, packet, 30)
	sender := stack.notifications.sender.(*stackDatagramSender)
	if err := sender.Send(packet.generatedHost, 200, "192.0.2.20:514", 514, []byte("event")); err != nil {
		t.Fatal(err)
	}
	receiveRoutedReply(t, stack)
	key := newNotificationNeighborKey(200, netip.MustParseAddr("192.0.2.1"))
	sender.mu.Lock()
	resolution := sender.pending[key]
	resolution.timer.Stop()
	resolution.attempts = notificationNeighborRetries
	sender.mu.Unlock()
	sender.retryNeighborResolution(key, resolution)
	sender.mu.Lock()
	pending := len(sender.pending)
	sender.mu.Unlock()
	if pending != 0 {
		t.Fatal("notification refresh reset retry budget")
	}
	select {
	case <-stack.sendQueue:
		t.Fatal("expired notification retry emitted a probe")
	default:
	}
}
