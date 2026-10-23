package snmp

import (
	"testing"

	"github.com/gosnmp/gosnmp"
)

func TestWalkMultilineStrings(t *testing.T) {
	tests := []struct{ name, input, want string }{
		{"paired quotes", `.1.3.6 = STRING: ""quoted""`, `"quoted"`},
		{
			"capture",
			".1.3.6 = STRING: \"Cisco IOS\nTechnical Support: https://example.test\nCompiled Wed\"\n",
			"Cisco IOS\nTechnical Support: https://example.test\nCompiled Wed",
		},
		{
			"payload",
			".1.3.6 = STRING: \"start  \n  # payload\n\nembedded \"quotes\" stay\nend\"\n",
			"start  \n  # payload\n\nembedded \"quotes\" stay\nend",
		},
		{"escaped quote", ".1.3.6 = STRING: \"start\\\"\nend\"\n", "start\\\"\nend"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entries, err := ParseWalkContent([]byte(tt.input))
			if err != nil {
				t.Fatal(err)
			}
			if len(entries) != 1 {
				t.Fatalf("entries = %v", entries)
			}
			if entries[0].Value != tt.want {
				t.Fatalf("value = %q, want %q", entries[0].Value, tt.want)
			}
		})
	}
}

func TestWalkUnterminatedStringDoesNotConsumeRecord(t *testing.T) {
	for _, input := range []string{
		".1.3.6 = STRING: \"incomplete\n.1.3.7 = INTEGER: 7\n",
		".1.3.6 = STRING: \"incomplete\ncontinuation without end\n.1.3.7 = INTEGER: 7\n",
	} {
		entries, err := ParseWalkContent([]byte(input))
		if err != nil {
			t.Fatal(err)
		}
		if len(entries) != 1 {
			t.Fatalf("entries = %v", entries)
		}
		if entries[0].OID != ".1.3.7" {
			t.Fatalf("OID = %s", entries[0].OID)
		}
	}
	entries, err := ParseWalkContent([]byte(".1.3.6 = STRING: \"incomplete\npayload\n"))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("entries = %v", entries)
	}
}

func TestWalkBareNull(t *testing.T) {
	entries, err := ParseWalkContent([]byte(".1.3.6 = NULL\n.1.3.7 = NULL garbage\n"))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Type != gosnmp.Null || entries[0].Value != nil {
		t.Fatalf("entries = %v", entries)
	}
	packet := gosnmp.SnmpPacket{
		Version:   gosnmp.Version2c,
		Community: "test",
		PDUType:   gosnmp.GetResponse,
		Variables: []gosnmp.SnmpPDU{
			{Name: entries[0].OID, Type: entries[0].Type, Value: entries[0].Value},
		},
	}
	if _, marshalErr := packet.MarshalMsg(); marshalErr != nil {
		t.Fatal(marshalErr)
	}
}

func TestWalkUnterminatedStringPreservesSymbolicBoundary(t *testing.T) {
	entries, err := ParseWalkContent(
		[]byte(".1.3.6 = STRING: \"incomplete\nSNMPv2-MIB::sysDescr.0 = STRING: \"next\"\n"),
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].OID != ".1.3.6.1.2.1.1.1.0" || entries[0].Value != "next" {
		t.Fatalf("entries = %v", entries)
	}
}

func TestWalkBareEndOfMIBMarker(t *testing.T) {
	for _, tc := range []struct {
		line string
		want bool
	}{
		{"End of MIB", true},
		{"End of MIB garbage", false},
		{"prefix End of MIB", false},
		{`.1.3.6 = STRING: "End of MIB"`, false},
	} {
		if got := isEndOfMIBLine(tc.line); got != tc.want {
			t.Errorf("isEndOfMIBLine(%q) = %v", tc.line, got)
		}
	}
}

func TestWalkClosingQuoteEscapeParity(t *testing.T) {
	for _, tc := range []struct {
		value string
		want  bool
	}{
		{`end"`, true},
		{`end\"`, false},
		{`end\\"`, true},
		{`end\\\"`, false},
		{`end\\\\"`, true},
		{`no quote`, false},
	} {
		if got := hasClosingWalkQuote(tc.value); got != tc.want {
			t.Errorf("hasClosingWalkQuote(%q) = %v", tc.value, got)
		}
	}
}
