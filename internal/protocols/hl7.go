package protocols

import (
	"bytes"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/gopacket/gopacket/layers"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
)

// TCPPortHL7 is the registered port for HL7 over MLLP.
const TCPPortHL7 = 2575

// MLLP frames a message as <VT> message <FS><CR> (HL7 v2.5.1 Appendix C).
const (
	mllpStartBlock     = 0x0B
	mllpEndBlock       = 0x1C
	mllpCarriageReturn = 0x0D
	mllpFramingLen     = 3 // start block, end block and carriage return

	// hl7MaxMessage bounds a message the service reads. An ADT event or an
	// order is a few kilobytes; a longer message is skipped unanswered.
	hl7MaxMessage = 64 << 10
)

// Defaults for a message whose header does not say, and the acknowledgement
// codes of original mode (HL7 v2.5.1 §2.9.2).
const (
	hl7SegmentSeparator = "\r"
	hl7DefaultEncoding  = `^~\&`
	hl7DefaultProcessID = "P"
	hl7DefaultVersion   = "2.5"
	hl7Accept           = "AA"
	hl7Reject           = "AR"
	hl7TimestampFormat  = "20060102150405-0700"
)

// The MSH fields the ACK reads (HL7 v2.5.1 §2.14.9).
const (
	mshSendingApplication   = 3
	mshSendingFacility      = 4
	mshReceivingApplication = 5
	mshReceivingFacility    = 6
	mshMessageType          = 9
	mshControlID            = 10
	mshProcessingID         = 11
	mshVersionID            = 12
)

// handleHL7 serves 2575 on a device with HL7 enabled: every framed message is
// acknowledged, which is how an interface engine and the clinical systems it
// feeds check that the other end is listening. Every other device keeps the
// closed-port RST.
func (h *TCPHandler) handleHL7(
	pkt *Packet,
	ipLayer *layers.IPv4,
	tcp *layers.TCP,
	devices []*config.Device,
) {
	device := h.serviceDevice(devices, ipLayer.DstIP, func(device *config.Device) bool {
		return device.HL7Config != nil && device.HL7Config.Enabled
	})
	h.serveStream(pkt, ipLayer, tcp, devices, device, h.hl7, func(stream []byte) ([]byte, int, bool) {
		messages, consumed := mllpMessages(stream)
		var answer []byte
		for _, message := range messages {
			answer = append(answer, mllpFrame(hl7Acknowledgement(
				message, device.Name, "NIAC"+strconv.FormatUint(h.hl7Messages.Add(1), 10), time.Now()))...)
		}

		return answer, consumed, false
	}, "HL7")
}

// mllpMessages returns the complete messages at the front of stream and how
// many bytes they took. Bytes outside a block are ignored, as an MLLP
// receiver ignores them, and a block that grows past hl7MaxMessage without
// ending is dropped, so a port scanner's probe or an oversized result costs
// at most one message of memory.
func mllpMessages(stream []byte) ([][]byte, int) {
	var messages [][]byte
	consumed := 0
	for {
		start := bytes.IndexByte(stream[consumed:], mllpStartBlock)
		if start < 0 {
			return messages, len(stream)
		}
		consumed += start
		block := stream[consumed+1:]
		end := bytes.IndexByte(block, mllpEndBlock)
		switch {
		case end < 0 && len(block) > hl7MaxMessage:
			return messages, len(stream)
		case end < 0:
			return messages, consumed
		}
		messages = append(messages, block[:end])
		// The carriage return after the end block, like any byte outside a
		// block, is skipped by the next search.
		consumed += 1 + end + 1
	}
}

func mllpFrame(message string) []byte {
	framed := make([]byte, 0, len(message)+mllpFramingLen)
	framed = append(framed, mllpStartBlock)
	framed = append(framed, message...)

	return append(framed, mllpEndBlock, mllpCarriageReturn)
}

// hl7Acknowledgement is the original-mode ACK to message: AA for a message
// with a header and a control ID, AR for anything else. The header is the
// inbound one turned round, so the sender sees its own application named as
// the receiver; receiver names this end when the sender did not.
func hl7Acknowledgement(message []byte, receiver, controlID string, at time.Time) string {
	header, ok := hl7Header(message)
	code := hl7Accept
	if !ok || header.field(mshControlID) == "" {
		code = hl7Reject
	}
	sendingApplication := header.field(mshReceivingApplication)
	if sendingApplication == "" {
		sendingApplication = receiver
	}
	fields := []string{
		"MSH", header.encoding,
		sendingApplication, header.field(mshReceivingFacility),
		header.field(mshSendingApplication), header.field(mshSendingFacility),
		at.UTC().Format(hl7TimestampFormat), "", header.acknowledgementType(), controlID,
		header.fieldOr(mshProcessingID, hl7DefaultProcessID), header.fieldOr(mshVersionID, hl7DefaultVersion),
	}
	separator := string(header.separator)

	return strings.Join(fields, separator) + hl7SegmentSeparator +
		strings.Join([]string{"MSA", code, header.field(mshControlID)}, separator) + hl7SegmentSeparator
}

// hl7MessageHeader is a message's MSH segment split into fields; fields[n-1]
// is MSH-n for every n above 1, because MSH-1 is the separator itself.
type hl7MessageHeader struct {
	separator byte
	encoding  string
	fields    []string
}

// hl7Header reads message's MSH segment. ok is false when the message does
// not start with one; the defaults then answer it.
func hl7Header(message []byte) (hl7MessageHeader, bool) {
	header := hl7MessageHeader{separator: '|', encoding: hl7DefaultEncoding}
	segment, _, _ := bytes.Cut(message, []byte(hl7SegmentSeparator))
	segment, _, _ = bytes.Cut(segment, []byte("\n"))
	if len(segment) < len("MSH|^~\\&") || !bytes.HasPrefix(segment, []byte("MSH")) ||
		!unicode.IsPunct(rune(segment[3])) && !unicode.IsSymbol(rune(segment[3])) {
		return header, false
	}
	fields := strings.Split(hl7Printable(segment), string(segment[3]))
	if fields[1] == "" {
		return header, false
	}
	header.separator, header.encoding, header.fields = segment[3], fields[1], fields

	return header, true
}

func (m hl7MessageHeader) field(n int) string {
	return m.fieldOr(n, "")
}

func (m hl7MessageHeader) fieldOr(n int, fallback string) string {
	if n-1 < len(m.fields) && m.fields[n-1] != "" {
		return m.fields[n-1]
	}

	return fallback
}

// acknowledgementType is MSH-9 of the ACK: ACK^<trigger>^ACK, naming the
// trigger event of the message acknowledged, or plain ACK when it has none.
func (m hl7MessageHeader) acknowledgementType() string {
	component := string(m.encoding[0])
	trigger := strings.Split(m.field(mshMessageType), component)
	if len(trigger) < 2 || trigger[1] == "" {
		return "ACK"
	}

	return strings.Join([]string{"ACK", trigger[1], "ACK"}, component)
}

// hl7Printable drops the control characters a header field could carry, so
// nothing echoed into the ACK can break its segments or its framing.
func hl7Printable(segment []byte) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, string(segment))
}
