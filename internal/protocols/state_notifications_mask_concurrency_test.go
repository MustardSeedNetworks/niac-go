package protocols

import (
	"errors"
	"net/netip"
	"testing"
	"time"
)

func TestNotificationRetryHostQueueAvoidsNestedReloadLock(t *testing.T) {
	stack, packet := hostEgressFixture(t, "192.0.2.20")
	setNotificationMask(t, stack, packet, 30)
	stack.capture = &recordingCapture{}
	stack.sendPacket(packet)
	receiveRoutedReply(t, stack)
	sender := stack.notifications.sender.(*stackDatagramSender)
	key := newNotificationNeighborKey(200, netip.MustParseAddr("192.0.2.1"))
	sender.mu.Lock()
	resolution := sender.pending[key]
	resolution.timer.Stop()
	locked := true
	defer func() {
		if locked {
			sender.mu.Unlock()
		}
	}()
	finished := make(chan struct{})
	go func() { sender.retryNeighborResolution(key, resolution); close(finished) }()
	waitHostLockCondition(t, func() bool {
		if stack.reloadMu.TryLock() {
			stack.reloadMu.Unlock()
			return false
		}
		return true
	})
	writerDone := make(chan struct{})
	go func() { stack.reloadMu.Lock(); close(writerDone); stack.reloadMu.Unlock() }()
	waitHostLockCondition(t, func() bool {
		if stack.reloadMu.TryRLock() {
			stack.reloadMu.RUnlock()
			return false
		}
		return true
	})
	sender.mu.Unlock()
	locked = false
	select {
	case <-finished:
	case <-time.After(time.Second):
		t.Fatal("notification retry reentered reload lock while writer was queued")
	}
	select {
	case <-writerDone:
	case <-time.After(time.Second):
		t.Fatal("notification retry prevented reload writer completion")
	}
	assertNotificationProbe(t, receiveRoutedReply(t, stack), "192.0.2.10", "192.0.2.1")
}

func TestNotificationNeighborQueueRejectsStoppedStack(t *testing.T) {
	stack, packet := hostEgressFixture(t, "192.0.2.20")
	setNotificationMask(t, stack, packet, 30)
	stack.Stop()
	sender := stack.notifications.sender.(*stackDatagramSender)
	if err := sender.Send(
		packet.generatedHost,
		200,
		"192.0.2.20:514",
		514,
		[]byte("event"),
	); !errors.Is(
		err,
		ErrStackStopped,
	) {
		t.Fatalf("stopped notification error = %v", err)
	}
	sender.mu.Lock()
	pending := len(sender.pending)
	sender.mu.Unlock()
	if pending != 0 {
		t.Fatal("stopped stack retained notification resolution")
	}
}
