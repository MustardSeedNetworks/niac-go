package protocols

import (
	"os"
	"path/filepath"
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
	respVars := agent.ProcessPDU(pduType, []gosnmp.SnmpPDU{{Name: name}}, maxRep)
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
