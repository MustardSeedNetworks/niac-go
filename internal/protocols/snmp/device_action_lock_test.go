package snmp

import (
	"runtime"
	"testing"
	"time"

	"github.com/gosnmp/gosnmp"

	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
)

func TestDeviceActionReadDoesNotRelockMIBBehindWriter(t *testing.T) {
	agent, state := actionProjectionAgent()
	entered, resume := make(chan struct{}), make(chan struct{})
	agent.mib.SetDynamic(dot1dStpTopChanges, func() *OIDValue {
		close(entered)
		<-resume
		return &OIDValue{Type: gosnmp.Counter32, Value: uint32(41)}
	})
	agent.Reindex()
	executeProjectionAction(t, state, devicestate.ActionSTPTopologyChange)
	readDone := make(chan *OIDValue, 1)
	go func() { readDone <- agent.mib.Get(dot1dStpTopChanges) }()
	deadline := time.NewTimer(2 * time.Second)
	defer deadline.Stop()
	select {
	case <-entered:
	case <-deadline.C:
		t.Fatal("action callback did not start")
	}
	writerDone := make(chan struct{})
	go func() {
		agent.mib.Set("1.3.6.1.4.1.99999.1.0", &OIDValue{Type: gosnmp.Integer, Value: 1})
		close(writerDone)
	}()
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
		if value.Type != gosnmp.Counter32 || value.Value != uint32(42) {
			t.Fatalf("action value = %#v", value)
		}
	case <-deadline.C:
		t.Fatal("action read deadlocked")
	}
	select {
	case <-writerDone:
	case <-deadline.C:
		t.Fatal("MIB writer did not finish")
	}
}
