package snmp

import (
	"fmt"
	"sync"
	"testing"

	"github.com/gosnmp/gosnmp"
)

func TestSNMPGroupRegistersRFC1213Objects(t *testing.T) {
	agent := NewAgent(createTestDevice(), 0)
	for suffix := 1; suffix <= 30; suffix++ {
		if value := agent.mib.Get(fmt.Sprintf("%s.%d.0", snmpGroup, suffix)); value == nil {
			t.Fatalf("SNMP group object %d is missing", suffix)
		}
	}
	control := agent.mib.Get(snmpGroup + ".30.0")
	if control.Type != gosnmp.Integer || control.Value != 2 {
		t.Fatalf("snmpEnableAuthenTraps = %#v, want disabled(2)", control)
	}
}

func TestSNMPGroupCountsProcessedRequests(t *testing.T) {
	agent := NewAgent(createTestDevice(), 0)
	agent.RecordInboundPacket()
	agent.ProcessPDU(gosnmp.GetRequest, []gosnmp.SnmpPDU{
		{Name: sysNameOID}, {Name: "1.3.6.1.2.1.999.0"},
	}, 0, 0)
	agent.ProcessPDU(gosnmp.GetNextRequest, []gosnmp.SnmpPDU{{Name: sysNameOID}}, 0, 0)
	agent.ProcessPDU(gosnmp.SetRequest, []gosnmp.SnmpPDU{{
		Name: snmpGroup + ".30.0", Type: gosnmp.Integer, Value: 1,
	}}, 0, 0)
	agent.RecordResponse(gosnmp.NoSuchName)

	wants := map[int]uint32{1: 1, 2: 1, 13: 3, 14: 1, 15: 1, 16: 1, 17: 1, 21: 1, 28: 1}
	for suffix, want := range wants {
		value := agent.mib.Get(fmt.Sprintf("%s.%d.0", snmpGroup, suffix))
		if got := value.Value.(uint32); got != want {
			t.Errorf("object %d = %d, want %d", suffix, got, want)
		}
	}
	if got := agent.mib.Get(snmpGroup + ".30.0").Value; got != 1 {
		t.Errorf("snmpEnableAuthenTraps = %v, want enabled(1)", got)
	}
}

func TestSNMPGroupRejectsInvalidAuthTrapControl(t *testing.T) {
	agent := NewAgent(createTestDevice(), 0)
	request := &gosnmp.SnmpPacket{Version: gosnmp.Version2c, PDUType: gosnmp.SetRequest, Variables: []gosnmp.SnmpPDU{{
		Name: snmpGroup + ".30.0", Type: gosnmp.Integer, Value: 3,
	}}}
	// A failed SET echoes the request varbinds and reports the failure in
	// error-status, which is what the protocol says and what a manager parses.
	response, status, _ := agent.ProcessRequest(request)
	if response[0].Name != snmpGroup+".30.0" {
		t.Fatalf("invalid SET response varbind = %v", response[0].Name)
	}
	if status != gosnmp.BadValue {
		t.Fatalf("invalid SET status = %v, want badValue", status)
	}
	agent.RecordResponse(status)
	if got := agent.mib.Get(snmpGroup + ".22.0").Value.(uint32); got != 1 {
		t.Fatalf("snmpOutBadValues = %d, want 1", got)
	}
}

func TestSNMPGroupV1MissingObjectProducesNoSuchName(t *testing.T) {
	agent := NewAgent(createTestDevice(), 0)
	request := &gosnmp.SnmpPacket{
		Version: gosnmp.Version1, PDUType: gosnmp.GetRequest,
		Variables: []gosnmp.SnmpPDU{{Name: "1.3.6.1.2.1.999.0"}},
	}
	_, status, index := agent.ProcessRequest(request)
	if status != gosnmp.NoSuchName || index != 1 {
		t.Fatalf("status/index = %v/%d, want noSuchName/1", status, index)
	}
	agent.RecordResponse(status)
	if got := agent.mib.Get(snmpGroup + ".21.0").Value.(uint32); got != 1 {
		t.Fatalf("snmpOutNoSuchNames = %d, want 1", got)
	}
}

func TestSNMPGroupCountersAreConcurrent(t *testing.T) {
	agent := NewAgent(createTestDevice(), 0)
	const workers = 32
	var wg sync.WaitGroup
	wg.Add(workers)
	for range workers {
		go func() {
			defer wg.Done()
			agent.RecordASNParseError()
			agent.ProcessPDU(gosnmp.GetRequest, []gosnmp.SnmpPDU{{Name: sysNameOID}}, 0, 0)
		}()
	}
	wg.Wait()
	if got := agent.mib.Get(snmpGroup + ".1.0").Value.(uint32); got != workers {
		t.Errorf("snmpInPkts = %d, want %d", got, workers)
	}
	if got := agent.mib.Get(snmpGroup + ".6.0").Value.(uint32); got != workers {
		t.Errorf("snmpInASNParseErrs = %d, want %d", got, workers)
	}
	if got := agent.mib.Get(snmpGroup + ".15.0").Value.(uint32); got != workers {
		t.Errorf("snmpInGetRequests = %d, want %d", got, workers)
	}
}
