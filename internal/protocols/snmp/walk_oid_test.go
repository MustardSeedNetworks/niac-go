package snmp

import (
	"strings"
	"testing"
)

func TestNormalizeKnownWalkOIDsPreservesValuesAndUnknownObjects(t *testing.T) {
	content := []byte(`SNMPv2-MIB::sysName.0 = STRING: "edge-1"
IF-MIB::ifName.48 = STRING: "Ethernet1/48"
VENDOR-MIB::privateObject.0 = Hex-STRING: 00 FF
`)
	normalized := string(NormalizeKnownWalkOIDs(content))
	for _, expected := range []string{
		`.1.3.6.1.2.1.1.5.0 = STRING: "edge-1"`,
		`.1.3.6.1.2.1.31.1.1.1.1.48 = STRING: "Ethernet1/48"`,
		`VENDOR-MIB::privateObject.0 = Hex-STRING: 00 FF`,
	} {
		if !strings.Contains(normalized, expected) {
			t.Errorf("normalized walk missing %q\n%s", expected, normalized)
		}
	}
}
