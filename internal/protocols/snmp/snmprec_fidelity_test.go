package snmp

import (
	"reflect"
	"testing"

	"github.com/gosnmp/gosnmp"
)

func TestRecordingPreservesTypedValues(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name, tag, value string
		wantType         gosnmp.Asn1BER
		wantValue        any
	}{
		{"quoted text", "4", `"quoted"`, gosnmp.OctetString, `"quoted"`},
		{"space padded text", "4", "  padded  ", gosnmp.OctetString, "  padded  "},
		{"hex opaque", "68x", "0001ff", gosnmp.Opaque, []byte{0, 1, 255}},
		{"negative signed hex", "2x", "ff", gosnmp.Integer, -1},
		{"minimum signed hex", "2x", "80000000", gosnmp.Integer, -2147483648},
		{"positive signed hex", "2x", "00ff", gosnmp.Integer, 255},
		{"escaped bytes", "4e", `\x00\xff\n\t\\\"\'\a\b\f\r\v`, gosnmp.OctetString, []byte{0, 255, '\n', '\t', '\\', '"', '\'', '\a', '\b', '\f', '\r', '\v'}},
		{"escaped IP address", "64e", `\xc0\xa8\x00\x01`, gosnmp.IPAddress, "192.168.0.1"},
		{"escaped opaque", "68e", `\x00\xff`, gosnmp.Opaque, []byte{0, 255}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			entries, err := ParseWalkContent([]byte("1.3.6.1.4.1.99999.1|" + tc.tag + "|" + tc.value + "\n"))
			if err != nil {
				t.Fatal(err)
			}
			if len(entries) != 1 {
				t.Fatalf("got %d entries, want 1", len(entries))
			}
			got := entries[0]
			if got.Type != tc.wantType || !reflect.DeepEqual(got.Value, tc.wantValue) {
				t.Errorf("got (%v, %#v), want (%v, %#v)", got.Type, got.Value, tc.wantType, tc.wantValue)
			}
		})
	}
}

func TestRecordingRejectsInvalidBindings(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct{ name, line string }{
		{"invalid OID arc", "1.3.6.bad.1|4|text"},
		{"empty OID arc", "1.3..6.1|4|text"},
		{"overflow OID arc", "1.3.4294967296.1|4|text"},
		{"invalid root OID arc", "3.1.1|4|text"},
		{"invalid second OID arc", "1.40.1|4|text"},
		{"invalid OID value", "1.3.6.1.4.1.99999.1|6|1.3.bad.1"},
		{"invalid IP", "1.3.6.1.4.1.99999.1|64|999.1.1.1"},
		{"short IP bytes", "1.3.6.1.4.1.99999.1|64x|010203"},
		{"integer overflow", "1.3.6.1.4.1.99999.1|2|2147483648"},
		{"counter overflow", "1.3.6.1.4.1.99999.1|65|4294967296"},
		{"integer units", "1.3.6.1.4.1.99999.1|2|1 seconds"},
		{"wide integer bytes", "1.3.6.1.4.1.99999.1|2x|0100000000"},
		{"wide counter bytes", "1.3.6.1.4.1.99999.1|70x|010000000000000000"},
		{"empty integer bytes", "1.3.6.1.4.1.99999.1|2x|"},
		{"repeated encoding suffix", "1.3.6.1.4.1.99999.1|4ex|00"},
		{"unknown escape", `1.3.6.1.4.1.99999.1|4e|\q`},
		{"truncated hex escape", `1.3.6.1.4.1.99999.1|4e|\xf`},
		{"truncated escape", "1.3.6.1.4.1.99999.1|4e|\\"},
		{"null with data", "1.3.6.1.4.1.99999.1|5|data"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if entry, err := parseSnmprecLine(tc.line); err == nil {
				t.Errorf("accepted invalid binding: %#v", entry)
			}
		})
	}
}

func TestRecordingUnsupportedFirstTagDoesNotHideFollowingBindings(t *testing.T) {
	t.Parallel()
	entries, err := ParseWalkContent([]byte(
		"1.3.6.1.4.1.99999.1|4:writecache|ignored\n" +
			"1.3.6.1.4.1.99999.2|4|retained\n",
	))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Value != "retained" {
		t.Fatalf("got %#v, want retained binding", entries)
	}
}

func TestWalkReadsNamedIntegerEnumerations(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		value string
		want  int
	}{{"up(1)", 1}, {"down(2)", 2}, {"other(-1)", -1}, {"ethernetCsmacd(6) interfaces", 6}} {
		t.Run(tc.value, func(t *testing.T) {
			t.Parallel()
			entry, err := parseWalkLine(".1.3.6.1.2.1.2.2.1.8.1 = INTEGER: " + tc.value)
			if err != nil {
				t.Fatal(err)
			}
			if entry.Type != gosnmp.Integer || entry.Value != tc.want {
				t.Fatalf("got %#v, want %d", entry, tc.want)
			}
		})
	}
}

func TestRecordingTrimsOnlyOIDWhitespace(t *testing.T) {
	t.Parallel()
	entries, err := ParseWalkContent([]byte(" \t.1.3.6.1.4.1.99999.1\t |4| \"quoted\" \n"))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("got %d entries, want 1", len(entries))
	}
	if entries[0].OID != "1.3.6.1.4.1.99999.1" || entries[0].Value != ` "quoted" ` {
		t.Fatalf("got %#v; want trimmed OID and unchanged value bytes", entries[0])
	}
}
