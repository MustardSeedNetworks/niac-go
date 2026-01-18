package protocols

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"

	"github.com/krisarmstrong/niac-go/internal/config"
)

// DNS protocol constants.
const (
	// dnsDefaultTTL is the default TTL for DNS records (5 minutes).
	dnsDefaultTTL = 300

	// dnsTypeNBSTAT is the DNS type code for NetBIOS Status Query.
	dnsTypeNBSTAT = 33

	// dnsPort is the standard DNS port.
	dnsPort = 53

	// dnsHeaderSize is the size of the DNS header in bytes.
	dnsHeaderSize = 12

	// DNS flag bits.
	dnsFlagQR = 0x8000 // Query/Response flag
	dnsFlagAA = 0x0400 // Authoritative Answer flag

	// dnsPointerByte is the high byte of a DNS name pointer.
	dnsPointerByte = 0xC0

	// dnsPointerOffset is the offset for name pointer (0x0C = 12).
	dnsPointerOffset = 0x0C

	// DNS encoding constants.
	dnsTerminator    = 0x00 // DNS name terminator byte
	dnsQuestionExtra = 4    // Extra bytes for question (Type + Class)
	dnsBufPadding    = 2    // Buffer padding for encoded name
	dnsByteShift     = 8    // Bit shift for encoding high byte of uint16
	dnsIPv6NibbleLen = 32   // Expected number of nibbles in IPv6 reverse lookup
	dnsIPv4Octets    = 4    // Number of octets in IPv4 address
	dnsMACOctets     = 6    // Number of octets in MAC address
	dnsMinLabelParts = 2    // Minimum label parts for PTR lookup
	dnsMaxByteValue  = 255  // Maximum value for a single byte
	dnsMaxLabelLen   = 63   // Maximum DNS label length per RFC 1035
	dnsMaxNameLen    = 255  // Maximum DNS name length per RFC 1035

	// NetBIOS constants.
	netbiosNameLen   = 15     // NetBIOS name length (without suffix)
	netbiosStatsSize = 40     // NetBIOS statistics block size
	netbiosGroupFlag = 0x8000 // NetBIOS group name flag

	// IP protocol constants for DNS responses.
	dnsIPv4Version       = 4  // IPv4 version field
	dnsIPv4TTL           = 64 // IPv4 default TTL
	dnsIPv6Version       = 6  // IPv6 version field
	dnsIPv6HopLimit      = 64 // IPv6 default hop limit
	dnsDotASCII          = 46 // ASCII value for '.'
	dnsArpaLabelParts    = 13 // Expected parts for ip6.arpa PTR lookup
	dnsChecksumWordStep  = 2  // Step size for 16-bit checksum calculation
	dnsChecksumWordShift = 4  // Checksum bit shift for IPv6 nibbles

	// NetBIOS name types (suffix bytes).
	nbNameTypeWorkstation   = 0x00 // Workstation Service
	nbNameTypeMessenger     = 0x03 // Messenger Service
	nbNameTypeFileServer    = 0x20 // File Server Service
	nbNameTypeDomainMaster  = 0x1B // Domain Master Browser
	nbNameTypeMasterBrowser = 0x1D // Master Browser
	nbNameTypeBrowserElec   = 0x1E // Browser Election
	nbNameTypeMSBrowse      = 0x01 // MSBROWSE / Internet Group Name Flag

	// NetBIOS NBSTAT constants.
	nbstatMACAndStatsSize  = 46   // MAC (6) + statistics (40)
	nbMaxNibbleValue       = 0x0F // Maximum nibble value for encoding
	nbstatNameEntrySize    = 18   // Name (15) + suffix (1) + flags (2)
	nbstatOwnerTypeShift   = 13   // Bit shift for owner node type
	nbNibbleEncodingFactor = 2    // Factor for nibble encoding/decoding
	nbNibbleShift          = 4    // Bit shift for nibble operations

	// NetBIOS node types (for netbiosOwnerNodeType).
	nbNodeTypeB = 0 // B-node (Broadcast)
	nbNodeTypeP = 1 // P-node (Point-to-Point)
	nbNodeTypeM = 2 // M-node (Mixed)
	nbNodeTypeH = 3 // H-node (Hybrid)
)

// DNSHandler handles DNS queries and responses.
type DNSHandler struct {
	stack         *Stack
	records       map[string][]dnsRecord // Hostname -> records
	ptrRecords    map[string]dnsPTR      // IP -> PTR record
	deviceRecords map[*config.Device]*dnsRecordSet
	mu            sync.RWMutex
	domain        string // Default domain
}

type dnsRecord struct {
	ip    net.IP
	ttl   uint32
	rcode layers.DNSResponseCode
}

type dnsPTR struct {
	name  string
	ttl   uint32
	rcode layers.DNSResponseCode
}

type dnsRecordSet struct {
	forward map[string][]dnsRecord
	reverse map[string]dnsPTR
}

// NewDNSHandler creates a new DNS handler.
func NewDNSHandler(stack *Stack) *DNSHandler {
	return &DNSHandler{
		stack:         stack,
		records:       make(map[string][]dnsRecord),
		ptrRecords:    make(map[string]dnsPTR),
		deviceRecords: make(map[*config.Device]*dnsRecordSet),
		domain:        "local",
	}
}

// Reset clears all cached DNS records.
func (h *DNSHandler) Reset() {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.records = make(map[string][]dnsRecord)
	h.ptrRecords = make(map[string]dnsPTR)
	h.deviceRecords = make(map[*config.Device]*dnsRecordSet)
	h.domain = "local"
}

// AddRecord adds a DNS A/AAAA record.
func (h *DNSHandler) AddRecord(hostname string, ip net.IP) {
	h.AddRecordWithTTL(hostname, ip, dnsDefaultTTL, layers.DNSResponseCodeNoErr)
}

// AddRecordWithTTL adds a DNS A/AAAA record with TTL and response code.
func (h *DNSHandler) AddRecordWithTTL(
	hostname string,
	ip net.IP,
	ttl uint32,
	rcode layers.DNSResponseCode,
) {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Normalize hostname
	hostname = strings.ToLower(strings.TrimSuffix(hostname, "."))

	// Add forward record
	h.records[hostname] = append(h.records[hostname], dnsRecord{
		ip:    ip,
		ttl:   ttl,
		rcode: rcode,
	})

	// Add reverse record (PTR)
	h.ptrRecords[ip.String()] = dnsPTR{
		name:  hostname,
		ttl:   ttl,
		rcode: rcode,
	}
}

// AddPTRRecord adds a PTR record with TTL and response code.
func (h *DNSHandler) AddPTRRecord(
	ip net.IP,
	hostname string,
	ttl uint32,
	rcode layers.DNSResponseCode,
) {
	h.mu.Lock()
	defer h.mu.Unlock()

	hostname = strings.ToLower(strings.TrimSuffix(hostname, "."))
	h.ptrRecords[ip.String()] = dnsPTR{
		name:  hostname,
		ttl:   ttl,
		rcode: rcode,
	}
}

// SetDomain sets the default DNS domain.
func (h *DNSHandler) SetDomain(domain string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.domain = domain
}

// LoadDeviceRecords loads DNS records from configured devices.
func (h *DNSHandler) LoadDeviceRecords(devices []*config.Device) {
	for _, device := range devices {
		hostname := device.SNMPConfig.SysName
		if hostname == "" {
			hostname = device.Name
		}

		for _, ip := range device.IPAddresses {
			h.AddRecord(hostname, ip)
		}
	}
}

// LoadDeviceDNSConfig loads device-specific DNS records.
func (h *DNSHandler) LoadDeviceDNSConfig(device *config.Device) {
	if device == nil || device.DNSConfig == nil {
		return
	}

	set := &dnsRecordSet{
		forward: make(map[string][]dnsRecord),
		reverse: make(map[string]dnsPTR),
	}

	for _, rec := range device.DNSConfig.ForwardRecords {
		name := strings.ToLower(strings.TrimSuffix(rec.Name, "."))
		set.forward[name] = append(set.forward[name], dnsRecord{
			ip:    rec.IP,
			ttl:   rec.TTL,
			rcode: layers.DNSResponseCode(safeDNSRCode(rec.RCode)),
		})
	}

	for _, rec := range device.DNSConfig.ReverseRecords {
		set.reverse[rec.IP.String()] = dnsPTR{
			name:  strings.ToLower(strings.TrimSuffix(rec.Name, ".")),
			ttl:   rec.TTL,
			rcode: layers.DNSResponseCode(safeDNSRCode(rec.RCode)),
		}
	}

	h.mu.Lock()
	h.deviceRecords[device] = set
	h.mu.Unlock()
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

// buildDNSResponse constructs a DNS response for the given query.
func (h *DNSHandler) buildDNSResponse(
	dns *layers.DNS,
	serverDevice *config.Device,
	debugLevel int,
	serial int,
) *layers.DNS {
	response := &layers.DNS{
		ID:           dns.ID,
		QR:           true,
		OpCode:       dns.OpCode,
		AA:           true,
		TC:           false,
		RD:           dns.RD,
		RA:           true,
		ResponseCode: layers.DNSResponseCodeNoErr,
		Questions:    dns.Questions,
		Answers:      []layers.DNSResourceRecord{},
	}

	recordSet := h.getRecordSetForDevice(serverDevice)
	response.Answers, response.ResponseCode = h.resolveQuestions(dns.Questions, recordSet, debugLevel, serial)

	if len(response.Answers) == 0 && debugLevel >= DebugLevelInfo {
		_, _ = fmt.Fprintf(os.Stdout, "DNS: NXDOMAIN for queries sn=%d\n", serial)
	} else if len(response.Answers) > 0 && response.ResponseCode == 0 {
		response.ResponseCode = layers.DNSResponseCodeNoErr
	}

	return response
}

// extractSourceMAC extracts the source MAC address from the Ethernet layer.
func (h *DNSHandler) extractSourceMAC(packet gopacket.Packet) net.HardwareAddr {
	ethLayer := packet.Layer(layers.LayerTypeEthernet)
	if eth, ok := ethLayer.(*layers.Ethernet); ok {
		return eth.SrcMAC
	}

	return nil
}

// sendAndLogResponse sends the DNS response and logs the result.
func (h *DNSHandler) sendAndLogResponse(
	response *layers.DNS,
	srcIP, dstIP net.IP,
	srcMAC, dstMAC net.HardwareAddr,
	dstPort layers.UDPPort,
	debugLevel int,
	serial int,
) {
	err := h.SendDNSResponse(response, srcIP, dstIP, srcMAC, dstMAC, dstPort)
	if err != nil {
		if debugLevel >= DebugLevelInfo {
			_, _ = fmt.Fprintf(os.Stdout, "DNS: Failed to send response: %v sn=%d\n", err, serial)
		}

		return
	}

	if debugLevel >= DebugLevelVerbose {
		_, _ = fmt.Fprintf(os.Stdout, "DNS: Sent response with %d answers sn=%d\n",
			len(response.Answers), serial)
	}
}

// lookupHost looks up IP addresses for a hostname.
func (h *DNSHandler) lookupHost(hostname string, set *dnsRecordSet) []dnsRecord {
	h.mu.RLock()
	defer h.mu.RUnlock()

	hostname = strings.ToLower(strings.TrimSuffix(hostname, "."))

	// Device-specific records
	if set != nil {
		if recs, ok := set.forward[hostname]; ok {
			return recs
		}

		if !strings.Contains(hostname, ".") {
			fullname := hostname + "." + h.domain
			if recs, ok := set.forward[fullname]; ok {
				return recs
			}
		}

		return nil
	}

	// Global records
	if recs, ok := h.records[hostname]; ok {
		return recs
	}

	if !strings.Contains(hostname, ".") {
		fullname := hostname + "." + h.domain
		if recs, ok := h.records[fullname]; ok {
			return recs
		}
	}

	return nil
}

// SendDNSResponse sends a DNS response.
func (h *DNSHandler) SendDNSResponse(
	response *layers.DNS,
	srcIP, dstIP net.IP,
	srcMAC, dstMAC net.HardwareAddr,
	dstPort layers.UDPPort,
) error {
	// Build UDP layer
	udp := &layers.UDP{
		SrcPort: dnsPort,
		DstPort: dstPort,
	}

	// Build IP layer
	ip := &layers.IPv4{
		Version:  dnsIPv4Version,
		TTL:      dnsIPv4TTL,
		Protocol: layers.IPProtocolUDP,
		SrcIP:    srcIP,
		DstIP:    dstIP,
	}

	// Build Ethernet layer
	eth := &layers.Ethernet{
		SrcMAC:       srcMAC,
		DstMAC:       dstMAC,
		EthernetType: layers.EthernetTypeIPv4,
	}

	// Serialize packet
	buf := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{
		ComputeChecksums: true,
		FixLengths:       true,
	}

	_ = udp.SetNetworkLayerForChecksum(ip) // error is non-critical for simulation

	err := gopacket.SerializeLayers(buf, opts, eth, ip, udp, response)
	if err != nil {
		return fmt.Errorf("failed to serialize DNS response: %w", err)
	}

	// Send packet
	return h.stack.SendRawPacket(buf.Bytes())
}

// SendDNSResponseV6 sends a DNS response over IPv6.
func (h *DNSHandler) SendDNSResponseV6(
	response *layers.DNS,
	srcIP, dstIP net.IP,
	srcMAC, dstMAC net.HardwareAddr,
	dstPort layers.UDPPort,
) error {
	udp := &layers.UDP{
		SrcPort: dnsPort,
		DstPort: dstPort,
	}

	ip := &layers.IPv6{
		Version:      dnsIPv6Version,
		TrafficClass: 0,
		FlowLabel:    0,
		NextHeader:   layers.IPProtocolUDP,
		HopLimit:     dnsIPv6HopLimit,
		SrcIP:        srcIP,
		DstIP:        dstIP,
	}

	eth := &layers.Ethernet{
		SrcMAC:       srcMAC,
		DstMAC:       dstMAC,
		EthernetType: layers.EthernetTypeIPv6,
	}

	buf := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{
		ComputeChecksums: true,
		FixLengths:       true,
	}

	_ = udp.SetNetworkLayerForChecksum(ip) // error is non-critical for simulation

	err := gopacket.SerializeLayers(buf, opts, eth, ip, udp, response)
	if err != nil {
		return fmt.Errorf("failed to serialize DNS/IPv6 response: %w", err)
	}

	return h.stack.SendRawPacket(buf.Bytes())
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

// buildDNSResponseV6 constructs a DNS response for IPv6 queries.
func (h *DNSHandler) buildDNSResponseV6(
	dns *layers.DNS,
	serverDevice *config.Device,
	debugLevel int,
	serial int,
) *layers.DNS {
	response := &layers.DNS{
		ID:           dns.ID,
		QR:           true,
		OpCode:       dns.OpCode,
		AA:           true,
		TC:           false,
		RD:           dns.RD,
		RA:           true,
		ResponseCode: layers.DNSResponseCodeNoErr,
		Questions:    dns.Questions,
	}

	recordSet := h.getRecordSetForDevice(serverDevice)
	response.Answers, response.ResponseCode = h.resolveQuestions(dns.Questions, recordSet, debugLevel, serial)

	if len(response.Answers) == 0 && debugLevel >= DebugLevelInfo {
		_, _ = fmt.Fprintf(os.Stdout, "DNS/IPv6: NXDOMAIN for queries sn=%d\n", serial)
	} else if len(response.Answers) > 0 && response.ResponseCode == 0 {
		response.ResponseCode = layers.DNSResponseCodeNoErr
	}

	return response
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

// sendAndLogResponseV6 sends the DNS response over IPv6 and logs the result.
func (h *DNSHandler) sendAndLogResponseV6(
	response *layers.DNS,
	srcIP, dstIP net.IP,
	srcMAC, dstMAC net.HardwareAddr,
	dstPort layers.UDPPort,
	debugLevel int,
	serial int,
) {
	err := h.SendDNSResponseV6(response, srcIP, dstIP, srcMAC, dstMAC, dstPort)
	if err != nil && debugLevel >= DebugLevelInfo {
		_, _ = fmt.Fprintf(os.Stdout, "DNS/IPv6: Failed to send response: %v sn=%d\n", err, serial)
	}
}

// dnsResolveContext holds state for resolving DNS questions.
type dnsResolveContext struct {
	answers      []layers.DNSResourceRecord
	responseCode layers.DNSResponseCode
	debugLevel   int
	serial       int
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

func (h *DNSHandler) selectServerDevice(
	devices []*config.Device,
	wantIPv6 bool,
) (*config.Device, net.IP) {
	for _, dev := range devices {
		if !h.deviceHasDNSRecords(dev) {
			continue
		}

		ip := pickIPAddressForDNS(dev, wantIPv6)
		if ip == nil {
			continue
		}

		if len(dev.MACAddress) == 0 {
			continue
		}

		return dev, ip
	}

	return nil, nil
}

func (h *DNSHandler) deviceHasDNSRecords(dev *config.Device) bool {
	if dev == nil {
		return false
	}

	h.mu.RLock()
	_, hasSet := h.deviceRecords[dev]
	hasGlobal := len(h.records) > 0
	h.mu.RUnlock()

	if hasSet {
		return true
	}

	if dev.DNSConfig != nil {
		return true
	}
	// If no device-specific record set exists, fall back to global records
	return dev.DNSConfig == nil && hasGlobal
}

func (h *DNSHandler) getRecordSetForDevice(dev *config.Device) *dnsRecordSet {
	if dev == nil {
		return nil
	}

	h.mu.RLock()
	set := h.deviceRecords[dev]
	h.mu.RUnlock()

	return set
}

func pickIPAddressForDNS(device *config.Device, wantIPv6 bool) net.IP {
	for _, ip := range device.IPAddresses {
		if wantIPv6 {
			if ip.To4() == nil && ip.To16() != nil {
				return ip
			}
		} else if v4 := ip.To4(); v4 != nil {
			return v4
		}
	}

	return nil
}

func (h *DNSHandler) lookupPTR(ip net.IP, set *dnsRecordSet) (dnsPTR, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if set != nil {
		rec, ok := set.reverse[ip.String()]

		return rec, ok
	}

	rec, ok := h.ptrRecords[ip.String()]

	return rec, ok
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

func serverDeviceIP(device *config.Device, wantIPv6 bool) net.IP {
	if device == nil {
		return nil
	}

	for _, ip := range device.IPAddresses {
		if wantIPv6 {
			if ip.To4() == nil && ip.To16() != nil {
				return ip
			}
		} else if v4 := ip.To4(); v4 != nil {
			return v4
		}
	}

	return nil
}

func (h *DNSHandler) sendDNSPayloadV6(
	payload []byte,
	srcIP, dstIP net.IP,
	srcMAC, dstMAC net.HardwareAddr,
	dstPort layers.UDPPort,
) error {
	udp := &layers.UDP{
		SrcPort: dnsPort,
		DstPort: dstPort,
	}
	ip := &layers.IPv6{
		Version:      dnsIPv6Version,
		TrafficClass: 0,
		FlowLabel:    0,
		NextHeader:   layers.IPProtocolUDP,
		HopLimit:     dnsIPv6HopLimit,
		SrcIP:        srcIP,
		DstIP:        dstIP,
	}
	eth := &layers.Ethernet{
		SrcMAC:       srcMAC,
		DstMAC:       dstMAC,
		EthernetType: layers.EthernetTypeIPv6,
	}
	buf := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{
		ComputeChecksums: true,
		FixLengths:       true,
	}

	_ = udp.SetNetworkLayerForChecksum(ip) // error is non-critical for simulation
	err := gopacket.SerializeLayers(buf, opts, eth, ip, udp, gopacket.Payload(payload))
	if err != nil {
		return fmt.Errorf("failed to serialize DNS/IPv6 response: %w", err)
	}

	return h.stack.SendRawPacket(buf.Bytes())
}

// buildNBSTATHeader builds the DNS header for an NBSTAT response.
func buildNBSTATHeader(id uint16) []byte {
	header := make([]byte, dnsHeaderSize)
	binary.BigEndian.PutUint16(header[0:2], id)
	flags := uint16(dnsFlagQR | dnsFlagAA)
	binary.BigEndian.PutUint16(header[2:4], flags)
	binary.BigEndian.PutUint16(header[4:6], 1) // QDCOUNT
	binary.BigEndian.PutUint16(header[6:8], 1) // ANCOUNT

	return header
}

// buildNBSTATQuestion builds the question section for an NBSTAT response.
func buildNBSTATQuestion(q layers.DNSQuestion, encodedQName []byte) []byte {
	question := make([]byte, 0, len(encodedQName)+dnsQuestionExtra+1)
	question = append(question, encodedQName...)
	question = append(question, dnsTerminator)
	question = append(question, byte(q.Type>>dnsByteShift), byte(q.Type))
	question = append(question, byte(q.Class>>dnsByteShift), byte(q.Class))

	return question
}

// buildNBSTATAnswer builds the answer section for an NBSTAT response.
func buildNBSTATAnswer(
	q layers.DNSQuestion,
	names []nbstatNameEntry,
	ownerNodeType uint8,
	macAddr net.HardwareAddr,
) []byte {
	rdLength := 1 + nbstatNameEntrySize*len(names) + nbstatMACAndStatsSize
	answer := make([]byte, 0, dnsHeaderSize+rdLength)

	// NAME pointer to question at offset 12 -> 0xC00C
	answer = append(answer, dnsPointerByte, dnsPointerOffset)
	answer = append(answer, byte(q.Type>>dnsByteShift), byte(q.Type))
	answer = append(answer, byte(q.Class>>dnsByteShift), byte(q.Class))
	// TTL = 0
	answer = append(answer, dnsTerminator, dnsTerminator, dnsTerminator, dnsTerminator)
	answer = append(answer, byte(rdLength>>dnsByteShift), byte(rdLength))

	// RDATA: name count + name entries
	answer = append(answer, byte(len(names)))
	answer = appendNBSTATNameEntries(answer, names, ownerNodeType)
	answer = appendNBSTATMACAndStats(answer, macAddr)

	return answer
}

// appendNBSTATNameEntries appends NetBIOS name entries to the answer buffer.
func appendNBSTATNameEntries(answer []byte, names []nbstatNameEntry, ownerNodeType uint8) []byte {
	for _, name := range names {
		rawName := name.Name
		if len(rawName) > netbiosNameLen {
			rawName = rawName[:netbiosNameLen]
		}

		nameBytes := make([]byte, netbiosNameLen)
		copy(nameBytes, rawName)

		for i := len(rawName); i < netbiosNameLen; i++ {
			nameBytes[i] = ' '
		}

		answer = append(answer, nameBytes...)
		answer = append(answer, name.Suffix)

		nameFlags := uint16(dnsFlagAA) | (uint16(ownerNodeType) << nbstatOwnerTypeShift)
		if name.Group {
			nameFlags |= netbiosGroupFlag
		}

		answer = append(answer, byte(nameFlags>>dnsByteShift), byte(nameFlags))
	}

	return answer
}

// appendNBSTATMACAndStats appends MAC address and statistics to the answer buffer.
func appendNBSTATMACAndStats(answer []byte, macAddr net.HardwareAddr) []byte {
	if len(macAddr) == dnsMACOctets {
		answer = append(answer, macAddr...)
	} else {
		answer = append(answer, []byte{0, 0, 0, 0, 0, 0}...)
	}

	answer = append(answer, make([]byte, netbiosStatsSize)...)

	return answer
}

func (h *DNSHandler) buildNBSTATResponse(
	device *config.Device,
	id uint16,
	q layers.DNSQuestion,
) []byte {
	if device == nil || device.NetBIOSConfig == nil {
		return nil
	}

	service := decodeNBSTATService(q.Name)
	if !isNBSTATServiceSupported(service, device) {
		return nil
	}

	names := netbiosNamesForDevice(device)
	if len(names) == 0 {
		return nil
	}

	header := buildNBSTATHeader(id)
	question := buildNBSTATQuestion(q, encodeDNSName(q.Name))
	answer := buildNBSTATAnswer(q, names, netbiosOwnerNodeType(device), device.MACAddress)

	payload := make([]byte, dnsHeaderSize+len(question)+len(answer))
	copy(payload, header)
	copy(payload[dnsHeaderSize:], question)
	copy(payload[dnsHeaderSize+len(question):], answer)

	return payload
}

func encodeDNSName(name []byte) []byte {
	trimmed := strings.TrimSuffix(string(name), ".")
	if trimmed == "" {
		return []byte{0}
	}

	labels := strings.Split(trimmed, ".")
	buf := make([]byte, 0, len(trimmed)+dnsBufPadding)

	for _, label := range labels {
		if label == "" {
			continue
		}

		buf = append(buf, byte(len(label)))
		buf = append(buf, []byte(label)...)
	}

	return buf
}

func decodeNBSTATService(name []byte) string {
	base := strings.TrimSuffix(string(name), ".")
	if idx := strings.IndexByte(base, '.'); idx != -1 {
		base = base[:idx]
	}

	if len(base) < dnsMinLabelParts {
		return ""
	}

	length := len(base) / nbNibbleEncodingFactor

	decoded := make([]byte, 0, length)

	for i := range length {
		hi := base[2*i] - 'A'

		lo := base[2*i+1] - 'A'

		if hi > nbMaxNibbleValue || lo > nbMaxNibbleValue {
			return ""
		}

		decoded = append(decoded, (hi<<nbNibbleShift)|lo)
	}

	return string(decoded)
}

func isNBSTATServiceSupported(service string, device *config.Device) bool {
	if device == nil || device.NetBIOSConfig == nil {
		return false
	}

	workstation := string(append([]byte{'*'}, make([]byte, netbiosNameLen)...))
	masterBrowser := string(
		[]byte{0x01, 0x02, '_', '_', 'M', 'S', 'B', 'R', 'O', 'W', 'S', 'E', '_', '_', 0x02, 0x01},
	)

	if service == workstation {
		return true
	}

	if service == masterBrowser {
		return device.NetBIOSConfig.MsBrowse
	}

	return false
}

type nbstatNameEntry struct {
	Name   string
	Suffix byte
	Group  bool
}

func netbiosNamesForDevice(device *config.Device) []nbstatNameEntry {
	if device == nil || device.NetBIOSConfig == nil {
		return nil
	}

	cfg := device.NetBIOSConfig
	names := make([]nbstatNameEntry, 0)

	if len(cfg.Names) > 0 {
		for _, n := range cfg.Names {
			names = append(names, nbstatNameEntry{Name: n.Name, Suffix: n.Suffix, Group: n.Group})
		}

		if cfg.MsBrowse {
			names = append(
				names,
				nbstatNameEntry{Name: "__MSBROWSE__", Suffix: nbNameTypeMSBrowse, Group: true},
			)
		}

		return names
	}

	baseName := cfg.Name
	if baseName == "" {
		baseName = device.Name
	}

	for _, svc := range cfg.Services {
		switch strings.ToLower(svc) {
		case "workstation":
			names = append(
				names,
				nbstatNameEntry{Name: baseName, Suffix: nbNameTypeWorkstation, Group: false},
			)
		case "messenger":
			names = append(
				names,
				nbstatNameEntry{Name: baseName, Suffix: nbNameTypeMessenger, Group: false},
			)
		case "fileserver":
			names = append(
				names,
				nbstatNameEntry{Name: baseName, Suffix: nbNameTypeFileServer, Group: false},
			)
		case "domainmaster":
			names = append(
				names,
				nbstatNameEntry{Name: baseName, Suffix: nbNameTypeDomainMaster, Group: true},
			)
		case "masterbrowser":
			names = append(
				names,
				nbstatNameEntry{Name: baseName, Suffix: nbNameTypeMasterBrowser, Group: true},
			)
		case "browser":
			names = append(
				names,
				nbstatNameEntry{Name: baseName, Suffix: nbNameTypeBrowserElec, Group: true},
			)
		case "msbrowse":
			names = append(
				names,
				nbstatNameEntry{Name: "__MSBROWSE__", Suffix: nbNameTypeMSBrowse, Group: true},
			)
		}
	}

	if cfg.MsBrowse {
		names = append(
			names,
			nbstatNameEntry{Name: "__MSBROWSE__", Suffix: nbNameTypeMSBrowse, Group: true},
		)
	}

	return names
}

func netbiosOwnerNodeType(device *config.Device) uint8 {
	if device == nil || device.NetBIOSConfig == nil {
		return nbNodeTypeB
	}

	switch strings.ToUpper(device.NetBIOSConfig.NodeType) {
	case "P":
		return nbNodeTypeP
	case "M":
		return nbNodeTypeM
	case "H":
		return nbNodeTypeH
	default:
		return nbNodeTypeB
	}
}

// isValidDNSName validates DNS name length per RFC 1035
// SECURITY FIX MEDIUM-2: Prevents malformed DNS responses.
func isValidDNSName(name []byte) bool {
	// RFC 1035: Maximum domain name length is 255 bytes
	if len(name) > dnsMaxNameLen {
		return false
	}

	// Validate individual label lengths (max 63 bytes per label)
	labels := strings.SplitSeq(string(name), ".")
	for label := range labels {
		if len(label) > dnsMaxLabelLen {
			return false
		}
	}

	return true
}
