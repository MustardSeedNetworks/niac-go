package protocols

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/gosnmp/gosnmp"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/protocols/snmp"
)

const wireReproWalk = `.1.3.6.1.2.1.1.1.0 = STRING: "Cisco IOS Software, C3750"
.1.3.6.1.2.1.1.5.0 = STRING: "cos-acc-sw1"
.1.3.6.1.2.1.2.2.1.2.1 = STRING: "GigabitEthernet1/0/1"
.1.3.6.1.2.1.2.2.1.5.1 = Gauge32: 1000000000
.1.3.6.1.2.1.31.1.1.1.1.1 = STRING: "Gi1/0/1"
`

// buildAgent builds an agent with the walk loaded, mirroring stack_snmp.go.
func buildAgent(t *testing.T) *snmp.Agent {
	t.Helper()
	dir := t.TempDir()
	walkPath := filepath.Join(dir, "w.walk")
	if err := os.WriteFile(walkPath, []byte(wireReproWalk), 0o600); err != nil {
		t.Fatal(err)
	}
	dev := &config.Device{Name: "sw1", Type: "switch", Properties: map[string]string{}}
	agent := snmp.NewAgentWithCommunity(dev, "public", 0)
	if err := agent.LoadWalkFile(walkPath); err != nil {
		t.Fatal(err)
	}
	return agent
}

// roundTrip marshals an agent response the way the live handler does
// (internal/protocols/snmp.go buildResponse) and decodes it back the way a
// net-snmp client would, returning the decoded variables.
func roundTrip(t *testing.T, agent *snmp.Agent, pduType gosnmp.PDUType, name string, maxRep uint32) []gosnmp.SnmpPDU {
	t.Helper()
	respVars := agent.ProcessPDU(pduType, []gosnmp.SnmpPDU{{Name: name}}, 0, maxRep)
	resp := &gosnmp.SnmpPacket{
		Version:   gosnmp.Version2c,
		Community: "public",
		PDUType:   gosnmp.GetResponse,
		RequestID: 1,
		Variables: respVars,
	}
	raw, err := resp.MarshalMsg()
	if err != nil {
		t.Fatalf("marshal (%v): %v", pduType, err)
	}
	decoder := gosnmp.GoSNMP{Version: gosnmp.Version2c, Community: "public", MaxOids: gosnmp.MaxOids}
	decoded, err := decoder.SnmpDecodePacket(raw)
	if err != nil {
		t.Fatalf("client decode (%v): %v", pduType, err)
	}
	return decoded.Variables
}

func TestWireGetNext(t *testing.T) {
	agent := buildAgent(t)
	vars := roundTrip(t, agent, gosnmp.GetNextRequest, ".1.3.6.1.2.1.2.2.1.2", 0)
	if len(vars) != 1 {
		t.Fatalf("got %d vars", len(vars))
	}
	t.Logf("GETNEXT(.1.3.6.1.2.1.2.2.1.2) -> name=%q type=%v value=%v", vars[0].Name, vars[0].Type, vars[0].Value)
	if vars[0].Name != ".1.3.6.1.2.1.2.2.1.2.1" {
		t.Errorf("GETNEXT returned name %q; want .1.3.6.1.2.1.2.2.1.2.1", vars[0].Name)
	}
}

func TestWireGetBulkSnmpwalk(t *testing.T) {
	agent := buildAgent(t)
	// Simulate snmpbulkwalk: GETBULK from the mib-2 root, follow responses.
	root := ".1.3.6.1.2.1"
	current := root
	var reached []string
	for range 50 {
		vars := roundTrip(t, agent, gosnmp.GetBulkRequest, current, 10)
		if len(vars) == 0 {
			break
		}
		done := false
		for _, v := range vars {
			if v.Type == gosnmp.EndOfMibView || v.Type == gosnmp.NoSuchObject {
				done = true
				break
			}
			reached = append(reached, v.Name)
			current = v.Name
		}
		if done {
			break
		}
	}
	t.Logf("snmpbulkwalk reached %d OIDs: %v", len(reached), reached)
	if len(reached) == 0 {
		t.Fatal("GETBULK snmpwalk reached 0 OIDs")
	}
}

// TestWireGetBulkBoundedBySize reproduces the live walk stall: a run of large
// values (Cisco AGENT-CAPABILITIES strings are ~250 bytes) in one GET-BULK
// response overflows the MTU and the datagram is dropped, timing out the walk.
// The response must stay within one Ethernet frame, and a full bulk walk must
// still reach every OID (the manager continues from the last returned OID).
func TestWireGetBulkBoundedBySize(t *testing.T) {
	dev := &config.Device{Name: "sw", Type: "switch", Properties: map[string]string{}}
	agent := snmp.NewAgentWithCommunity(dev, "public", 0)

	// 40 OIDs each carrying a ~250-byte string under a private subtree.
	big := ""
	for len(big) < 250 {
		big += "AGENT-CAPABILITIES description padding. "
	}

	base := ".1.3.6.1.4.1.9.9.999.1.1"
	want := map[string]bool{}

	for i := 1; i <= 40; i++ {
		oid := base + "." + strconv.Itoa(i)
		_ = agent.SetOID(oid, &snmp.OIDValue{Type: gosnmp.OctetString, Value: big})
		want[oid[1:]] = true // stored without leading dot
	}

	// Every GET-BULK response with a generous maxRep must fit one frame, and a
	// full walk must still reach every OID (the manager continues from the last).
	current := base
	reached := map[string]bool{}

	for range 100 {
		respVars := agent.ProcessPDU(gosnmp.GetBulkRequest, []gosnmp.SnmpPDU{{Name: current}}, 0, 20)
		assertFitsFrame(t, respVars)

		next, done := advanceWalk(respVars, want, reached)
		if next != "" {
			current = next
		}

		if done || len(reached) == len(want) {
			break
		}
	}

	if len(reached) != len(want) {
		t.Errorf("full bulk walk reached %d/%d big-value OIDs", len(reached), len(want))
	}
}

// assertFitsFrame marshals a response and fails if it exceeds the MTU payload.
func assertFitsFrame(t *testing.T, respVars []gosnmp.SnmpPDU) {
	t.Helper()

	resp := &gosnmp.SnmpPacket{
		Version:   gosnmp.Version2c,
		Community: "public",
		PDUType:   gosnmp.GetResponse,
		RequestID: 1,
		Variables: respVars,
	}

	raw, err := resp.MarshalMsg()
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	if len(raw) > 1472 { // 1500 MTU - 20 IP - 8 UDP
		t.Fatalf("GET-BULK datagram %d bytes exceeds MTU payload budget", len(raw))
	}
}

// advanceWalk records which wanted OIDs a response covered and returns the OID
// to continue from plus whether end-of-MIB was reached.
func advanceWalk(respVars []gosnmp.SnmpPDU, want, reached map[string]bool) (string, bool) {
	next := ""

	for _, v := range respVars {
		if v.Type == gosnmp.EndOfMibView || v.Type == gosnmp.NoSuchObject {
			return next, true
		}

		name := v.Name
		if len(name) > 0 && name[0] == '.' {
			name = name[1:]
		}

		if want[name] {
			reached[name] = true
		}

		next = v.Name
	}

	return next, false
}
