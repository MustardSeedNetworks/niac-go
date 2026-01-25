package protocols

import (
	"encoding/hex"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"

	"github.com/krisarmstrong/niac-go/internal/config"
)

// dnsResolveContext holds state for resolving DNS questions.
type dnsResolveContext struct {
	answers      []layers.DNSResourceRecord
	responseCode layers.DNSResponseCode
	debugLevel   int
	serial       int
}

// HandleQuery processes a DNS query.
func (h *DNSHandler) HandleQuery(
	pkt *Packet,
	ipLayer *layers.IPv4,
	udpLayer *layers.UDP,
	devices []*config.Device,
) {
	debugLevel := h.stack.GetDebugLevel()

	packet := gopacket.NewPacket(pkt.Buffer, layers.LayerTypeEthernet, gopacket.Default)

	dns, ok := h.parseDNSLayer(packet, debugLevel, pkt.SerialNumber, "DNS")
	if !ok {
		return
	}

	h.stack.IncrementStat("dns_queries")
	h.logDNSQueries(dns.Questions, ipLayer.SrcIP, debugLevel, pkt.SerialNumber)

	serverDevice, serverIP := h.selectServerDevice(devices, false)
	if !h.validateServerDevice(serverDevice, serverIP, debugLevel, pkt.SerialNumber, "DNS") {
		return
	}

	if h.handleNBSTATIfPresent(pkt, ipLayer, udpLayer, serverDevice, dns, packet, debugLevel) {
		return
	}

	response := h.buildDNSResponse(dns, serverDevice, debugLevel, pkt.SerialNumber)
	srcMAC := h.extractSourceMAC(packet)
	h.sendAndLogResponse(response, serverIP, ipLayer.SrcIP, serverDevice.MACAddress, srcMAC,
		udpLayer.SrcPort, debugLevel, pkt.SerialNumber)
}

// HandleQueryV6 processes a DNS query over IPv6.
func (h *DNSHandler) HandleQueryV6(
	pkt *Packet,
	packet gopacket.Packet,
	ipv6 *layers.IPv6,
	udpLayer *layers.UDP,
	devices []*config.Device,
) {
	debugLevel := h.stack.GetDebugLevel()

	dns, ok := h.parseDNSLayer(packet, debugLevel, pkt.SerialNumber, "DNS/IPv6")
	if !ok {
		return
	}

	h.stack.IncrementStat("dns_queries")

	serverDevice, serverIP := h.selectServerDevice(devices, true)
	if !h.validateServerDevice(serverDevice, serverIP, debugLevel, pkt.SerialNumber, "DNS/IPv6") {
		return
	}

	if h.handleNBSTATIfPresentV6(pkt, ipv6, udpLayer, serverDevice, dns, packet, debugLevel) {
		return
	}

	response := h.buildDNSResponseV6(dns, serverDevice, debugLevel, pkt.SerialNumber)

	dstMAC, ok := h.extractSourceMACWithValidation(packet, debugLevel, pkt.SerialNumber)
	if !ok {
		return
	}

	h.sendAndLogResponseV6(response, serverIP, ipv6.SrcIP, serverDevice.MACAddress, dstMAC,
		udpLayer.SrcPort, debugLevel, pkt.SerialNumber)
}

// parseDNSLayer extracts and validates the DNS layer from a packet.
func (h *DNSHandler) parseDNSLayer(
	packet gopacket.Packet,
	debugLevel int,
	serial int,
	prefix string,
) (*layers.DNS, bool) {
	dnsLayer := packet.Layer(layers.LayerTypeDNS)
	if dnsLayer == nil {
		if debugLevel >= DebugLevelInfo {
			_, _ = fmt.Fprintf(os.Stdout, "%s packet missing DNS layer sn=%d\n", prefix, serial)
		}

		return nil, false
	}

	dns, ok := dnsLayer.(*layers.DNS)

	return dns, ok
}

// logDNSQueries logs DNS queries at verbose debug level.
func (h *DNSHandler) logDNSQueries(
	questions []layers.DNSQuestion,
	srcIP any,
	debugLevel int,
	serial int,
) {
	if debugLevel < DebugLevelVerbose {
		return
	}

	for _, q := range questions {
		_, _ = fmt.Fprintf(os.Stdout, "DNS Query: %s type=%s class=%s from %s sn=%d\n",
			string(q.Name), q.Type, q.Class, srcIP, serial)
	}
}

// validateServerDevice checks if the server device is properly configured.
func (h *DNSHandler) validateServerDevice(
	serverDevice *config.Device,
	serverIP net.IP,
	debugLevel int,
	serial int,
	prefix string,
) bool {
	if serverDevice == nil || serverIP == nil {
		if debugLevel >= DebugLevelInfo {
			_, _ = fmt.Fprintf(os.Stdout, "%s: No server device/IP configured sn=%d\n", prefix, serial)
		}

		return false
	}

	if len(serverDevice.MACAddress) == 0 {
		if debugLevel >= DebugLevelInfo {
			_, _ = fmt.Fprintf(os.Stdout, "%s: Server device %s missing MAC address sn=%d\n",
				prefix, serverDevice.Name, serial)
		}

		return false
	}

	return true
}

// handleNBSTATIfPresent checks for and handles NBSTAT queries.
// Returns true if an NBSTAT query was handled.
func (h *DNSHandler) handleNBSTATIfPresent(
	pkt *Packet,
	ipLayer *layers.IPv4,
	udpLayer *layers.UDP,
	serverDevice *config.Device,
	dns *layers.DNS,
	packet gopacket.Packet,
	debugLevel int,
) bool {
	for _, q := range dns.Questions {
		if q.Type != layers.DNSType(dnsTypeNBSTAT) {
			continue
		}

		err := h.handleNBSTATQuery(pkt, ipLayer, udpLayer, serverDevice, dns.ID, q, packet)
		if err != nil && debugLevel >= DebugLevelInfo {
			_, _ = fmt.Fprintf(os.Stdout, "DNS: NBSTAT handling failed: %v sn=%d\n", err, pkt.SerialNumber)
		}

		return true
	}

	return false
}

// handleNBSTATIfPresentV6 checks for and handles NBSTAT queries over IPv6.
// Returns true if an NBSTAT query was handled.
func (h *DNSHandler) handleNBSTATIfPresentV6(
	pkt *Packet,
	ipv6 *layers.IPv6,
	udpLayer *layers.UDP,
	serverDevice *config.Device,
	dns *layers.DNS,
	packet gopacket.Packet,
	debugLevel int,
) bool {
	for _, q := range dns.Questions {
		if q.Type != layers.DNSType(dnsTypeNBSTAT) {
			continue
		}

		err := h.handleNBSTATQueryV6(pkt, ipv6, udpLayer, serverDevice, dns.ID, q, packet)
		if err != nil && debugLevel >= DebugLevelInfo {
			_, _ = fmt.Fprintf(os.Stdout, "DNS/IPv6: NBSTAT handling failed: %v sn=%d\n",
				err, pkt.SerialNumber)
		}

		return true
	}

	return false
}

func (h *DNSHandler) handleNBSTATQuery(
	_ *Packet,
	ipLayer *layers.IPv4,
	udpLayer *layers.UDP,
	serverDevice *config.Device,
	id uint16,
	q layers.DNSQuestion,
	packet gopacket.Packet,
) error {
	payload := h.buildNBSTATResponse(serverDevice, id, q)
	if len(payload) == 0 {
		return nil
	}

	ethLayer := packet.Layer(layers.LayerTypeEthernet)

	var dstMAC net.HardwareAddr

	if eth, ok := ethLayer.(*layers.Ethernet); ok {
		dstMAC = eth.SrcMAC
	}

	return h.stack.udpHandler.SendUDP(
		serverDeviceIP(serverDevice, false),
		ipLayer.SrcIP,
		dnsPort,
		uint16(udpLayer.SrcPort),
		payload,
		[]byte(serverDevice.MACAddress),
		[]byte(dstMAC),
	)
}

func (h *DNSHandler) handleNBSTATQueryV6(
	_ *Packet,
	ipv6 *layers.IPv6,
	udpLayer *layers.UDP,
	serverDevice *config.Device,
	id uint16,
	q layers.DNSQuestion,
	packet gopacket.Packet,
) error {
	payload := h.buildNBSTATResponse(serverDevice, id, q)
	if len(payload) == 0 {
		return nil
	}

	ethLayer := packet.Layer(layers.LayerTypeEthernet)

	var dstMAC net.HardwareAddr

	if eth, ok := ethLayer.(*layers.Ethernet); ok {
		dstMAC = eth.SrcMAC
	}

	return h.sendDNSPayloadV6(
		payload,
		serverDeviceIP(serverDevice, true),
		ipv6.SrcIP,
		serverDevice.MACAddress,
		dstMAC,
		udpLayer.SrcPort,
	)
}

// extractSourceMAC extracts the source MAC address from the Ethernet layer.
func (h *DNSHandler) extractSourceMAC(packet gopacket.Packet) net.HardwareAddr {
	ethLayer := packet.Layer(layers.LayerTypeEthernet)
	if eth, ok := ethLayer.(*layers.Ethernet); ok {
		return eth.SrcMAC
	}

	return nil
}

// extractSourceMACWithValidation extracts the source MAC and validates the Ethernet layer.
func (h *DNSHandler) extractSourceMACWithValidation(
	packet gopacket.Packet,
	debugLevel int,
	serial int,
) (net.HardwareAddr, bool) {
	ethLayer := packet.Layer(layers.LayerTypeEthernet)
	if ethLayer == nil {
		if debugLevel >= DebugLevelInfo {
			_, _ = fmt.Fprintf(os.Stdout, "DNS/IPv6: Missing Ethernet layer sn=%d\n", serial)
		}

		return nil, false
	}

	eth, ok := ethLayer.(*layers.Ethernet)
	if !ok {
		return nil, false
	}

	return eth.SrcMAC, true
}

func (h *DNSHandler) resolveQuestions(
	questions []layers.DNSQuestion,
	set *dnsRecordSet,
	debugLevel int,
	serial int,
) ([]layers.DNSResourceRecord, layers.DNSResponseCode) {
	ctx := &dnsResolveContext{
		answers:      make([]layers.DNSResourceRecord, 0, len(questions)),
		responseCode: layers.DNSResponseCodeNoErr,
		debugLevel:   debugLevel,
		serial:       serial,
	}

	for _, q := range questions {
		if !h.validateAndResolveQuestion(q, set, ctx) {
			continue
		}
	}

	if len(ctx.answers) == 0 && ctx.responseCode == layers.DNSResponseCodeNoErr {
		ctx.responseCode = layers.DNSResponseCodeNXDomain
	}

	return ctx.answers, ctx.responseCode
}

// validateAndResolveQuestion validates and resolves a single DNS question.
// Returns false if the question should be skipped.
func (h *DNSHandler) validateAndResolveQuestion(
	q layers.DNSQuestion,
	set *dnsRecordSet,
	ctx *dnsResolveContext,
) bool {
	if !isValidDNSName(q.Name) {
		h.logInvalidDNSName(q.Name, ctx)
		return false
	}

	hostname := strings.ToLower(strings.TrimSuffix(string(q.Name), "."))
	h.resolveByType(q, hostname, set, ctx)

	return true
}

// resolveByType dispatches to the appropriate resolver based on DNS query type.
func (h *DNSHandler) resolveByType(
	q layers.DNSQuestion,
	hostname string,
	set *dnsRecordSet,
	ctx *dnsResolveContext,
) {
	//exhaustive:ignore
	switch q.Type {
	case layers.DNSTypeA:
		h.resolveARecord(q, hostname, set, ctx)
	case layers.DNSTypeAAAA:
		h.resolveAAAARecord(q, hostname, set, ctx)
	case layers.DNSTypePTR:
		h.resolvePTRRecord(q, set, ctx)
	case layers.DNSType(dnsTypeNBSTAT):
		// NBSTAT handled separately in HandleQuery.
	}
}

// resolveARecord resolves DNS A record queries.
func (h *DNSHandler) resolveARecord(
	q layers.DNSQuestion,
	hostname string,
	set *dnsRecordSet,
	ctx *dnsResolveContext,
) {
	for _, rec := range h.lookupHost(hostname, set) {
		if rec.ip.To4() == nil {
			continue
		}

		ctx.answers = append(ctx.answers, layers.DNSResourceRecord{
			Name:  q.Name,
			Type:  layers.DNSTypeA,
			Class: layers.DNSClassIN,
			TTL:   rec.ttl,
			IP:    rec.ip,
		})

		if rec.rcode != layers.DNSResponseCodeNoErr {
			ctx.responseCode = rec.rcode
		}

		h.logDNSRecord("A", hostname, rec.ip.String(), ctx)
	}
}

// resolveAAAARecord resolves DNS AAAA record queries.
func (h *DNSHandler) resolveAAAARecord(
	q layers.DNSQuestion,
	hostname string,
	set *dnsRecordSet,
	ctx *dnsResolveContext,
) {
	for _, rec := range h.lookupHost(hostname, set) {
		if rec.ip.To4() != nil || rec.ip.To16() == nil {
			continue
		}

		ctx.answers = append(ctx.answers, layers.DNSResourceRecord{
			Name:  q.Name,
			Type:  layers.DNSTypeAAAA,
			Class: layers.DNSClassIN,
			TTL:   rec.ttl,
			IP:    rec.ip,
		})

		if rec.rcode != layers.DNSResponseCodeNoErr {
			ctx.responseCode = rec.rcode
		}

		h.logDNSRecord("AAAA", hostname, rec.ip.String(), ctx)
	}
}

// resolvePTRRecord resolves DNS PTR record queries.
func (h *DNSHandler) resolvePTRRecord(
	q layers.DNSQuestion,
	set *dnsRecordSet,
	ctx *dnsResolveContext,
) {
	ip, ok, isV6 := parsePTRName(q.Name)
	if !ok {
		h.logPTRParseFailure(q.Name, ctx)
		return
	}

	host, found := h.lookupPTR(ip, set)
	if !found || host.name == "" {
		if isV6 {
			ctx.responseCode = layers.DNSResponseCodeNXDomain
		}

		return
	}

	ptr := host.name
	if !strings.HasSuffix(ptr, ".") {
		ptr += "."
	}

	ctx.answers = append(ctx.answers, layers.DNSResourceRecord{
		Name:  q.Name,
		Type:  layers.DNSTypePTR,
		Class: layers.DNSClassIN,
		TTL:   host.ttl,
		PTR:   []byte(ptr),
	})

	if host.rcode != layers.DNSResponseCodeNoErr {
		ctx.responseCode = host.rcode
	}

	h.logDNSRecord("PTR", string(q.Name), ptr, ctx)
}

// logInvalidDNSName logs an invalid DNS name error.
func (h *DNSHandler) logInvalidDNSName(name []byte, ctx *dnsResolveContext) {
	if ctx.debugLevel >= DebugLevelInfo {
		_, _ = fmt.Fprintf(
			os.Stdout,
			"DNS: Invalid domain name length (> 255 or label > 63): %s sn=%d\n",
			name,
			ctx.serial,
		)
	}
}

// logDNSRecord logs a resolved DNS record.
func (h *DNSHandler) logDNSRecord(recordType, name, value string, ctx *dnsResolveContext) {
	if ctx.debugLevel >= DebugLevelInfo {
		_, _ = fmt.Fprintf(
			os.Stdout,
			"DNS: %s -> %s (%s record) sn=%d\n",
			name,
			value,
			recordType,
			ctx.serial,
		)
	}
}

// logPTRParseFailure logs a PTR query parse failure.
func (h *DNSHandler) logPTRParseFailure(name []byte, ctx *dnsResolveContext) {
	if ctx.debugLevel >= DebugLevelInfo {
		_, _ = fmt.Fprintf(
			os.Stdout,
			"DNS: PTR query %s could not be parsed sn=%d\n",
			name,
			ctx.serial,
		)
	}
}

func parsePTRName(name []byte) (net.IP, bool, bool) {
	ptrName := strings.ToLower(strings.TrimSuffix(string(name), "."))

	switch {
	case strings.HasSuffix(ptrName, ".in-addr.arpa"):
		ip, ok := parseIPv4PTRName(ptrName)

		return ip, ok, false
	case strings.HasSuffix(ptrName, ".ip6.arpa"):
		ip, ok := parseIPv6PTRName(ptrName)

		return ip, ok, true
	default:
		return nil, false, false
	}
}

func parseIPv4PTRName(name string) (net.IP, bool) {
	base := strings.TrimSuffix(name, ".in-addr.arpa")

	parts := strings.Split(strings.Trim(base, "."), ".")
	if len(parts) != dnsIPv4Octets {
		return nil, false
	}

	ip := net.IPv4(0, 0, 0, 0).To4()

	for i := range dnsIPv4Octets {
		val, err := strconv.Atoi(parts[len(parts)-1-i])
		if err != nil || val < 0 || val > dnsMaxByteValue {
			return nil, false
		}

		ip[i] = byte(val)
	}

	return ip, true
}

func parseIPv6PTRName(name string) (net.IP, bool) {
	base := strings.TrimSuffix(name, ".ip6.arpa")

	nibbles := strings.Split(strings.Trim(base, "."), ".")
	if len(nibbles) != dnsIPv6NibbleLen {
		return nil, false
	}

	var builder strings.Builder

	builder.Grow(dnsIPv6NibbleLen)

	for i := len(nibbles) - 1; i >= 0; i-- {
		if len(nibbles[i]) != 1 {
			return nil, false
		}

		builder.WriteString(nibbles[i])
	}

	data, err := hex.DecodeString(builder.String())
	if err != nil || len(data) != net.IPv6len {
		return nil, false
	}

	return net.IP(data), true
}
