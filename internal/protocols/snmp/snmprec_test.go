package snmp

import (
	"testing"

	"github.com/gosnmp/gosnmp"
)

// A recording carries the ASN.1 tag as a number, which is what WalkEntry.Type
// already is. These cases pin that the tag survives unchanged rather than
// being rendered to a printed type name and parsed back.
func TestParseWalkContentReadsSnmprec(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		line      string
		wantOID   string
		wantType  gosnmp.Asn1BER
		wantValue any
	}{
		{
			name:      "octet string",
			line:      "1.3.6.1.2.1.1.1.0|4|Cisco IOS Software",
			wantOID:   "1.3.6.1.2.1.1.1.0",
			wantType:  gosnmp.OctetString,
			wantValue: "Cisco IOS Software",
		},
		{
			name:      "integer",
			line:      "1.3.6.1.2.1.2.2.1.8.1|2|1",
			wantOID:   "1.3.6.1.2.1.2.2.1.8.1",
			wantType:  gosnmp.Integer,
			wantValue: 1,
		},
		{
			name:      "gauge32",
			line:      "1.3.6.1.2.1.2.2.1.5.1|66|1000000000",
			wantOID:   "1.3.6.1.2.1.2.2.1.5.1",
			wantType:  gosnmp.Gauge32,
			wantValue: uint(1000000000),
		},
		{
			name:      "counter32",
			line:      "1.3.6.1.2.1.2.2.1.10.1|65|4294967295",
			wantOID:   "1.3.6.1.2.1.2.2.1.10.1",
			wantType:  gosnmp.Counter32,
			wantValue: uint(4294967295),
		},
		{
			name:      "counter64",
			line:      "1.3.6.1.2.1.31.1.1.1.6.1|70|18446744073709551615",
			wantOID:   "1.3.6.1.2.1.31.1.1.1.6.1",
			wantType:  gosnmp.Counter64,
			wantValue: uint64(18446744073709551615),
		},
		{
			name:      "timeticks",
			line:      "1.3.6.1.2.1.1.3.0|67|218726300",
			wantOID:   "1.3.6.1.2.1.1.3.0",
			wantType:  gosnmp.TimeTicks,
			wantValue: uint32(218726300),
		},
		{
			name:      "object identifier keeps no leading dot",
			line:      "1.3.6.1.2.1.1.2.0|6|1.3.6.1.4.1.9.1.516",
			wantOID:   "1.3.6.1.2.1.1.2.0",
			wantType:  gosnmp.ObjectIdentifier,
			wantValue: "1.3.6.1.4.1.9.1.516",
		},
		{
			name:      "ip address",
			line:      "1.3.6.1.2.1.4.20.1.1.10.1.1.1|64|10.1.1.1",
			wantOID:   "1.3.6.1.2.1.4.20.1.1.10.1.1.1",
			wantType:  gosnmp.IPAddress,
			wantValue: "10.1.1.1",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			entries, err := ParseWalkContent([]byte(testCase.line + "\n"))
			if err != nil {
				t.Fatalf("ParseWalkContent: %v", err)
			}
			if len(entries) != 1 {
				t.Fatalf("got %d entries, want 1", len(entries))
			}
			entry := entries[0]
			if entry.OID != testCase.wantOID {
				t.Errorf("OID = %q, want %q", entry.OID, testCase.wantOID)
			}
			if entry.Type != testCase.wantType {
				t.Errorf("Type = %v, want %v", entry.Type, testCase.wantType)
			}
			if entry.Value != testCase.wantValue {
				t.Errorf("Value = %#v, want %#v", entry.Value, testCase.wantValue)
			}
		})
	}
}

// The `x` tag suffix marks a hex-encoded value, which is how a recording
// carries octets that are not safe to print. Decoding must yield the exact
// bytes, because replay re-encodes them onto the wire.
func TestParseWalkContentDecodesHexTaggedOctets(t *testing.T) {
	t.Parallel()

	entries, err := ParseWalkContent([]byte("1.3.6.1.2.1.2.2.1.6.1|4x|0011223344ff\n"))
	if err != nil {
		t.Fatalf("ParseWalkContent: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("got %d entries, want 1", len(entries))
	}
	if entries[0].Type != gosnmp.OctetString {
		t.Errorf("Type = %v, want OctetString", entries[0].Type)
	}
	got, ok := entries[0].Value.([]byte)
	if !ok {
		t.Fatalf("Value is %T, want []byte", entries[0].Value)
	}
	want := []byte{0x00, 0x11, 0x22, 0x33, 0x44, 0xff}
	if string(got) != string(want) {
		t.Errorf("Value = % x, want % x", got, want)
	}
}

// An IpAddress is the one hex form that is neither raw octets nor a number:
// four bytes that must read back as a dotted quad, not as 167772161.
func TestParseWalkContentDecodesHexTaggedAddress(t *testing.T) {
	t.Parallel()

	entries, err := ParseWalkContent([]byte("1.3.6.1.2.1.4.20.1.1.10.1.1.1|64x|0a010101\n"))
	if err != nil {
		t.Fatalf("ParseWalkContent: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("got %d entries, want 1", len(entries))
	}
	if entries[0].Type != gosnmp.IPAddress {
		t.Errorf("Type = %v, want IPAddress", entries[0].Type)
	}
	if entries[0].Value != "10.1.1.1" {
		t.Errorf("Value = %#v, want \"10.1.1.1\"", entries[0].Value)
	}
}

// A recording and a net-snmp walk must not be told apart by file extension —
// the caller may have either under any name. Detection reads the content.
func TestParseWalkContentStillReadsNetSNMP(t *testing.T) {
	t.Parallel()

	entries, err := ParseWalkContent([]byte(
		".1.3.6.1.2.1.1.1.0 = STRING: Cisco IOS Software\n" +
			".1.3.6.1.2.1.1.3.0 = Timeticks: (218726300) 25 days, 7:34:23.00\n",
	))
	if err != nil {
		t.Fatalf("ParseWalkContent: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("got %d entries, want 2", len(entries))
	}
	if entries[0].Type != gosnmp.OctetString || entries[1].Type != gosnmp.TimeTicks {
		t.Errorf("types = %v, %v; want OctetString, TimeTicks", entries[0].Type, entries[1].Type)
	}
}

// A recording's leading comment lines must not decide the format for it.
func TestParseWalkContentSniffsPastComments(t *testing.T) {
	t.Parallel()

	entries, err := ParseWalkContent([]byte(
		"#CISCO BAD TEMP\n\n1.3.6.1.2.1.1.1.0|4|Cisco IOS Software\n",
	))
	if err != nil {
		t.Fatalf("ParseWalkContent: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("got %d entries, want 1", len(entries))
	}
	if entries[0].Type != gosnmp.OctetString {
		t.Errorf("Type = %v, want OctetString", entries[0].Type)
	}
}
