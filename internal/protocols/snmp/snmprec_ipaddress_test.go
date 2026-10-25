package snmp

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/gosnmp/gosnmp"
)

type recordedIPAddressCase struct {
	name, tag, payload, want string
}

func recordingIPAddressCases() []recordedIPAddressCase {
	return []recordedIPAddressCase{
		{"literal ASCII", "64", "eEij", "101.69.105.106"},
		{"literal digits", "64", "1234", "49.50.51.52"},
		{"literal dotted prefix", "64", "1.2.", "49.46.50.46"},
		{"literal binary", "64", "\xc0\xa8\x00\x01", "192.168.0.1"},
		{"literal whitespace", "64", "\x20\x00\xff\t", "32.0.255.9"},
		{"literal UTF8 octets", "64", "\xc3\xa9\xc3\xa9", "195.169.195.169"},
		{"literal zero", "64", "\x00\x00\x00\x00", "0.0.0.0"},
		{"dotted", "64", "192.168.0.1", "192.168.0.1"},
		{"dotted zero", "64", "0.0.0.0", "0.0.0.0"},
		{"dotted broadcast", "64", "255.255.255.255", "255.255.255.255"},
		{"hex", "64x", "c0a80001", "192.168.0.1"},
		{"escaped", "64e", `\xc0\xa8\x00\x01`, "192.168.0.1"},
	}
}

func TestRecordingIPAddressLiterals(t *testing.T) {
	t.Parallel()
	for _, tc := range recordingIPAddressCases() {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			entries, err := ParseWalkContent([]byte("1.3.6.1.4.1.99999.1.0|" + tc.tag + "|" + tc.payload + "\n"))
			if err != nil || len(entries) != 1 {
				t.Fatalf("entries = %#v, error = %v", entries, err)
			}
			if entries[0].Type != gosnmp.IPAddress || entries[0].Value != tc.want {
				t.Fatalf("entry = %#v, want IPAddress %s", entries[0], tc.want)
			}
		})
	}
}

func TestRecordingIPAddressRejectsMalformedPayloads(t *testing.T) {
	t.Parallel()
	for _, payload := range []string{
		"", "abc", "abcde", "\xff\x80\x00", "\xff\x80\x00\x01\x02", "\xc3\xa9\xc3", "ÿÿÿÿ",
		"999.1.1.1", "192.168.01.1", " 192.168.0.1", "192.168.0.1 ", "::1", "::ffff:192.168.0.1",
	} {
		t.Run(payload, func(t *testing.T) {
			t.Parallel()
			if entry, err := parseSnmprecLine("1.3.6.1.4.1.99999.1.0|64|" + payload); err == nil {
				t.Fatalf("accepted invalid IPAddress: %#v", entry)
			}
		})
	}
}

func TestRecordingIPAddressAgentBER(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		payload string
		literal []byte
	}{
		{"eEij", []byte{0x40, 4, 101, 69, 105, 106}},
		{"\xc0\xa8\x00\x01", []byte{0x40, 4, 192, 168, 0, 1}},
	} {
		t.Run(tc.payload, func(t *testing.T) {
			t.Parallel()
			wire := recordedIPAddressAgentResponse(t, tc.payload)
			if !bytes.Contains(wire, tc.literal) {
				t.Fatalf("response %x lacks IPAddress tag/length/octet sequence %x", wire, tc.literal)
			}
		})
	}
}

func recordedIPAddressAgentResponse(t *testing.T, payload string) []byte {
	t.Helper()
	const oid = "1.3.6.1.4.1.99999.1.0"
	path := filepath.Join(t.TempDir(), "address.snmprec")
	if err := os.WriteFile(path, []byte(oid+"|64|"+payload+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	agent := NewAgent(createTestDevice(), 0)
	if err := agent.LoadWalkFile(path); err != nil {
		t.Fatal(err)
	}
	packet := gosnmp.SnmpPacket{
		Version: gosnmp.Version2c, Community: "test", PDUType: gosnmp.GetResponse,
		Variables: agent.ProcessPDU(gosnmp.GetRequest, []gosnmp.SnmpPDU{{Name: oid, Type: gosnmp.Null}}, 0, 0),
	}
	wire, err := packet.MarshalMsg()
	if err != nil {
		t.Fatal(err)
	}
	return wire
}
