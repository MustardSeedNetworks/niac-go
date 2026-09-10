package snmp

import (
	"runtime"
	"testing"
	"time"

	"github.com/gosnmp/gosnmp"
)

func TestResourceReadCompletesWithQueuedMIBWriter(t *testing.T) {
	agent, _ := resourceTestAgent(t)
	entered := make(chan struct{})
	resume := make(chan struct{})
	agent.mib.SetDynamic(hrStorageUsedPrefix+"10", func() *OIDValue {
		close(entered)
		<-resume
		return &OIDValue{Type: gosnmp.Integer, Value: 201}
	})
	agent.Reindex()
	readDone := make(chan *OIDValue, 1)
	go func() { readDone <- agent.mib.Get(hrStorageUsedPrefix + "10") }()
	deadline := time.NewTimer(2 * time.Second)
	defer deadline.Stop()
	select {
	case <-entered:
	case <-deadline.C:
		t.Fatal("resource callback did not start")
	}
	writerDone := make(chan struct{})
	go func() {
		agent.mib.Set("1.3.6.1.4.1.99999.1.0", &OIDValue{Type: gosnmp.Integer, Value: 1})
		close(writerDone)
	}()
	// The callback holds a read lock, so a rejected new reader proves that
	// a writer has queued, rather than merely that its goroutine has started.
	for agent.mib.mu.TryRLock() {
		agent.mib.mu.RUnlock()
		select {
		case <-deadline.C:
			close(resume)
			t.Fatal("MIB writer did not queue")
		default:
			runtime.Gosched()
		}
	}
	close(resume)
	select {
	case value := <-readDone:
		if value == nil || value.Type != gosnmp.Integer || oidValueString(value) != "201" {
			t.Fatalf("resource read = %#v, want INTEGER 201", value)
		}
	case <-deadline.C:
		t.Fatal("resource read deadlocked behind queued MIB writer")
	}
	select {
	case <-writerDone:
	case <-deadline.C:
		t.Fatal("MIB writer did not finish after resource read")
	}
}
