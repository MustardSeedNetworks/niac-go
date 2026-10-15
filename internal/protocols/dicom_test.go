package protocols

import (
	"bytes"
	"encoding/binary"
	"net"
	"net/netip"
	"testing"

	"github.com/gopacket/gopacket/layers"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
)

const dicomStorageClass = "1.2.840.10008.5.1.4.1.1.4" // MR Image Storage

// dicomRequestContext is one proposed presentation context: its ID, abstract
// syntax and the transfer syntaxes offered for it.
type dicomRequestContext struct {
	id       byte
	abstract string
	transfer []string
}

// dicomAssociateRequest builds an A-ASSOCIATE-RQ the way echoscu does: both AE
// titles, the DICOM application context, the contexts and user information.
func dicomAssociateRequest(applicationContext string, contexts ...dicomRequestContext) []byte {
	body := binary.BigEndian.AppendUint16(nil, 1)
	body = append(body, 0, 0)
	body = append(body, []byte("ANY-SCP         ECHOSCU         ")...)
	body = append(body, make([]byte, 32)...)
	body = append(body, dicomItem(dicomItemApplicationContext, []byte(applicationContext))...)
	for _, context := range contexts {
		value := append([]byte{context.id, 0, 0, 0}, dicomItem(dicomItemAbstractSyntax, []byte(context.abstract))...)
		for _, transfer := range context.transfer {
			value = append(value, dicomItem(dicomItemTransferSyntax, []byte(transfer))...)
		}
		body = append(body, dicomItem(dicomItemPresentationRQ, value)...)
	}
	body = append(body, dicomItem(dicomItemUserInformation,
		dicomItem(dicomItemMaxLength, binary.BigEndian.AppendUint32(nil, dicomMaxPDULength)))...)

	return dicomPDU(dicomAssociateRQ, body)
}

// dicomEchoRequest is a C-ECHO-RQ on presentation context 1.
func dicomEchoRequest(messageID uint16) []byte {
	var elements []byte
	elements = appendDICOMElement(elements, dicomTagAffectedSOPClass, append([]byte(dicomVerificationClass), 0))
	elements = appendDICOMElement(elements, dicomTagCommandField, binary.LittleEndian.AppendUint16(nil, dicomCEchoRQ))
	elements = appendDICOMElement(elements, dicomTagMessageID, binary.LittleEndian.AppendUint16(nil, messageID))
	elements = appendDICOMElement(elements, dicomTagDataSetType, binary.LittleEndian.AppendUint16(nil, dicomNoDataSet))
	command := appendDICOMElement(
		nil,
		dicomTagGroupLength,
		binary.LittleEndian.AppendUint32(nil, uint32(len(elements))),
	)
	command = append(command, elements...)
	pdv := binary.BigEndian.AppendUint32(nil, uint32(len(command)+2))
	pdv = append(pdv, 1, dicomPDVCommandLast)

	return dicomPDU(dicomPDataTF, append(pdv, command...))
}

// acceptedContexts reads the presentation context results from an
// A-ASSOCIATE-AC: context ID to result and transfer syntax.
func acceptedContexts(t *testing.T, answer []byte) map[byte][2]string {
	t.Helper()
	if len(answer) < dicomPDUHeaderLen+dicomAssociateLen || answer[0] != dicomAssociateAC {
		t.Fatalf("answer is no A-ASSOCIATE-AC: % x", answer)
	}
	results := map[byte][2]string{}
	items := answer[dicomPDUHeaderLen+dicomAssociateLen:]
	for len(items) > 0 {
		itemType, value, rest, ok := nextDICOMItem(items)
		if !ok {
			t.Fatalf("malformed item in % x", items)
		}
		items = rest
		if itemType != dicomItemPresentationAC {
			continue
		}
		_, transfer, _, ok := nextDICOMItem(value[4:])
		if !ok {
			t.Fatalf("context %d has no transfer syntax", value[0])
		}
		results[value[0]] = [2]string{string(rune('0' + value[2])), string(transfer)}
	}

	return results
}

func TestDICOMAssociationAcceptsVerificationOnly(t *testing.T) {
	t.Parallel()

	request := dicomAssociateRequest(dicomApplicationContext,
		dicomRequestContext{1, dicomVerificationClass, []string{dicomExplicitLittle, dicomImplicitLittle}},
		dicomRequestContext{3, dicomVerificationClass, []string{dicomExplicitLittle}},
		dicomRequestContext{5, dicomVerificationClass, []string{"1.2.840.10008.1.2.4.50"}},
		dicomRequestContext{7, dicomStorageClass, []string{dicomImplicitLittle}},
	)
	answer, consumed, ended := dicomAnswer(request)
	if consumed != len(request) || ended {
		t.Fatalf("consumed %d of %d, ended %v", consumed, len(request), ended)
	}

	if got, want := int(binary.BigEndian.Uint32(answer[2:6])), len(answer)-dicomPDUHeaderLen; got != want {
		t.Fatalf("PDU length = %d, want %d", got, want)
	}
	if !bytes.Equal(answer[dicomPDUHeaderLen+4:dicomPDUHeaderLen+dicomAssociateLen],
		request[dicomPDUHeaderLen+4:dicomPDUHeaderLen+dicomAssociateLen]) {
		t.Error("AE titles and reserved field were not echoed")
	}
	want := map[byte][2]string{
		1: {"0", dicomImplicitLittle},
		3: {"0", dicomExplicitLittle},
		5: {"4", dicomImplicitLittle},
		7: {"3", dicomImplicitLittle},
	}
	got := acceptedContexts(t, answer)
	for id, result := range want {
		if got[id] != result {
			t.Errorf("context %d = %v, want %v", id, got[id], result)
		}
	}
	if !bytes.Contains(answer, []byte(dicomImplementationUID)) {
		t.Error("A-ASSOCIATE-AC carries no implementation class UID")
	}
}

func TestDICOMEchoAndRelease(t *testing.T) {
	t.Parallel()

	answer, _, _ := dicomAnswer(append(dicomEchoRequest(0x1234), dicomPDU(dicomReleaseRQ, make([]byte, 4))...))

	if len(answer) < dicomPDUHeaderLen+dicomPDVHeaderLen || answer[0] != dicomPDataTF {
		t.Fatalf("answer is no P-DATA-TF: % x", answer)
	}
	pduEnd := dicomPDUHeaderLen + int(binary.BigEndian.Uint32(answer[2:6]))
	pdv := answer[dicomPDUHeaderLen:pduEnd]
	if pdv[4] != 1 || pdv[5] != dicomPDVCommandLast {
		t.Errorf("PDV context %d control %#x, want 1 and %#x", pdv[4], pdv[5], dicomPDVCommandLast)
	}
	fields := map[uint16][]byte{}
	for command := pdv[dicomPDVHeaderLen:]; len(command) > 0; {
		length := int(binary.LittleEndian.Uint32(command[4:8]))
		fields[binary.LittleEndian.Uint16(command[2:4])] = command[8 : 8+length]
		command = command[8+length:]
	}
	if got := binary.LittleEndian.Uint32(fields[dicomTagGroupLength]); int(got) != len(pdv)-dicomPDVHeaderLen-12 {
		t.Errorf("command group length = %d, want %d", got, len(pdv)-dicomPDVHeaderLen-12)
	}
	checks := map[uint16]uint16{
		dicomTagCommandField: dicomCEchoRSP, dicomTagMessageIDResponded: 0x1234,
		dicomTagDataSetType: dicomNoDataSet, dicomTagStatus: 0,
	}
	for element, want := range checks {
		if value := fields[element]; len(value) != 2 || binary.LittleEndian.Uint16(value) != want {
			t.Errorf("element (0000,%04x) = % x, want %#04x", element, value, want)
		}
	}
	if string(fields[dicomTagAffectedSOPClass]) != dicomVerificationClass+"\x00" {
		t.Errorf("affected SOP class = %q", fields[dicomTagAffectedSOPClass])
	}
	if release := answer[pduEnd:]; !bytes.Equal(release, []byte{dicomReleaseRP, 0, 0, 0, 0, 4, 0, 0, 0, 0}) {
		t.Errorf("release answer = % x", release)
	}
}

func TestDICOMRefusals(t *testing.T) {
	t.Parallel()

	abort := func(reason byte) []byte { return []byte{dicomAbort, 0, 0, 0, 0, 4, 0, 0, 2, reason} }
	echo := dicomEchoRequest(7)
	notEcho := bytes.Clone(echo)
	// The command field's value sits after the PDU and PDV headers, the group
	// length (12 bytes) and the affected SOP class (8 + 18 bytes).
	binary.LittleEndian.PutUint16(notEcho[dicomPDUHeaderLen+dicomPDVHeaderLen+12+26+8:], 0x0001) // C-STORE-RQ
	tests := []struct {
		name    string
		payload []byte
		want    []byte
	}{
		{
			name:    "foreign application context",
			payload: dicomAssociateRequest("1.2.3.4"),
			want:    []byte{dicomAssociateRJ, 0, 0, 0, 0, 4, 0, 1, 1, 2},
		},
		{name: "short association body", payload: dicomPDU(dicomAssociateRQ, make([]byte, 10)), want: abort(6)},
		{
			name:    "item overruns the PDU",
			payload: dicomPDU(dicomAssociateRQ, append(make([]byte, dicomAssociateLen), 0x10, 0, 0, 40)),
			want:    abort(6),
		},
		{name: "command other than C-ECHO", payload: notEcho, want: abort(6)},
		{name: "unknown PDU type", payload: dicomPDU(0x09, nil), want: abort(1)},
		{name: "HTTP probe, header only", payload: []byte("GET / "), want: abort(1)},
		{name: "TLS probe", payload: []byte{0x16, 0x03, 0x01, 0x00, 0xa5, 0x01}, want: abort(1)},
		{name: "oversized PDU, header only", payload: []byte{1, 0, 0, 1, 0, 1}, want: abort(6)},
		{name: "a PDU only an SCP sends", payload: dicomPDU(dicomAssociateAC, nil), want: abort(2)},
		{name: "abort is not answered", payload: append(abort(0), echo...), want: nil},
		{name: "PDU split across segments", payload: echo[:len(echo)-1], want: nil},
		{name: "data fragment alone", payload: dicomPDU(dicomPDataTF, []byte{0, 0, 0, 3, 1, 0x02, 0xff}), want: nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got, _, _ := dicomAnswer(tt.payload); !bytes.Equal(got, tt.want) {
				t.Errorf("answer = % x, want % x", got, tt.want)
			}
		})
	}
}

func dicomTestNode() *config.Device {
	return &config.Device{
		Name:        "MED-PACS01",
		MACAddress:  net.HardwareAddr{0x00, 0x50, 0x56, 0x00, 0x00, 0x04},
		IPAddresses: []net.IP{net.ParseIP("10.20.30.42")},
		DICOMConfig: &config.DICOMConfig{Enabled: true},
	}
}

func TestTCPHandlerServesDICOM(t *testing.T) {
	node := dicomTestNode()
	for _, port := range []layers.TCPPort{TCPPortDICOM, TCPPortDICOMAlt} {
		if reply := serviceReply(t, node, port, &layers.TCP{SYN: true, Seq: 500}); reply == nil ||
			!reply.SYN || !reply.ACK || reply.Ack != 501 {
			t.Fatalf("port %d: SYN answered with %+v, want a SYN-ACK acknowledging 501", port, reply)
		}
		request := dicomAssociateRequest(dicomApplicationContext,
			dicomRequestContext{1, dicomVerificationClass, []string{dicomImplicitLittle}})
		reply := serviceReply(t, node, port, &layers.TCP{ACK: true, PSH: true, Seq: 501, Ack: 1001, Payload: request})
		if reply == nil || reply.Seq != 1001 || reply.Ack != 501+uint32(len(request)) {
			t.Fatalf("port %d: association answered with %+v", port, reply)
		}
		if got := acceptedContexts(t, reply.Payload)[1]; got != [2]string{"0", dicomImplicitLittle} {
			t.Errorf("port %d: verification context = %v, want accepted", port, got)
		}
	}
}

// A device without DICOM keeps the closed-port RST on both ports, as does one
// whose DICOM is switched off.
func TestTCPHandlerDICOMClosedWithoutDICOM(t *testing.T) {
	for _, dicom := range []*config.DICOMConfig{nil, {}} {
		device := dicomTestNode()
		device.DICOMConfig = dicom
		for _, port := range []layers.TCPPort{TCPPortDICOM, TCPPortDICOMAlt} {
			if reply := serviceReply(t, device, port, &layers.TCP{SYN: true, Seq: 500}); reply == nil || !reply.RST {
				t.Errorf("DICOM %+v port %d: SYN answered with %+v, want RST", dicom, port, reply)
			}
			data := &layers.TCP{ACK: true, Seq: 501, Ack: 1001, Payload: dicomEchoRequest(1)}
			if reply := serviceReply(t, device, port, data); reply != nil {
				t.Errorf("DICOM %+v port %d: data answered with %+v", dicom, port, reply)
			}
		}
	}
}

// dcmtk writes a C-ECHO-RQ as two segments, the PDU and PDV headers and then
// the command; the answer comes once the PDU is whole.
func TestDICOMStreamsReassembleSplitPDUs(t *testing.T) {
	t.Parallel()

	streams := newServiceStreams()
	key := streamKey{}
	streams.open(key, 101)
	echo := dicomEchoRequest(9)
	whole, _, _ := dicomAnswer(echo)

	if answer, inOrder := streams.receive(key, 101, echo[:12], dicomAnswer); !inOrder || answer != nil {
		t.Fatalf("headers alone answered %x (in order %v), want nothing yet", answer, inOrder)
	}
	if answer, inOrder := streams.receive(key, 113, echo[12:], dicomAnswer); !inOrder || !bytes.Equal(answer, whole) {
		t.Fatalf("completed PDU answered % x, want % x", answer, whole)
	}
	if answer, inOrder := streams.receive(key, 113, echo[12:], dicomAnswer); !inOrder || answer != nil {
		t.Errorf("a repeated segment answered % x (in order %v), want a bare acknowledgement", answer, inOrder)
	}
	if _, inOrder := streams.receive(key, 500, echo, dicomAnswer); inOrder {
		t.Error("a segment past a gap was taken in order")
	}
	release := dicomPDU(dicomReleaseRQ, make([]byte, 4))
	if answer, _ := streams.receive(key, 101+uint32(len(echo)), release, dicomAnswer); answer[0] != dicomReleaseRP {
		t.Errorf("release after the gap answered % x", answer)
	}
}

func TestDICOMStreamsBounds(t *testing.T) {
	t.Parallel()

	streams := newServiceStreams()
	for port := range uint16(streamMaxConnections + 1) {
		streams.open(streamKey{client: netip.AddrPortFrom(netip.IPv4Unspecified(), port)}, 0)
	}
	if len(streams.streams) != streamMaxConnections {
		t.Errorf("%d streams held, want at most %d", len(streams.streams), streamMaxConnections)
	}
}

// The C-ECHO split the way dcmtk sends it is answered once whole, after the
// association, on one connection.
func TestTCPHandlerReassemblesDICOM(t *testing.T) {
	conversation := newServiceConversation(t, dicomTestNode(), TCPPortDICOMAlt)
	if reply := conversation.reply(&layers.TCP{SYN: true, Seq: 100}); reply == nil || !reply.SYN {
		t.Fatalf("SYN answered with %+v", reply)
	}
	request := dicomAssociateRequest(dicomApplicationContext,
		dicomRequestContext{1, dicomVerificationClass, []string{dicomImplicitLittle}})
	seq := uint32(101)
	segment := func(payload []byte) *layers.TCP {
		sent := &layers.TCP{ACK: true, PSH: true, Seq: seq, Ack: 1001, Payload: payload}
		seq += uint32(len(payload))
		return sent
	}
	if reply := conversation.reply(segment(request)); reply == nil || len(reply.Payload) == 0 {
		t.Fatalf("association answered with %+v", reply)
	}
	echo := dicomEchoRequest(3)
	if reply := conversation.reply(segment(echo[:12])); reply == nil || len(reply.Payload) != 0 || reply.Ack != seq {
		t.Fatalf("PDU headers answered with %+v, want a bare ACK of %d", reply, seq)
	}
	reply := conversation.reply(segment(echo[12:]))
	if want, _, _ := dicomAnswer(echo); reply == nil || !bytes.Equal(reply.Payload, want) || reply.Ack != seq {
		t.Fatalf("completed C-ECHO answered with %+v, want % x acknowledging %d", reply, want, seq)
	}
}

// An SCU proposing every storage class gets an A-ASSOCIATE-AC longer than one
// segment; it goes out as consecutive segments, every one refusing storage.
func TestTCPHandlerSegmentsLongDICOMAnswers(t *testing.T) {
	contexts := make([]dicomRequestContext, 0, 120)
	for id := 1; id < 240; id += 2 {
		contexts = append(contexts, dicomRequestContext{byte(id), dicomStorageClass, []string{dicomImplicitLittle}})
	}
	request := dicomAssociateRequest(dicomApplicationContext, contexts...)
	want, _, _ := dicomAnswer(request)
	conversation := newServiceConversation(t, dicomTestNode(), TCPPortDICOM)
	conversation.reply(&layers.TCP{SYN: true, Seq: 100})
	first := conversation.reply(&layers.TCP{ACK: true, PSH: true, Seq: 101, Ack: 1001, Payload: request})
	if first == nil || len(first.Payload) != statelessMaxSegment || first.Seq != 1001 {
		t.Fatalf("first segment %+v, want %d bytes at 1001", first, statelessMaxSegment)
	}
	got := bytes.Clone(first.Payload)
	for len(got) < len(want) {
		next := conversation.next()
		if next == nil || next.Seq != 1001+uint32(len(got)) {
			t.Fatalf("after %d of %d bytes the next segment is %+v", len(got), len(want), next)
		}
		got = append(got, next.Payload...)
	}
	if !bytes.Equal(got, want) {
		t.Error("the segments do not add up to the A-ASSOCIATE-AC")
	}
}
