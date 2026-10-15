package protocols

import (
	"bytes"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/gopacket/gopacket/layers"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
)

// hl7Admit is an ADT^A01 the way an interface engine sends one.
const hl7Admit = "MSH|^~\\&|EPIC|MAINHOSP|LAB|MAINLAB|20261015050000||ADT^A01^ADT_A01|MSG00042|P|2.5.1\r" +
	"EVN|A01|20261015050000\rPID|1||12345^^^MRN||DOE^JANE\r"

func TestMLLPMessages(t *testing.T) {
	t.Parallel()

	frame := func(message string) string { return "\x0b" + message + "\x1c\r" }
	tests := []struct {
		name     string
		stream   string
		want     []string
		consumed int
	}{
		{"one message", frame("A"), []string{"A"}, 4},
		{"two in one segment", frame("A") + frame("B"), []string{"A", "B"}, 8},
		{"split before the end block", "\x0bMSH|partial", nil, 0},
		{"end block waiting for its CR", "\x0bA\x1c", []string{"A"}, 3},
		{"bytes outside a block are skipped", "GET / HTTP/1.0\r\n\r\n" + frame("A"), []string{"A"}, 22},
		{"no block at all", "GET / HTTP/1.0\r\n\r\n", nil, 18},
		{"oversized block is dropped", "\x0b" + strings.Repeat("x", hl7MaxMessage+1), nil, hl7MaxMessage + 2},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			messages, consumed := mllpMessages([]byte(test.stream))
			got := make([]string, 0, len(messages))
			for _, message := range messages {
				got = append(got, string(message))
			}
			if strings.Join(got, ",") != strings.Join(test.want, ",") || consumed != test.consumed {
				t.Errorf("messages %q consumed %d, want %q consumed %d", got, consumed, test.want, test.consumed)
			}
		})
	}
}

func TestHL7Acknowledgement(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		message string
		want    string
	}{
		{
			name:    "header turned round, trigger and version echoed",
			message: hl7Admit,
			want: "MSH|^~\\&|LAB|MAINLAB|EPIC|MAINHOSP|20261015050001+0000||ACK^A01^ACK|NIAC7|P|2.5.1\r" +
				"MSA|AA|MSG00042\r",
		},
		{
			name:    "the sender's separators are kept",
			message: "MSH#$~\\&#EPIC#MAINHOSP#LAB#MAINLAB#20261015##ORM$O01#ORD9#T#2.3\r",
			want:    "MSH#$~\\&#LAB#MAINLAB#EPIC#MAINHOSP#20261015050001+0000##ACK$O01$ACK#NIAC7#T#2.3\rMSA#AA#ORD9\r",
		},
		{
			name:    "no receiving application names this end, no trigger is plain ACK",
			message: "MSH|^~\\&|EPIC|MAINHOSP||||||MSG1\r",
			want:    "MSH|^~\\&|MED-EMR01||EPIC|MAINHOSP|20261015050001+0000||ACK|NIAC7|P|2.5\rMSA|AA|MSG1\r",
		},
		{
			name:    "no control ID is rejected",
			message: "MSH|^~\\&|EPIC|MAINHOSP|LAB|MAINLAB|20261015||ADT^A01||P|2.5\r",
			want:    "MSH|^~\\&|LAB|MAINLAB|EPIC|MAINHOSP|20261015050001+0000||ACK^A01^ACK|NIAC7|P|2.5\rMSA|AR|\r",
		},
		{
			name:    "not HL7 is rejected",
			message: "hello",
			want:    "MSH|^~\\&|MED-EMR01||||20261015050001+0000||ACK|NIAC7|P|2.5\rMSA|AR|\r",
		},
		{
			name:    "an empty encoding field is rejected",
			message: "MSH||EPIC\r",
			want:    "MSH|^~\\&|MED-EMR01||||20261015050001+0000||ACK|NIAC7|P|2.5\rMSA|AR|\r",
		},
		{
			name:    "control characters echoed from the header are dropped",
			message: "MSH|^~\\&|EP\x0bIC|||||||ID\x1c1|P|2.5\r",
			want:    "MSH|^~\\&|MED-EMR01||EPIC||20261015050001+0000||ACK|NIAC7|P|2.5\rMSA|AA|ID1\r",
		},
	}
	at := time.Date(2026, 10, 15, 5, 0, 1, 0, time.UTC)
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if got := hl7Acknowledgement([]byte(test.message), "MED-EMR01", "NIAC7", at); got != test.want {
				t.Errorf("ACK\n got %q\nwant %q", got, test.want)
			}
		})
	}
}

func hl7TestNode() *config.Device {
	return &config.Device{
		Name:        "MED-EMR01",
		MACAddress:  net.HardwareAddr{0x00, 0x50, 0x56, 0x00, 0x00, 0x05},
		IPAddresses: []net.IP{net.ParseIP("10.20.30.43")},
		HL7Config:   &config.HL7Config{Enabled: true},
	}
}

// A message split across segments is acknowledged once whole, framed in MLLP,
// and a second message on the same connection gets its own control ID.
func TestTCPHandlerServesHL7(t *testing.T) {
	conversation := newServiceConversation(t, hl7TestNode(), TCPPortHL7)
	if reply := conversation.reply(&layers.TCP{SYN: true, Seq: 100}); reply == nil ||
		!reply.SYN || !reply.ACK || reply.Ack != 101 {
		t.Fatalf("SYN answered with %+v, want a SYN-ACK acknowledging 101", reply)
	}
	seq := uint32(101)
	segment := func(payload string, fin bool) *layers.TCP {
		sent := &layers.TCP{ACK: true, PSH: true, FIN: fin, Seq: seq, Ack: 1001, Payload: []byte(payload)}
		seq += uint32(len(payload))
		return sent
	}
	framed := "\x0b" + hl7Admit + "\x1c\r"
	if reply := conversation.reply(segment(framed[:40], false)); reply == nil || len(reply.Payload) != 0 ||
		reply.Ack != seq {
		t.Fatalf("first half answered with %+v, want a bare ACK of %d", reply, seq)
	}
	first := conversation.reply(segment(framed[40:], false))
	if first == nil || first.Seq != 1001 || first.Ack != seq {
		t.Fatalf("whole message answered with %+v", first)
	}
	ack := hl7Payload(t, first.Payload)
	if !strings.Contains(ack, "|ACK^A01^ACK|NIAC1|P|2.5.1\rMSA|AA|MSG00042\r") ||
		!strings.HasPrefix(ack, "MSH|^~\\&|LAB|MAINLAB|EPIC|MAINHOSP|") {
		t.Errorf("ACK %q does not acknowledge MSG00042", ack)
	}
	second := conversation.reply(segment(framed, true))
	if second == nil || !second.FIN || second.Ack != seq+1 ||
		!strings.Contains(hl7Payload(t, second.Payload), "|NIAC2|") {
		t.Errorf("second message with FIN answered with %+v", second)
	}
}

func hl7Payload(t *testing.T, payload []byte) string {
	t.Helper()
	message, ok := bytes.CutPrefix(payload, []byte{mllpStartBlock})
	if !ok {
		t.Fatalf("answer % x has no start block", payload)
	}
	message, ok = bytes.CutSuffix(message, []byte{mllpEndBlock, mllpCarriageReturn})
	if !ok {
		t.Fatalf("answer % x has no end block", payload)
	}

	return string(message)
}

// A device without HL7 keeps the closed-port RST, as does one whose HL7 is
// switched off, and never answers a message.
func TestTCPHandlerHL7ClosedWithoutHL7(t *testing.T) {
	for _, hl7 := range []*config.HL7Config{nil, {}} {
		device := hl7TestNode()
		device.HL7Config = hl7
		if reply := serviceReply(t, device, TCPPortHL7, &layers.TCP{SYN: true, Seq: 500}); reply == nil || !reply.RST {
			t.Errorf("HL7 %+v: SYN answered with %+v, want RST", hl7, reply)
		}
		data := &layers.TCP{ACK: true, Seq: 501, Ack: 1001, Payload: []byte("\x0b" + hl7Admit + "\x1c\r")}
		if reply := serviceReply(t, device, TCPPortHL7, data); reply != nil {
			t.Errorf("HL7 %+v: message answered with %+v", hl7, reply)
		}
	}
}
