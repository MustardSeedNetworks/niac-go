package protocols

import (
	"fmt"
	"hash/fnv"
	"net"
	"net/netip"
	"strconv"
	"strings"

	"github.com/gopacket/gopacket/layers"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/logging"
)

// UDPPortSIP is where a SIP user agent listens for requests over UDP.
const UDPPortSIP = 5060

// sipAllow is the method list a Cisco 8800-series handset returns to OPTIONS.
// It describes the phone being modelled, not what NIAC implements: only
// OPTIONS is answered, because calls and registrations are the application
// workflow and the simulation stops at identity.
const sipAllow = "ACK,BYE,CANCEL,INVITE,NOTIFY,OPTIONS,REFER,REGISTER,UPDATE,SUBSCRIBE,INFO"

// sipHeaderName lower-cases a header name and expands the RFC 3261 §7.3.3
// compact forms of the headers a response copies.
func sipHeaderName(name string) string {
	switch name = strings.ToLower(strings.TrimSpace(name)); name {
	case "v":
		return "via"
	case "f":
		return "from"
	case "t":
		return "to"
	case "i":
		return "call-id"
	}
	return name
}

// sipRequest is the part of a request a response has to echo (RFC 3261
// §8.2.6.2): every Via in order, From, To, Call-ID and CSeq.
type sipRequest struct {
	method string
	vias   []string
	from   string
	to     string
	callID string
	cseq   string
}

// handleSIP answers a request to a SIP-enabled device and reports whether one
// owned the destination. A false return leaves the datagram to the
// closed-port path, so 5060 on every other device behaves as before.
func (h *UDPHandler) handleSIP(pkt *Packet, ipLayer *layers.IPv4, udp *layers.UDP, devices []*config.Device) bool {
	device := h.sipDeviceForIP(devices, ipLayer.DstIP)
	if device == nil {
		return false
	}

	source, valid := netip.AddrFromSlice(ipLayer.SrcIP)
	if !valid {
		return true
	}
	response, port, ok := sipOptionsResponse(
		udp.Payload, netip.AddrPortFrom(source.Unmap(), uint16(udp.SrcPort)), device,
	)
	if !ok {
		return true
	}

	identity, ok := h.stack.replyEthernet(pkt, device)
	if !ok {
		return true
	}
	err := h.SendUDP(
		device,
		ipLayer.DstIP.To4(), ipLayer.SrcIP.To4(),
		UDPPortSIP, port,
		response,
		identity.source, identity.destination, identity.vlan,
	)
	if err != nil && h.stack.GetDebugLevel() >= DebugLevelInfo {
		logging.Debugf("SIP: response from %s failed sn=%d err=%v", device.Name, pkt.SerialNumber, err)
	}

	return true
}

func (h *UDPHandler) sipDeviceForIP(devices []*config.Device, ip net.IP) *config.Device {
	for _, device := range devices {
		if device.SIPConfig != nil && device.SIPConfig.Enabled && h.stack.deviceOwnsIPv4(device, ip) {
			return device
		}
	}

	return nil
}

// sipOptionsResponse builds the 200 OK to an OPTIONS request from source, and
// the UDP port it goes to. Anything else -- another method, a response, a
// request missing a header the answer must echo -- gets no reply.
func sipOptionsResponse(payload []byte, source netip.AddrPort, device *config.Device) ([]byte, uint16, bool) {
	request, ok := parseSIPRequest(string(payload))
	if !ok || request.method != "OPTIONS" {
		return nil, 0, false
	}
	topVia, port, ok := sipStampVia(request.vias[0], source)
	if !ok {
		return nil, 0, false
	}

	to := request.to
	if !sipHasTag(to) {
		to += ";tag=" + sipTag(device.Name, request.callID)
	}

	var response strings.Builder
	response.WriteString("SIP/2.0 200 OK\r\n")
	response.WriteString("Via: " + topVia + "\r\n")
	for _, via := range request.vias[1:] {
		response.WriteString("Via: " + via + "\r\n")
	}
	response.WriteString("From: " + request.from + "\r\n")
	response.WriteString("To: " + to + "\r\n")
	response.WriteString("Call-ID: " + request.callID + "\r\n")
	response.WriteString("CSeq: " + request.cseq + "\r\n")
	response.WriteString("Allow: " + sipAllow + "\r\n")
	response.WriteString("Accept: application/sdp\r\n")
	// RFC 3261 names the answering software in Server; scanners read either
	// header, and handsets differ in which one they send, so both carry it.
	response.WriteString("Server: " + device.SIPConfig.UserAgent + "\r\n")
	response.WriteString("User-Agent: " + device.SIPConfig.UserAgent + "\r\n")
	response.WriteString("Content-Length: 0\r\n\r\n")

	return []byte(response.String()), port, true
}

// parseSIPRequest reads the request line and the headers a response echoes.
// It rejects a request whose CSeq names a different method (RFC 3261 §8.1.1.5)
// or that lacks one of those headers, since no valid answer can be built.
func parseSIPRequest(message string) (sipRequest, bool) {
	head, _, _ := strings.Cut(message, "\r\n\r\n")
	lines := strings.Split(strings.ReplaceAll(head, "\r\n", "\n"), "\n")
	parts := strings.Fields(lines[0])
	if len(parts) != 3 || parts[2] != "SIP/2.0" {
		return sipRequest{}, false
	}

	request := sipRequest{method: parts[0]}
	for _, line := range lines[1:] {
		name, value, found := strings.Cut(line, ":")
		if !found {
			continue
		}
		value = strings.TrimSpace(value)
		switch sipHeaderName(name) {
		case "via":
			for via := range strings.SplitSeq(value, ",") {
				request.vias = append(request.vias, strings.TrimSpace(via))
			}
		case "from":
			request.from = value
		case "to":
			request.to = value
		case "call-id":
			request.callID = value
		case "cseq":
			request.cseq = value
		}
	}

	_, cseqMethod, _ := strings.Cut(request.cseq, " ")
	if len(request.vias) == 0 || request.from == "" || request.to == "" || request.callID == "" ||
		strings.TrimSpace(cseqMethod) != request.method {
		return sipRequest{}, false
	}

	return request, true
}

// sipStampVia records where the request really came from in the top Via and
// returns the port the response goes to. The address is always the source
// (RFC 3261 §18.2.2 with received); the port is the source port when the
// client asked for rport (RFC 3581), otherwise the sent-by port.
func sipStampVia(via string, source netip.AddrPort) (string, uint16, bool) {
	protocol, rest, found := strings.Cut(via, " ")
	if !found || !strings.EqualFold(protocol, "SIP/2.0/UDP") {
		return "", 0, false
	}
	params := strings.Split(strings.TrimSpace(rest), ";")
	host, port := sipSentBy(params[0])
	if host == "" {
		return "", 0, false
	}

	received := source.Addr().String()
	rport := false
	for index := 1; index < len(params); index++ {
		if strings.EqualFold(strings.TrimSpace(params[index]), "rport") {
			params[index] = "rport=" + strconv.Itoa(int(source.Port()))
			rport = true
		}
	}
	if rport || host != received {
		params = append(params, "received="+received)
	}
	if rport {
		port = source.Port()
	}

	return protocol + " " + strings.Join(params, ";"), port, true
}

// sipSentBy splits a Via sent-by into host and port, the port defaulting to
// 5060.
func sipSentBy(sentBy string) (string, uint16) {
	sentBy = strings.TrimSpace(sentBy)
	if host, portText, err := net.SplitHostPort(sentBy); err == nil {
		port, parseErr := strconv.ParseUint(portText, 10, 16)
		if parseErr != nil || port == 0 {
			return "", 0
		}
		return host, uint16(port)
	}

	return strings.Trim(sentBy, "[]"), UDPPortSIP
}

func sipHasTag(header string) bool {
	_, params, _ := strings.Cut(header, ">")
	if !strings.Contains(header, ">") {
		params = header
	}
	for param := range strings.SplitSeq(params, ";") {
		name, _, _ := strings.Cut(strings.TrimSpace(param), "=")
		if strings.EqualFold(name, "tag") {
			return true
		}
	}

	return false
}

// sipTag is the To tag for an out-of-dialog answer. It is derived rather than
// random so a retransmitted OPTIONS gets the same answer, as RFC 3261 §8.2.6.2
// expects of the UAS.
func sipTag(deviceName, callID string) string {
	hash := fnv.New32a()
	_, _ = hash.Write([]byte(deviceName + "\x00" + callID))

	return fmt.Sprintf("%08x", hash.Sum32())
}
