package snmp

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gosnmp/gosnmp"
)

// reproWalk is a small but representative sanitized walk. It includes:
//   - a real sysDescr (should override the config-derived value)
//   - ifTable / ifXTable style rows
//   - the malformed empty-value line reported from the sanitized catalog
//     (`... = ""` with no TYPE: prefix)
const reproWalk = `.1.3.6.1.2.1.1.1.0 = STRING: "Cisco IOS Software, C3750 Software"
.1.3.6.1.2.1.1.2.0 = OID: .1.3.6.1.4.1.9.1.516
.1.3.6.1.2.1.1.5.0 = STRING: "cos-acc-sw1"
.1.3.6.1.2.1.2.2.1.2.1 = STRING: "GigabitEthernet1/0/1"
.1.3.6.1.2.1.2.2.1.2.2 = STRING: "GigabitEthernet1/0/2"
.1.3.6.1.2.1.2.2.1.5.1 = Gauge32: 1000000000
.1.3.6.1.2.1.2.2.1.5.2 = Gauge32: 1000000000
.1.3.6.1.2.1.31.1.1.1.1.1 = STRING: "Gi1/0/1"
.1.3.6.1.2.1.31.1.1.1.1.2 = STRING: "Gi1/0/2"
.1.3.6.1.2.1.31.1.10.100.42.92 = ""
`

// TestReproSnmpwalk simulates what net-snmp `snmpwalk` does: a chain of
// GET-NEXT requests starting at the mib-2 root, following each returned OID,
// until end-of-mib. It asserts the walk-file entries are actually reachable.
func TestReproSnmpwalk(t *testing.T) {
	dir := t.TempDir()
	walkPath := filepath.Join(dir, "repro.walk")
	if err := os.WriteFile(walkPath, []byte(reproWalk), 0o600); err != nil {
		t.Fatal(err)
	}

	device := createTestDevice()
	device.Properties["sysDescr"] = "router test-device" // config-derived default
	agent := NewAgent(device, 0)

	if err := agent.LoadWalkFile(walkPath); err != nil {
		t.Fatalf("LoadWalkFile: %v", err)
	}

	// 1) GET on sysDescr must return the WALK value, not the config default.
	got, err := agent.HandleGet(".1.3.6.1.2.1.1.1.0")
	if err != nil {
		t.Fatalf("GET sysDescr: %v", err)
	}
	if s, _ := got.Value.(string); s != "Cisco IOS Software, C3750 Software" {
		t.Errorf("sysDescr = %q; want the walk value (config default leaked)", s)
	}

	// 2) Walk the whole mib-2 subtree via repeated GET-NEXT (like snmpwalk).
	root := ".1.3.6.1.2.1"
	current := root
	seen := map[string]bool{}
	var walked []string
	for range 100 {
		resp := agent.ProcessPDU(gosnmp.GetNextRequest, []gosnmp.SnmpPDU{{Name: current}}, 0)
		if len(resp) != 1 {
			t.Fatalf("GET-NEXT returned %d vars", len(resp))
		}
		v := resp[0]
		if v.Type == gosnmp.EndOfMibView || v.Type == gosnmp.NoSuchObject {
			break
		}
		name := normalizeName(v.Name)
		if !hasPrefixOID(name, root) {
			break // left the subtree — snmpwalk stops here
		}
		if seen[name] {
			t.Fatalf("GET-NEXT looped on %s (walk would never terminate)", name)
		}
		seen[name] = true
		walked = append(walked, name)
		current = v.Name
	}

	t.Logf("snmpwalk reached %d OIDs: %v", len(walked), walked)
	if len(walked) == 0 {
		t.Fatal("snmpwalk reached 0 OIDs — walk replay is broken")
	}
	// We expect at least the interface OIDs from the walk to be reachable.
	if !seen["1.3.6.1.2.1.2.2.1.2.1"] {
		t.Errorf("walk did not reach ifDescr.1 from the walk file")
	}
	if !seen["1.3.6.1.2.1.31.1.1.1.1.1"] {
		t.Errorf("walk did not reach ifName.1 from the walk file")
	}
}

func normalizeName(oid string) string {
	if len(oid) > 0 && oid[0] == '.' {
		return oid[1:]
	}
	return oid
}

func hasPrefixOID(oid, prefix string) bool {
	oid = normalizeName(oid)
	prefix = normalizeName(prefix)
	if len(oid) < len(prefix) {
		return false
	}
	if oid[:len(prefix)] != prefix {
		return false
	}
	return len(oid) == len(prefix) || oid[len(prefix)] == '.'
}
