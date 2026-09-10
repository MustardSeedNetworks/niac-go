package protocols

import (
	"runtime"
	"testing"
	"time"
)

func TestHostNeighborProbeDoesNotReenterReloadReadLock(t *testing.T) {
	stack, packet := hostEgressFixture(t, "192.0.3.20")
	armHostMask(t, stack, packet)
	sender := stack.notifications.sender.(*stackDatagramSender)
	sender.mu.Lock()
	locked := true
	defer func() {
		if locked {
			sender.mu.Unlock()
		}
	}()
	prepared := make(chan error, 1)
	go func() { _, err := stack.prepareHostEgress(packet); prepared <- err }()
	waitHostLockCondition(t, func() bool {
		if stack.reloadMu.TryLock() {
			stack.reloadMu.Unlock()
			return false
		}
		return true
	})
	writerAcquired := make(chan struct{})
	releaseWriter := make(chan struct{})
	writerReleased := false
	defer func() {
		if !writerReleased {
			close(releaseWriter)
		}
	}()
	go func() {
		stack.reloadMu.Lock()
		close(writerAcquired)
		<-releaseWriter
		stack.reloadMu.Unlock()
	}()
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
	case <-writerAcquired:
	case <-time.After(time.Second):
		t.Fatal("host probe held a nested read lock ahead of queued writer")
	}
	close(releaseWriter)
	writerReleased = true
	select {
	case err := <-prepared:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("host probe did not finish after reload writer")
	}
}

func waitHostLockCondition(t *testing.T, ready func() bool) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if ready() {
			return
		}
		runtime.Gosched()
	}
	t.Fatal("timed out arranging host egress lock contention")
}
