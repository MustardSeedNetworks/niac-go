package protocols

import (
	"encoding/binary"
	"slices"
	"strings"

	"github.com/gopacket/gopacket/layers"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/safeconv"
)

// DICOM upper-layer ports: 104 is the registered port, 11112 the one most
// archives and modalities are configured with in practice.
const (
	TCPPortDICOM    = 104
	TCPPortDICOMAlt = 11112
)

// DICOM upper-layer PDU types (PS3.8 §9.3).
const (
	dicomAssociateRQ = 0x01
	dicomAssociateAC = 0x02
	dicomAssociateRJ = 0x03
	dicomPDataTF     = 0x04
	dicomReleaseRQ   = 0x05
	dicomReleaseRP   = 0x06
	dicomAbort       = 0x07
)

// Variable items of the association PDUs (PS3.8 §9.3.2 and Annex D of PS3.7).
const (
	dicomItemApplicationContext = 0x10
	dicomItemPresentationRQ     = 0x20
	dicomItemPresentationAC     = 0x21
	dicomItemAbstractSyntax     = 0x30
	dicomItemTransferSyntax     = 0x40
	dicomItemUserInformation    = 0x50
	dicomItemMaxLength          = 0x51
	dicomItemImplementationUID  = 0x52
)

const (
	dicomApplicationContext = "1.2.840.10008.3.1.1.1"
	dicomVerificationClass  = "1.2.840.10008.1.1"
	dicomImplicitLittle     = "1.2.840.10008.1.2"
	dicomExplicitLittle     = "1.2.840.10008.1.2.1"

	// dicomImplementationUID is a UUID-derived (2.25) UID, which needs no
	// registered root. The association answer must carry one.
	dicomImplementationUID = "2.25.329800735698586629295641978511506172918"
	dicomMaxPDULength      = 16384
)

// Presentation context results (PS3.8 table 9-18).
const (
	dicomContextAccepted          = 0
	dicomContextAbstractRejected  = 3
	dicomContextTransferRejected  = 4
	dicomRejectPermanent          = 1
	dicomRejectSourceServiceUser  = 1
	dicomRejectContextUnsupported = 2
	dicomAbortSourceProvider      = 2
	dicomAbortUnrecognizedPDU     = 1
	dicomAbortUnexpectedPDU       = 2
	dicomAbortInvalidParameter    = 6
)

// Lengths of the fixed parts of the PDUs and items.
const (
	dicomPDUHeaderLen  = 6  // type, reserved, 32-bit length
	dicomItemHeaderLen = 4  // type, reserved, 16-bit length
	dicomAssociateLen  = 68 // version, reserved, both AE titles, 32 reserved
	dicomPDVHeaderLen  = 6  // 32-bit length, context ID, message control
	dicomPDVLengthLen  = 4  // the PDV item length counts what follows it
	dicomElementLen    = 8  // implicit VR little endian tag and 32-bit length
)

// DIMSE command set elements, all in group 0000 (PS3.7 §E.1).
const (
	dicomTagGroupLength        = 0x0000
	dicomTagAffectedSOPClass   = 0x0002
	dicomTagCommandField       = 0x0100
	dicomTagMessageID          = 0x0110
	dicomTagMessageIDResponded = 0x0120
	dicomTagDataSetType        = 0x0800
	dicomTagStatus             = 0x0900
	dicomCEchoRQ               = 0x0030
	dicomCEchoRSP              = 0x8030
	dicomNoDataSet             = 0x0101
	dicomPDVCommandLast        = 0x03 // message control: command, last fragment
)

// handleDICOM serves 104 and 11112 on a device with DICOM enabled: it accepts
// an association for the Verification SOP class, answers C-ECHO and releases,
// which is how a PACS and the modalities sending to it check each other.
// Every other device keeps the closed-port RST.
func (h *TCPHandler) handleDICOM(
	pkt *Packet,
	ipLayer *layers.IPv4,
	tcp *layers.TCP,
	devices []*config.Device,
) {
	device := h.serviceDevice(devices, ipLayer.DstIP, func(device *config.Device) bool {
		return device.DICOMConfig != nil && device.DICOMConfig.Enabled
	})
	key := newDICOMStreamKey(ipLayer, tcp)
	switch {
	case device == nil:
		if tcp.SYN && !tcp.ACK {
			h.sendRST(pkt, ipLayer, tcp, devices)
		}
	case tcp.RST:
		h.dicom.close(key)
	case tcp.SYN && !tcp.ACK:
		h.dicom.open(key, tcp.Seq+1)
		h.stack.healthCheckHandler.HandleTCPConnect(pkt, ipLayer, tcp, []*config.Device{device}, uint16(tcp.DstPort))
	case len(tcp.Payload) > 0 || tcp.FIN:
		answer, inOrder := h.dicom.receive(key, tcp.Seq, tcp.Payload)
		if !inOrder {
			return // the client retransmits what is missing
		}
		if tcp.FIN {
			h.dicom.close(key)
		}
		h.sendStatelessSegment(pkt, ipLayer, tcp, device, answer, "DICOM")
	}
}

// dicomAnswer is the reply to the complete PDUs at the front of stream, in
// order, and how many bytes they took. ended reports that the association is
// over: an abort, ours or the peer's, ends it and nothing after is answered.
// A header that cannot start a DICOM PDU is aborted as soon as it arrives,
// the way an SCP answers a port scanner's HTTP or TLS probe.
func dicomAnswer(stream []byte) ([]byte, int, bool) {
	var (
		answer   []byte
		consumed int
	)
	for len(stream)-consumed >= dicomPDUHeaderLen {
		pdu := stream[consumed:]
		length := uint64(binary.BigEndian.Uint32(pdu[2:dicomPDUHeaderLen]))
		switch {
		case pdu[0] < dicomAssociateRQ || pdu[0] > dicomAbort:
			return append(answer, dicomAbortPDU(dicomAbortUnrecognizedPDU)...), len(stream), true
		case length > dicomMaxPDU:
			return append(answer, dicomAbortPDU(dicomAbortInvalidParameter)...), len(stream), true
		case dicomPDUHeaderLen+length > uint64(len(pdu)):
			return answer, consumed, false
		}
		pduType, body := pdu[0], pdu[dicomPDUHeaderLen:dicomPDUHeaderLen+length]
		consumed += dicomPDUHeaderLen + int(length)
		switch pduType {
		case dicomAssociateRQ:
			answer = append(answer, dicomAssociateAnswer(body)...)
		case dicomPDataTF:
			reply, ok := dicomEchoReply(body)
			if !ok {
				return append(answer, dicomAbortPDU(dicomAbortInvalidParameter)...), consumed, true
			}
			answer = append(answer, reply...)
		case dicomReleaseRQ:
			answer = append(answer, dicomPDU(dicomReleaseRP, make([]byte, dicomItemHeaderLen))...)
		case dicomAbort:
			return answer, consumed, true
		default: // A-ASSOCIATE-AC, -RJ and A-RELEASE-RP come from an SCP, never to one
			return append(answer, dicomAbortPDU(dicomAbortUnexpectedPDU)...), consumed, true
		}
	}

	return answer, consumed, false
}

type dicomPresentationContext struct {
	id       byte
	abstract string
	transfer []string
}

// dicomAssociateAnswer accepts the Verification SOP class in Implicit or
// Explicit VR Little Endian and refuses every other presentation context: the
// archive and modality answer C-ECHO, but storage and worklist are the
// clinical application and are not simulated.
func dicomAssociateAnswer(body []byte) []byte {
	if len(body) < dicomAssociateLen {
		return dicomAbortPDU(dicomAbortInvalidParameter)
	}
	applicationContext, contexts, ok := parseDICOMAssociateItems(body[dicomAssociateLen:])
	if !ok {
		return dicomAbortPDU(dicomAbortInvalidParameter)
	}
	if applicationContext != dicomApplicationContext {
		return dicomPDU(dicomAssociateRJ, []byte{
			0, dicomRejectPermanent, dicomRejectSourceServiceUser, dicomRejectContextUnsupported,
		})
	}

	accept := make([]byte, 0, dicomAssociateLen)
	accept = binary.BigEndian.AppendUint16(accept, 1) // protocol version 1
	accept = append(accept, 0, 0)
	// The AE titles and the reserved field go back exactly as they came.
	accept = append(accept, body[4:dicomAssociateLen]...)
	accept = append(accept, dicomItem(dicomItemApplicationContext, []byte(dicomApplicationContext))...)
	for _, context := range contexts {
		result, transfer := negotiateDICOMContext(context)
		value := append([]byte{context.id, 0, result, 0},
			dicomItem(dicomItemTransferSyntax, []byte(transfer))...)
		accept = append(accept, dicomItem(dicomItemPresentationAC, value)...)
	}
	maxLength := binary.BigEndian.AppendUint32(nil, dicomMaxPDULength)
	userInformation := append(dicomItem(dicomItemMaxLength, maxLength),
		dicomItem(dicomItemImplementationUID, []byte(dicomImplementationUID))...)
	accept = append(accept, dicomItem(dicomItemUserInformation, userInformation)...)

	return dicomPDU(dicomAssociateAC, accept)
}

func negotiateDICOMContext(context dicomPresentationContext) (byte, string) {
	if context.abstract != dicomVerificationClass {
		return dicomContextAbstractRejected, dicomImplicitLittle
	}
	for _, transfer := range []string{dicomImplicitLittle, dicomExplicitLittle} {
		if slices.Contains(context.transfer, transfer) {
			return dicomContextAccepted, transfer
		}
	}

	return dicomContextTransferRejected, dicomImplicitLittle
}

// parseDICOMAssociateItems reads the application context and the proposed
// presentation contexts. User information is the requester's own and needs
// no answer beyond ours.
func parseDICOMAssociateItems(items []byte) (string, []dicomPresentationContext, bool) {
	var (
		applicationContext string
		contexts           []dicomPresentationContext
	)
	for len(items) > 0 {
		itemType, value, rest, ok := nextDICOMItem(items)
		if !ok {
			return "", nil, false
		}
		items = rest
		switch itemType {
		case dicomItemApplicationContext:
			applicationContext = dicomUID(value)
		case dicomItemPresentationRQ:
			context, valid := parseDICOMPresentationContext(value)
			if !valid {
				return "", nil, false
			}
			contexts = append(contexts, context)
		}
	}

	return applicationContext, contexts, true
}

func parseDICOMPresentationContext(value []byte) (dicomPresentationContext, bool) {
	const fixedLen = 4 // context ID and three reserved bytes
	if len(value) < fixedLen {
		return dicomPresentationContext{}, false
	}
	context := dicomPresentationContext{id: value[0]}
	for subItems := value[fixedLen:]; len(subItems) > 0; {
		itemType, uid, rest, ok := nextDICOMItem(subItems)
		if !ok {
			return dicomPresentationContext{}, false
		}
		subItems = rest
		switch itemType {
		case dicomItemAbstractSyntax:
			context.abstract = dicomUID(uid)
		case dicomItemTransferSyntax:
			context.transfer = append(context.transfer, dicomUID(uid))
		}
	}

	return context, true
}

func nextDICOMItem(items []byte) (byte, []byte, []byte, bool) {
	if len(items) < dicomItemHeaderLen {
		return 0, nil, nil, false
	}
	end := dicomItemHeaderLen + int(binary.BigEndian.Uint16(items[2:dicomItemHeaderLen]))
	if end > len(items) {
		return 0, nil, nil, false
	}

	return items[0], items[dicomItemHeaderLen:end], items[end:], true
}

// dicomUID drops the padding some implementations put after a UID.
func dicomUID(value []byte) string {
	return strings.TrimRight(string(value), "\x00 ")
}

// dicomEchoReply answers each C-ECHO-RQ command in a P-DATA-TF with a
// C-ECHO-RSP of status Success on the same presentation context. Data
// fragments carry nothing to answer; any other command, or a command split
// across fragments, is not something this service handles.
func dicomEchoReply(body []byte) ([]byte, bool) {
	var reply []byte
	for len(body) > 0 {
		if len(body) < dicomPDVHeaderLen {
			return nil, false
		}
		end := dicomPDVLengthLen + uint64(binary.BigEndian.Uint32(body[:dicomPDVLengthLen]))
		if end < dicomPDVHeaderLen || end > uint64(len(body)) {
			return nil, false
		}
		contextID, control, fragment := body[4], body[5], body[dicomPDVHeaderLen:end]
		body = body[end:]
		if control&0x01 == 0 {
			continue
		}
		if control != dicomPDVCommandLast {
			return nil, false
		}
		messageID, ok := dicomEchoRequestID(fragment)
		if !ok {
			return nil, false
		}
		command := dicomEchoResponse(messageID)
		pdv := binary.BigEndian.AppendUint32(nil, safeconv.Uint32(dicomPDVHeaderLen-dicomPDVLengthLen+len(command)))
		pdv = append(pdv, contextID, dicomPDVCommandLast)
		reply = append(reply, dicomPDU(dicomPDataTF, append(pdv, command...))...)
	}

	return reply, true
}

// dicomEchoRequestID returns the message ID of a C-ECHO-RQ command set, which
// is always Implicit VR Little Endian whatever the context's transfer syntax.
func dicomEchoRequestID(command []byte) (uint16, bool) {
	var (
		field, messageID uint16
		haveID           bool
	)
	for len(command) > 0 {
		if len(command) < dicomElementLen {
			return 0, false
		}
		group, element := binary.LittleEndian.Uint16(command[:2]), binary.LittleEndian.Uint16(command[2:4])
		end := dicomElementLen + uint64(binary.LittleEndian.Uint32(command[4:dicomElementLen]))
		if end > uint64(len(command)) {
			return 0, false
		}
		value := command[dicomElementLen:end]
		command = command[end:]
		if group != 0 || len(value) != 2 {
			continue
		}
		switch element {
		case dicomTagCommandField:
			field = binary.LittleEndian.Uint16(value)
		case dicomTagMessageID:
			messageID, haveID = binary.LittleEndian.Uint16(value), true
		}
	}

	return messageID, field == dicomCEchoRQ && haveID
}

func dicomEchoResponse(messageID uint16) []byte {
	var elements []byte
	elements = appendDICOMElement(elements, dicomTagAffectedSOPClass, append([]byte(dicomVerificationClass), 0))
	elements = appendDICOMElement(elements, dicomTagCommandField, binary.LittleEndian.AppendUint16(nil, dicomCEchoRSP))
	elements = appendDICOMElement(
		elements,
		dicomTagMessageIDResponded,
		binary.LittleEndian.AppendUint16(nil, messageID),
	)
	elements = appendDICOMElement(elements, dicomTagDataSetType, binary.LittleEndian.AppendUint16(nil, dicomNoDataSet))
	elements = appendDICOMElement(elements, dicomTagStatus, binary.LittleEndian.AppendUint16(nil, 0))
	groupLength := binary.LittleEndian.AppendUint32(nil, safeconv.Uint32(len(elements)))

	return append(appendDICOMElement(nil, dicomTagGroupLength, groupLength), elements...)
}

// appendDICOMElement encodes a command element: group 0000, Implicit VR.
func appendDICOMElement(buffer []byte, element uint16, value []byte) []byte {
	buffer = binary.LittleEndian.AppendUint16(buffer, 0)
	buffer = binary.LittleEndian.AppendUint16(buffer, element)
	buffer = binary.LittleEndian.AppendUint32(buffer, safeconv.Uint32(len(value)))

	return append(buffer, value...)
}

func dicomItem(itemType byte, value []byte) []byte {
	item := []byte{itemType, 0}
	item = binary.BigEndian.AppendUint16(item, safeconv.Uint16(len(value)))

	return append(item, value...)
}

func dicomPDU(pduType byte, body []byte) []byte {
	pdu := []byte{pduType, 0}
	pdu = binary.BigEndian.AppendUint32(pdu, safeconv.Uint32(len(body)))

	return append(pdu, body...)
}

func dicomAbortPDU(reason byte) []byte {
	return dicomPDU(dicomAbort, []byte{0, 0, dicomAbortSourceProvider, reason})
}
