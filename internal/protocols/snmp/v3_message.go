package snmp

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/gosnmp/gosnmp"
)

const (
	berLongFormFlag = 0x80 // a length octet with this bit set counts the length octets that follow
	berMaxLenOctets = 4    // longer than any UDP datagram needs
)

var errBERTruncated = errors.New("truncated BER element")

// v3Message is the cleartext part of an SNMPv3 datagram (RFC 3412 §6, RFC 3414
// §2.4): what the engine needs to choose a usmStats Report before any key is
// applied. gosnmp cannot parse an authenticated header without the sender's
// keys, and a request it cannot authenticate is exactly the one that needs a
// Report, so the engine reads these fields itself.
type v3Message struct {
	msgID       uint32
	maxSize     uint32
	flags       gosnmp.SnmpV3MsgFlags
	engineID    string
	userName    string
	authParams  []byte // msgAuthenticationParameters, aliasing raw
	contextName string // zero when the scoped PDU is encrypted
	requestID   uint32 // zero when the scoped PDU is encrypted
	raw         []byte // the engine's own copy of the datagram
}

// parseV3Message reads the header and USM security parameters of an SNMPv3
// datagram and, when the scoped PDU is plaintext, its context name and
// request-id. It works on a copy, so the caller's buffer is untouched when the
// digest check zeroes the authentication parameters.
func parseV3Message(req []byte) (*v3Message, error) {
	m := &v3Message{raw: bytes.Clone(req)}

	outer := berReader{b: m.raw}
	msg := outer.next(byte(gosnmp.Sequence))
	msg.next(byte(gosnmp.Integer)) // msgVersion; the caller routed on it

	hdr := msg.next(byte(gosnmp.Sequence))
	m.msgID = hdr.uint32()
	m.maxSize = hdr.uint32()
	flags := hdr.next(byte(gosnmp.OctetString))
	model := hdr.uint32()

	sec := msg.next(byte(gosnmp.OctetString))
	usm := sec.next(byte(gosnmp.Sequence))
	m.engineID = string(usm.next(byte(gosnmp.OctetString)).b)
	usm.next(byte(gosnmp.Integer)) // msgAuthoritativeEngineBoots
	usm.next(byte(gosnmp.Integer)) // msgAuthoritativeEngineTime
	m.userName = string(usm.next(byte(gosnmp.OctetString)).b)
	m.authParams = usm.next(byte(gosnmp.OctetString)).b

	for _, r := range []*berReader{&outer, &msg, &hdr, &sec, &usm} {
		if r.err != nil {
			return nil, fmt.Errorf("snmpv3: parse header: %w", r.err)
		}
	}
	if len(flags.b) != 1 {
		return nil, fmt.Errorf("snmpv3: parse header: msgFlags is %d octets, want 1", len(flags.b))
	}
	if model != uint32(gosnmp.UserSecurityModel) {
		return nil, fmt.Errorf("snmpv3: security model %d is not USM", model)
	}
	m.flags = gosnmp.SnmpV3MsgFlags(flags.b[0])

	if m.flags&gosnmp.AuthPriv != gosnmp.AuthPriv {
		scoped := msg.next(byte(gosnmp.Sequence))
		scoped.next(byte(gosnmp.OctetString)) // contextEngineID
		contextName := scoped.next(byte(gosnmp.OctetString))
		pdu := scoped.next(0) // any PDU type
		requestID := pdu.uint32()
		if pdu.err == nil && scoped.err == nil && msg.err == nil && contextName.err == nil {
			m.contextName = string(contextName.b)
			m.requestID = requestID
		}
	}

	return m, nil
}

// reportTo is the request a Report about m answers: the message and request
// identifiers it echoes, and the context it names.
func (m *v3Message) reportTo() *gosnmp.SnmpPacket {
	return &gosnmp.SnmpPacket{
		MsgID:       m.msgID,
		MsgMaxSize:  m.maxSize,
		RequestID:   m.requestID,
		ContextName: m.contextName,
	}
}

// berReader walks a run of BER TLVs. The first error sticks, so a parse reads
// straight through and checks once.
type berReader struct {
	b   []byte
	err error
}

// next consumes one element and returns a reader over its value. A non-zero
// tag must match; zero accepts any tag (the scoped PDU's type varies).
func (r *berReader) next(tag byte) berReader {
	if r.err != nil {
		return berReader{err: r.err}
	}
	const minTLV = 2
	if len(r.b) < minTLV {
		r.err = errBERTruncated
		return berReader{err: r.err}
	}
	if tag != 0 && r.b[0] != tag {
		r.err = fmt.Errorf("BER tag 0x%02x, want 0x%02x", r.b[0], tag)
		return berReader{err: r.err}
	}

	length, header := int(r.b[1]), minTLV
	if r.b[1]&berLongFormFlag != 0 {
		octets := int(r.b[1] &^ berLongFormFlag)
		if octets == 0 || octets > berMaxLenOctets || len(r.b) < minTLV+octets {
			r.err = errBERTruncated
			return berReader{err: r.err}
		}
		length = 0
		for _, o := range r.b[minTLV : minTLV+octets] {
			length = length<<bitsPerOctet | int(o)
		}
		header += octets
	}
	if length > len(r.b)-header {
		r.err = errBERTruncated
		return berReader{err: r.err}
	}

	value := r.b[header : header+length : header+length]
	r.b = r.b[header+length:]
	return berReader{b: value}
}

// uint32 consumes an INTEGER that SNMP constrains to 0..2^31-1.
func (r *berReader) uint32() uint32 {
	v := r.next(byte(gosnmp.Integer))
	const maxOctets = 5 // 2^31-1 plus a leading zero octet
	if v.err == nil && (len(v.b) == 0 || len(v.b) > maxOctets) {
		r.err = fmt.Errorf("BER INTEGER of %d octets", len(v.b))
	}
	if r.err != nil {
		return 0
	}
	var n uint32
	for _, o := range v.b {
		n = n<<bitsPerOctet | uint32(o)
	}
	return n
}
