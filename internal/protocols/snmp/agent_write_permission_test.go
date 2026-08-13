package snmp

import (
	"testing"

	"github.com/gosnmp/gosnmp"
)

// A real agent refuses to write a read-only object. Accepting every SET is not
// just permissive - it gives a scanner the wrong answer: a compliance check that
// asks "can I write to this device?" is told yes, when the device it is
// imitating would have said no. It also lets a stray SET rename a device and
// silently put the simulation out of step with its authored truth.
func TestSetRequestRefusesReadOnlyObjects(t *testing.T) {
	agent := NewAgent(createTestDevice(), 0)
	const sysName = "1.3.6.1.2.1.1.5.0"
	before := agent.mib.Get(sysName)

	_, status, index := agent.ProcessRequest(&gosnmp.SnmpPacket{
		Version: gosnmp.Version2c, PDUType: gosnmp.SetRequest,
		Variables: []gosnmp.SnmpPDU{
			{Name: sysName, Type: gosnmp.OctetString, Value: "attacker-renamed-me"},
		},
	})

	if status != gosnmp.NotWritable {
		t.Errorf("error-status = %v, want NotWritable", status)
	}
	if index != 1 {
		t.Errorf("error-index = %d, want 1", index)
	}
	after := agent.mib.Get(sysName)
	if before != nil && after != nil && after.Value != before.Value {
		t.Errorf("sysName changed to %v", after.Value)
	}
}

// The one object this agent genuinely implements as writable stays writable.
func TestSetRequestAcceptsTheWritableObject(t *testing.T) {
	agent := NewAgent(createTestDevice(), 0)

	_, status, _ := agent.ProcessRequest(&gosnmp.SnmpPacket{
		Version: gosnmp.Version2c, PDUType: gosnmp.SetRequest,
		Variables: []gosnmp.SnmpPDU{
			{Name: "1.3.6.1.2.1.11.30.0", Type: gosnmp.Integer, Value: 2},
		},
	})

	if status != gosnmp.NoError {
		t.Errorf("error-status = %v, want NoError", status)
	}
}

// A writable object still rejects a value outside its range, and says so as
// wrongValue rather than as a permission problem.
func TestSetRequestRejectsABadValueOnAWritableObject(t *testing.T) {
	agent := NewAgent(createTestDevice(), 0)

	_, status, _ := agent.ProcessRequest(&gosnmp.SnmpPacket{
		Version: gosnmp.Version2c, PDUType: gosnmp.SetRequest,
		Variables: []gosnmp.SnmpPDU{
			{Name: "1.3.6.1.2.1.11.30.0", Type: gosnmp.Integer, Value: 7},
		},
	})

	if status != gosnmp.BadValue {
		t.Errorf("error-status = %v, want BadValue", status)
	}
}

// The simulation still publishes its own MIB freely: the restriction is on what
// a manager may write over the wire, not on what NIAC authors.
func TestTheSimulationStillPublishesItsOwnValues(t *testing.T) {
	agent := NewAgent(createTestDevice(), 0)
	const ifDescr = "1.3.6.1.2.1.2.2.1.2.1"

	if err := agent.SetOID(ifDescr, &OIDValue{
		Type: gosnmp.OctetString, Value: "GigabitEthernet0/1",
	}); err != nil {
		t.Fatalf("SetOID: %v", err)
	}
	value := agent.mib.Get(ifDescr)
	if value == nil || value.Value != "GigabitEthernet0/1" {
		t.Errorf("authored value = %v", value)
	}
}

// notWritable is an SNMPv2c status with no version 1 equivalent, so a version 1
// manager has to be told something it understands.
func TestVersionOneIsToldNoSuchName(t *testing.T) {
	agent := NewAgent(createTestDevice(), 0)

	_, status, _ := agent.ProcessRequest(&gosnmp.SnmpPacket{
		Version: gosnmp.Version1, PDUType: gosnmp.SetRequest,
		Variables: []gosnmp.SnmpPDU{
			{Name: "1.3.6.1.2.1.1.5.0", Type: gosnmp.OctetString, Value: "nope"},
		},
	})

	if status != gosnmp.NoSuchName {
		t.Errorf("SNMPv1 error-status = %v, want NoSuchName", status)
	}
}
