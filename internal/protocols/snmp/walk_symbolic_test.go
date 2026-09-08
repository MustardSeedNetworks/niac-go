package snmp

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gosnmp/gosnmp"
)

// symbolicWalk is the shape net-snmp emits when it is run without -On and has
// MIBs installed: object names instead of numbers, and — where it cannot
// resolve a name at all — the nearest SMI anchor with a numeric tail. The last
// two lines are the two kinds it must refuse: a symbolic INDEX (`ipv6`), which
// no name table can resolve, and an object from a MIB nothing ships.
const symbolicWalk = `SNMPv2-MIB::sysDescr.0 = STRING: Cisco NX-OS(tm) n5000
SNMPv2-SMI::enterprises.9.12.3.1.3.1008 = INTEGER: 1
SNMPv2-SMI::transmission.7.2.1.1.1 = INTEGER: 2
IF-MIB::ifDescr.99 = STRING: Ethernet1/1
IP-MIB::icmpMsgStatsOutPkts.ipv6.3 = Counter32: 5
FOO-MIB::barObject.1 = STRING: vendor private
`

// TestLoadWalkFileRefusesUnresolvableSymbolicOIDs pins the invariant that makes
// a walk replayable at all: every OID the agent stores must be numeric.
//
// A non-numeric key is not merely untidy. parseOIDParts drops any arc Atoi
// rejects, so `SNMPv2-MIB::sysORDescr.3` and `IP-MIB::icmpMsgStatsOutPkts.ipv6.3`
// both reduce to [3] and compare *equal* — the sorted list the GET-NEXT binary
// search walks is no longer ordered, and a chain through it goes backwards.
// Worse, gosnmp cannot marshal a non-numeric OID: MarshalMsg fails and the
// responder sends nothing at all, so one such row silently kills discovery of
// the whole device.
func TestLoadWalkFileRefusesUnresolvableSymbolicOIDs(t *testing.T) {
	path := filepath.Join(t.TempDir(), "symbolic.walk")
	if err := os.WriteFile(path, []byte(symbolicWalk), 0o600); err != nil {
		t.Fatal(err)
	}

	agent := NewAgent(createTestDevice(), 0)
	if err := agent.LoadWalkFile(path); err != nil {
		t.Fatalf("LoadWalkFile: %v", err)
	}

	assertEveryOIDReachesTheWire(t, agent)

	// The resolvable names arrive at their numeric form, values intact. sysDescr
	// is deliberately absent: it is signed substitution 2, so the scenario owns
	// it whatever the walk says, and ifDescr is asserted at an ifIndex the
	// device does not author so substitution 5 leaves the walk's row alone.
	for oid, want := range map[string]string{
		".1.3.6.1.4.1.9.12.3.1.3.1008": "1",
		".1.3.6.1.2.1.10.7.2.1.1.1":    "2",
		".1.3.6.1.2.1.2.2.1.2.99":      "Ethernet1/1",
	} {
		value, err := agent.HandleGet(oid)
		if err != nil {
			t.Errorf("resolvable symbolic OID %s did not load: %v", oid, err)

			continue
		}
		if got := oidValueString(value); got != want {
			t.Errorf("%s = %q, want %q", oid, got, want)
		}
	}

	assertGetNextVisitsEveryOIDInOrder(t, agent)
}

// assertEveryOIDReachesTheWire is the invariant in two halves: an OID the MIB
// holds must be numeric, and it must survive gosnmp's marshaller. The second
// half is the one that bites — a single unmarshallable varbind fails the whole
// response, so the agent answers nothing at all.
func assertEveryOIDReachesTheWire(t *testing.T, agent *Agent) {
	t.Helper()

	for _, oid := range agent.mib.AllOIDs() {
		if !IsValidOID(oid) {
			t.Errorf("non-numeric OID reached the MIB: %q", oid)

			continue
		}

		packet := &gosnmp.SnmpPacket{
			Version:   gosnmp.Version2c,
			Community: "public",
			PDUType:   gosnmp.GetResponse,
			Variables: []gosnmp.SnmpPDU{{Name: oid, Type: gosnmp.Null}},
		}
		if _, err := packet.MarshalMsg(); err != nil {
			t.Errorf("OID %q does not marshal onto the wire: %v", oid, err)
		}
	}
}

// assertGetNextVisitsEveryOIDInOrder chains GET-NEXT the way snmpwalk does.
// Two symbolic OIDs that reduce to the same arcs compare equal, which both
// breaks the ordering and makes the chain skip whatever sorted between them —
// so the count matters as much as the direction.
func assertGetNextVisitsEveryOIDInOrder(t *testing.T, agent *Agent) {
	t.Helper()

	held := len(agent.mib.AllOIDs())
	visited := 0
	previous := ".0"

	for visited <= held {
		next, value, err := agent.HandleGetNext(previous)
		if err != nil || value == nil {
			break
		}
		if compareOIDs(next, previous) <= 0 {
			t.Fatalf("GET-NEXT went backwards or stalled: %s after %s", next, previous)
		}
		previous = next
		visited++
	}

	if visited != held {
		t.Errorf("GET-NEXT chain visited %d OIDs, MIB holds %d", visited, held)
	}
}
