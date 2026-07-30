package snmp

import (
	"strings"
	"testing"

	"github.com/gosnmp/gosnmp"
)

func TestFormatWalkEntriesPreservesTextAndBinaryOctets(t *testing.T) {
	content := string(FormatWalkEntries([]WalkEntry{
		{OID: ".1.3.6.1.2.1.1.5.0", Type: gosnmp.OctetString, Value: []byte("switch-1")},
		{
			OID:   ".1.3.6.1.2.1.2.2.1.6.1",
			Type:  gosnmp.OctetString,
			Value: []byte{0, 17, 34, 51, 68, 85},
		},
		{OID: ".1.3.6.1.2.1.2.1.0", Type: gosnmp.Integer, Value: 48},
		{OID: ".1.3.6.1.4.1.1.0", Type: gosnmp.Uinteger32, Value: uint32(7)},
		{OID: ".1.3.6.1.4.1.2.0", Type: gosnmp.OctetString, Value: []byte(`quoted "path\name"`)},
		{OID: ".1.3.6.1.4.1.3.0", Type: gosnmp.BitString, Value: []byte{0x80, 0x01}},
		{OID: ".1.3.6.1.4.1.4.0", Type: gosnmp.Opaque, Value: []byte{0, 0xff}},
		{OID: ".1.3.6.1.4.1.5.0", Type: gosnmp.OpaqueFloat, Value: float32(1.25)},
		{OID: ".1.3.6.1.4.1.6.0", Type: gosnmp.OpaqueDouble, Value: 2.5},
	}))
	for _, expected := range []string{
		`.1.3.6.1.2.1.1.5.0 = STRING: "switch-1"`,
		`.1.3.6.1.2.1.2.2.1.6.1 = Hex-STRING: 00 11 22 33 44 55`,
		`.1.3.6.1.2.1.2.1.0 = INTEGER: 48`,
		`.1.3.6.1.4.1.1.0 = UInteger32: 7`,
		`.1.3.6.1.4.1.2.0 = Hex-STRING: 71 75 6F 74 65 64 20 22 70 61 74 68 5C 6E 61 6D 65 22`,
		`.1.3.6.1.4.1.3.0 = BITS: HEX 80 01`,
		`.1.3.6.1.4.1.4.0 = Opaque: HEX 00 FF`,
		`.1.3.6.1.4.1.5.0 = Opaque Float: 1.25`,
		`.1.3.6.1.4.1.6.0 = Opaque Double: 2.5`,
	} {
		if !strings.Contains(content, expected) {
			t.Errorf("formatted walk missing %q:\n%s", expected, content)
		}
	}
	parsed, err := ParseWalkContent([]byte(content))
	if err != nil {
		t.Fatalf("ParseWalkContent() error = %v", err)
	}
	if len(parsed) != 9 || parsed[3].Type != gosnmp.Uinteger32 ||
		string(parsed[4].Value.([]byte)) != `quoted "path\name"` ||
		parsed[5].Type != gosnmp.BitString || parsed[7].Value != float32(1.25) ||
		parsed[8].Value != 2.5 {
		t.Fatalf("round-trip entries = %+v", parsed)
	}
}
