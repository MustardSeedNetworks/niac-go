package protocols

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/krisarmstrong/niac-go/pkg/config"
)

// DNSHandler handles DNS queries and responses
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

// NewDNSHandler creates a new DNS handler
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

// AddRecord adds a DNS A/AAAA record
func (h *DNSHandler) AddRecord(hostname string, ip net.IP) {
	h.AddRecordWithTTL(hostname, ip, 300, layers.DNSResponseCodeNoErr)
}

// AddRecordWithTTL adds a DNS A/AAAA record with TTL and response code.
func (h *DNSHandler) AddRecordWithTTL(hostname string, ip net.IP, ttl uint32, rcode layers.DNSResponseCode) {
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
func (h *DNSHandler) AddPTRRecord(ip net.IP, hostname string, ttl uint32, rcode layers.DNSResponseCode) {
	h.mu.Lock()
	defer h.mu.Unlock()
	hostname = strings.ToLower(strings.TrimSuffix(hostname, "."))
	h.ptrRecords[ip.String()] = dnsPTR{
		name:  hostname,
		ttl:   ttl,
		rcode: rcode,
	}
}

// SetDomain sets the default DNS domain
func (h *DNSHandler) SetDomain(domain string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.domain = domain
}

// LoadDeviceRecords loads DNS records from configured devices
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
			rcode: layers.DNSResponseCode(rec.RCode),
		})
	}
	for _, rec := range device.DNSConfig.ReverseRecords {
		set.reverse[rec.IP.String()] = dnsPTR{
			name:  strings.ToLower(strings.TrimSuffix(rec.Name, ".")),
			ttl:   rec.TTL,
			rcode: layers.DNSResponseCode(rec.RCode),
		}
	}

	h.mu.Lock()
	h.deviceRecords[device] = set
	h.mu.Unlock()
}

// HandleQuery processes a DNS query
func (h *DNSHandler) HandleQuery(pkt *Packet, ipLayer *layers.IPv4, udpLayer *layers.UDP, devices []*config.Device) {
	debugLevel := h.stack.GetDebugLevel()

	// Parse DNS layer
	packet := gopacket.NewPacket(pkt.Buffer, layers.LayerTypeEthernet, gopacket.Default)
	dnsLayer := packet.Layer(layers.LayerTypeDNS)
	if dnsLayer == nil {
		if debugLevel >= 2 {
			fmt.Printf("DNS packet missing DNS layer sn=%d\n", pkt.SerialNumber)
		}
		return
	}

	dns, ok := dnsLayer.(*layers.DNS)
	if !ok {
		return
	}

	h.stack.IncrementStat("dns_queries")

	if debugLevel >= 3 {
		for _, q := range dns.Questions {
			fmt.Printf("DNS Query: %s type=%s class=%s from %s sn=%d\n",
				string(q.Name), q.Type, q.Class, ipLayer.SrcIP, pkt.SerialNumber)
		}
	}

	serverDevice, serverIP := h.selectServerDevice(devices, false)
	if serverDevice == nil || serverIP == nil {
		if debugLevel >= 2 {
			fmt.Printf("DNS: No IPv4 server device/IP configured sn=%d\n", pkt.SerialNumber)
		}
		return
	}

	if len(serverDevice.MACAddress) == 0 {
		if debugLevel >= 2 {
			fmt.Printf("DNS: Server device %s missing MAC address sn=%d\n", serverDevice.Name, pkt.SerialNumber)
		}
		return
	}

	// Build DNS response
	// Handle NBSTAT queries with custom response (gopacket does not support NBSTAT serialization).
	for _, q := range dns.Questions {
		if q.Type == layers.DNSType(33) {
			if err := h.handleNBSTATQuery(pkt, ipLayer, udpLayer, serverDevice, dns.ID, q, packet); err != nil {
				if debugLevel >= 2 {
					fmt.Printf("DNS: NBSTAT handling failed: %v sn=%d\n", err, pkt.SerialNumber)
				}
			}
			return
		}
	}

	response := &layers.DNS{
		ID:           dns.ID,
		QR:           true, // Response
		OpCode:       dns.OpCode,
		AA:           true,  // Authoritative Answer
		TC:           false, // Not truncated
		RD:           dns.RD,
		RA:           true, // Recursion available
		ResponseCode: layers.DNSResponseCodeNoErr,
		Questions:    dns.Questions,
		Answers:      []layers.DNSResourceRecord{},
	}

	recordSet := h.getRecordSetForDevice(serverDevice)
	response.Answers, response.ResponseCode = h.resolveQuestions(dns.Questions, recordSet, debugLevel, pkt.SerialNumber)
	if len(response.Answers) == 0 && debugLevel >= 2 {
		fmt.Printf("DNS: NXDOMAIN for queries sn=%d\n", pkt.SerialNumber)
	} else if len(response.Answers) > 0 {
		if response.ResponseCode == 0 {
			response.ResponseCode = layers.DNSResponseCodeNoErr
		}
	}

	// Get source MAC from Ethernet layer
	ethLayer := packet.Layer(layers.LayerTypeEthernet)
	var srcMAC net.HardwareAddr
	if eth, ok := ethLayer.(*layers.Ethernet); ok {
		srcMAC = eth.SrcMAC
	}

	// Send response
	if err := h.SendDNSResponse(response, serverIP, ipLayer.SrcIP, serverDevice.MACAddress, srcMAC, udpLayer.SrcPort); err != nil {
		if debugLevel >= 1 {
			fmt.Printf("DNS: Failed to send response: %v sn=%d\n", err, pkt.SerialNumber)
		}
	} else if debugLevel >= 3 {
		fmt.Printf("DNS: Sent response with %d answers sn=%d\n", len(response.Answers), pkt.SerialNumber)
	}
}

// lookupHost looks up IP addresses for a hostname
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

// SendDNSResponse sends a DNS response
func (h *DNSHandler) SendDNSResponse(response *layers.DNS, srcIP, dstIP net.IP, srcMAC, dstMAC net.HardwareAddr, dstPort layers.UDPPort) error {
	// Build UDP layer
	udp := &layers.UDP{
		SrcPort: 53,
		DstPort: dstPort,
	}

	// Build IP layer
	ip := &layers.IPv4{
		Version:  4,
		TTL:      64,
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

	udp.SetNetworkLayerForChecksum(ip) // #nosec G104 -- error logged or non-critical

	if err := gopacket.SerializeLayers(buf, opts, eth, ip, udp, response); err != nil {
		return fmt.Errorf("failed to serialize DNS response: %w", err)
	}

	// Send packet
	return h.stack.SendRawPacket(buf.Bytes())
}

// SendDNSResponseV6 sends a DNS response over IPv6.
func (h *DNSHandler) SendDNSResponseV6(response *layers.DNS, srcIP, dstIP net.IP, srcMAC, dstMAC net.HardwareAddr, dstPort layers.UDPPort) error {
	udp := &layers.UDP{
		SrcPort: 53,
		DstPort: dstPort,
	}

	ip := &layers.IPv6{
		Version:      6,
		TrafficClass: 0,
		FlowLabel:    0,
		NextHeader:   layers.IPProtocolUDP,
		HopLimit:     64,
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

	udp.SetNetworkLayerForChecksum(ip) // #nosec G104 -- error logged or non-critical

	if err := gopacket.SerializeLayers(buf, opts, eth, ip, udp, response); err != nil {
		return fmt.Errorf("failed to serialize DNS/IPv6 response: %w", err)
	}

	return h.stack.SendRawPacket(buf.Bytes())
}

// HandleQueryV6 processes a DNS query over IPv6
func (h *DNSHandler) HandleQueryV6(pkt *Packet, packet gopacket.Packet, ipv6 *layers.IPv6, udpLayer *layers.UDP, devices []*config.Device) {
	debugLevel := h.stack.GetDebugLevel()

	dnsLayer := packet.Layer(layers.LayerTypeDNS)
	if dnsLayer == nil {
		if debugLevel >= 2 {
			fmt.Printf("DNS/IPv6 packet missing DNS layer sn=%d\n", pkt.SerialNumber)
		}
		return
	}

	dns, ok := dnsLayer.(*layers.DNS)
	if !ok {
		return
	}

	h.stack.IncrementStat("dns_queries")

	serverDevice, serverIP := h.selectServerDevice(devices, true)
	if serverDevice == nil || serverIP == nil {
		if debugLevel >= 2 {
			fmt.Printf("DNS/IPv6: No server device/IP configured sn=%d\n", pkt.SerialNumber)
		}
		return
	}
	if len(serverDevice.MACAddress) == 0 {
		if debugLevel >= 2 {
			fmt.Printf("DNS/IPv6: Server device %s missing MAC address sn=%d\n", serverDevice.Name, pkt.SerialNumber)
		}
		return
	}

	// Handle NBSTAT queries with custom response
	for _, q := range dns.Questions {
		if q.Type == layers.DNSType(33) {
			if err := h.handleNBSTATQueryV6(pkt, ipv6, udpLayer, serverDevice, dns.ID, q, packet); err != nil {
				if debugLevel >= 2 {
					fmt.Printf("DNS/IPv6: NBSTAT handling failed: %v sn=%d\n", err, pkt.SerialNumber)
				}
			}
			return
		}
	}

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
	response.Answers, response.ResponseCode = h.resolveQuestions(dns.Questions, recordSet, debugLevel, pkt.SerialNumber)
	if len(response.Answers) == 0 {
		if debugLevel >= 2 {
			fmt.Printf("DNS/IPv6: NXDOMAIN for queries sn=%d\n", pkt.SerialNumber)
		}
	} else {
		if response.ResponseCode == 0 {
			response.ResponseCode = layers.DNSResponseCodeNoErr
		}
	}

	ethLayer := packet.Layer(layers.LayerTypeEthernet)
	if ethLayer == nil {
		if debugLevel >= 2 {
			fmt.Printf("DNS/IPv6: Missing Ethernet layer sn=%d\n", pkt.SerialNumber)
		}
		return
	}
	dstMAC := ethLayer.(*layers.Ethernet).SrcMAC

	if err := h.SendDNSResponseV6(response, serverIP, ipv6.SrcIP, serverDevice.MACAddress, dstMAC, udpLayer.SrcPort); err != nil {
		if debugLevel >= 1 {
			fmt.Printf("DNS/IPv6: Failed to send response: %v sn=%d\n", err, pkt.SerialNumber)
		}
	}
}

func (h *DNSHandler) resolveQuestions(questions []layers.DNSQuestion, set *dnsRecordSet, debugLevel int, serial int) ([]layers.DNSResourceRecord, layers.DNSResponseCode) {
	answers := make([]layers.DNSResourceRecord, 0, len(questions))
	responseCode := layers.DNSResponseCodeNoErr

	for _, q := range questions {
		// SECURITY FIX MEDIUM-2: Validate DNS name length per RFC 1035
		// Maximum domain name length is 255 bytes total
		// Maximum label length is 63 bytes
		if !isValidDNSName(q.Name) {
			if debugLevel >= 2 {
				fmt.Printf("DNS: Invalid domain name length (> 255 or label > 63): %s sn=%d\n", q.Name, serial)
			}
			continue // Skip invalid names
		}

		hostname := strings.ToLower(strings.TrimSuffix(string(q.Name), "."))
		switch q.Type {
		case layers.DNSTypeA:
			for _, rec := range h.lookupHost(hostname, set) {
				if rec.ip.To4() != nil {
					answers = append(answers, layers.DNSResourceRecord{
						Name:  q.Name,
						Type:  layers.DNSTypeA,
						Class: layers.DNSClassIN,
						TTL:   rec.ttl,
						IP:    rec.ip,
					})
					if rec.rcode != layers.DNSResponseCodeNoErr {
						responseCode = rec.rcode
					}
					if debugLevel >= 2 {
						fmt.Printf("DNS: %s -> %s (A record) sn=%d\n", hostname, rec.ip, serial)
					}
				}
			}
		case layers.DNSTypeAAAA:
			for _, rec := range h.lookupHost(hostname, set) {
				if rec.ip.To4() == nil && rec.ip.To16() != nil {
					answers = append(answers, layers.DNSResourceRecord{
						Name:  q.Name,
						Type:  layers.DNSTypeAAAA,
						Class: layers.DNSClassIN,
						TTL:   rec.ttl,
						IP:    rec.ip,
					})
					if rec.rcode != layers.DNSResponseCodeNoErr {
						responseCode = rec.rcode
					}
					if debugLevel >= 2 {
						fmt.Printf("DNS: %s -> %s (AAAA record) sn=%d\n", hostname, rec.ip, serial)
					}
				}
			}
		case layers.DNSTypePTR:
			if ip, ok, isV6 := parsePTRName(q.Name); ok {
				if host, found := h.lookupPTR(ip, set); found && host.name != "" {
					ptr := host.name
					if !strings.HasSuffix(ptr, ".") {
						ptr += "."
					}
					answers = append(answers, layers.DNSResourceRecord{
						Name:  q.Name,
						Type:  layers.DNSTypePTR,
						Class: layers.DNSClassIN,
						TTL:   host.ttl,
						PTR:   []byte(ptr),
					})
					if host.rcode != layers.DNSResponseCodeNoErr {
						responseCode = host.rcode
					}
					if debugLevel >= 2 {
						fmt.Printf("DNS: %s -> %s (PTR record) sn=%d\n", q.Name, ptr, serial)
					}
				} else if isV6 {
					responseCode = layers.DNSResponseCodeNXDomain
				}
			} else if debugLevel >= 2 {
				fmt.Printf("DNS: PTR query %s could not be parsed sn=%d\n", q.Name, serial)
			}
		case layers.DNSType(33): // NBSTAT (NetBIOS)
			// NBSTAT handled separately.
		}
	}

	if len(answers) == 0 {
		if responseCode == layers.DNSResponseCodeNoErr {
			responseCode = layers.DNSResponseCodeNXDomain
		}
		return answers, responseCode
	}
	return answers, responseCode
}

func (h *DNSHandler) selectServerDevice(devices []*config.Device, wantIPv6 bool) (*config.Device, net.IP) {
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
	if len(parts) != 4 {
		return nil, false
	}

	ip := net.IPv4(0, 0, 0, 0).To4()
	for i := 0; i < 4; i++ {
		val, err := strconv.Atoi(parts[len(parts)-1-i])
		if err != nil || val < 0 || val > 255 {
			return nil, false
		}
		ip[i] = byte(val)
	}
	return ip, true
}

func parseIPv6PTRName(name string) (net.IP, bool) {
	base := strings.TrimSuffix(name, ".ip6.arpa")
	nibbles := strings.Split(strings.Trim(base, "."), ".")
	if len(nibbles) != 32 {
		return nil, false
	}

	var builder strings.Builder
	builder.Grow(32)
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

func (h *DNSHandler) handleNBSTATQuery(pkt *Packet, ipLayer *layers.IPv4, udpLayer *layers.UDP, serverDevice *config.Device, id uint16, q layers.DNSQuestion, packet gopacket.Packet) error {
	payload, err := h.buildNBSTATResponse(serverDevice, id, q)
	if err != nil || len(payload) == 0 {
		return err
	}
	ethLayer := packet.Layer(layers.LayerTypeEthernet)
	var dstMAC net.HardwareAddr
	if eth, ok := ethLayer.(*layers.Ethernet); ok {
		dstMAC = eth.SrcMAC
	}
	return h.stack.udpHandler.SendUDP(serverDeviceIP(serverDevice, false), ipLayer.SrcIP, 53, uint16(udpLayer.SrcPort), payload, []byte(serverDevice.MACAddress), []byte(dstMAC))
}

func (h *DNSHandler) handleNBSTATQueryV6(pkt *Packet, ipv6 *layers.IPv6, udpLayer *layers.UDP, serverDevice *config.Device, id uint16, q layers.DNSQuestion, packet gopacket.Packet) error {
	payload, err := h.buildNBSTATResponse(serverDevice, id, q)
	if err != nil || len(payload) == 0 {
		return err
	}
	ethLayer := packet.Layer(layers.LayerTypeEthernet)
	var dstMAC net.HardwareAddr
	if eth, ok := ethLayer.(*layers.Ethernet); ok {
		dstMAC = eth.SrcMAC
	}
	return h.sendDNSPayloadV6(payload, serverDeviceIP(serverDevice, true), ipv6.SrcIP, serverDevice.MACAddress, dstMAC, udpLayer.SrcPort)
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

func (h *DNSHandler) sendDNSPayloadV6(payload []byte, srcIP, dstIP net.IP, srcMAC, dstMAC net.HardwareAddr, dstPort layers.UDPPort) error {
	udp := &layers.UDP{
		SrcPort: 53,
		DstPort: dstPort,
	}
	ip := &layers.IPv6{
		Version:      6,
		TrafficClass: 0,
		FlowLabel:    0,
		NextHeader:   layers.IPProtocolUDP,
		HopLimit:     64,
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
	udp.SetNetworkLayerForChecksum(ip) // #nosec G104 -- error logged or non-critical
	if err := gopacket.SerializeLayers(buf, opts, eth, ip, udp, gopacket.Payload(payload)); err != nil {
		return fmt.Errorf("failed to serialize DNS/IPv6 response: %w", err)
	}
	return h.stack.SendRawPacket(buf.Bytes())
}

func (h *DNSHandler) buildNBSTATResponse(device *config.Device, id uint16, q layers.DNSQuestion) ([]byte, error) {
	if device == nil {
		return nil, nil
	}
	// Only answer NBSTAT for NetBIOS-enabled devices.
	if device.NetBIOSConfig == nil {
		return nil, nil
	}

	service := decodeNBSTATService(q.Name)
	if !isNBSTATServiceSupported(service, device) {
		return nil, nil
	}

	names := netbiosNamesForDevice(device)
	if len(names) == 0 {
		return nil, nil
	}

	encodedQName := encodeDNSName(q.Name)

	// Build header
	header := make([]byte, 12)
	binary.BigEndian.PutUint16(header[0:2], id)
	// We'll fill ID later from caller if needed.
	flags := uint16(0x8000 | 0x0400) // QR | AA
	binary.BigEndian.PutUint16(header[2:4], flags)
	binary.BigEndian.PutUint16(header[4:6], 1) // QDCOUNT
	binary.BigEndian.PutUint16(header[6:8], 1) // ANCOUNT
	// NSCOUNT, ARCOUNT = 0

	// Question section
	question := make([]byte, 0, len(encodedQName)+4)
	question = append(question, encodedQName...)
	question = append(question, 0x00) // terminator
	question = append(question, byte(q.Type>>8), byte(q.Type))
	question = append(question, byte(q.Class>>8), byte(q.Class))

	// Answer section
	rdLength := 1 + (16+2)*len(names) + 46
	answer := make([]byte, 0, 12+rdLength)
	// NAME pointer to question at offset 12 -> 0xC00C
	answer = append(answer, 0xC0, 0x0C)
	// TYPE and CLASS
	answer = append(answer, byte(q.Type>>8), byte(q.Type))
	answer = append(answer, byte(q.Class>>8), byte(q.Class))
	// TTL = 0
	answer = append(answer, 0x00, 0x00, 0x00, 0x00)
	// RDLENGTH
	answer = append(answer, byte(rdLength>>8), byte(rdLength))

	// RDATA
	answer = append(answer, byte(len(names)))

	ownerNodeType := netbiosOwnerNodeType(device)
	for _, name := range names {
		// 15-char name padded with spaces
		rawName := name.Name
		if len(rawName) > 15 {
			rawName = rawName[:15]
		}
		nameBytes := make([]byte, 15)
		copy(nameBytes, rawName)
		for i := len(rawName); i < 15; i++ {
			nameBytes[i] = ' '
		}
		answer = append(answer, nameBytes...)
		answer = append(answer, name.Suffix)
		flags := uint16(0x0400 | (uint16(ownerNodeType) << 13))
		if name.Group {
			flags |= 0x8000
		}
		answer = append(answer, byte(flags>>8), byte(flags))
	}

	// MAC address (6 bytes)
	if len(device.MACAddress) == 6 {
		answer = append(answer, device.MACAddress...)
	} else {
		answer = append(answer, []byte{0, 0, 0, 0, 0, 0}...)
	}
	// Statistics (40 bytes)
	answer = append(answer, make([]byte, 40)...)

	// Assemble full payload
	payload := append(header, question...)
	payload = append(payload, answer...)
	return payload, nil
}

func encodeDNSName(name []byte) []byte {
	trimmed := strings.TrimSuffix(string(name), ".")
	if trimmed == "" {
		return []byte{0}
	}
	labels := strings.Split(trimmed, ".")
	buf := make([]byte, 0, len(trimmed)+2)
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
	if len(base) < 2 {
		return ""
	}
	length := len(base) / 2
	decoded := make([]byte, 0, length)
	for i := 0; i < length; i++ {
		hi := base[2*i] - 'A'
		lo := base[2*i+1] - 'A'
		if hi > 0x0F || lo > 0x0F {
			return ""
		}
		decoded = append(decoded, (hi<<4)|lo)
	}
	return string(decoded)
}

func isNBSTATServiceSupported(service string, device *config.Device) bool {
	if device == nil || device.NetBIOSConfig == nil {
		return false
	}
	workstation := string(append([]byte{'*'}, make([]byte, 15)...))
	masterBrowser := string([]byte{0x01, 0x02, '_', '_', 'M', 'S', 'B', 'R', 'O', 'W', 'S', 'E', '_', '_', 0x02, 0x01})
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
			names = append(names, nbstatNameEntry{Name: "__MSBROWSE__", Suffix: 0x01, Group: true})
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
			names = append(names, nbstatNameEntry{Name: baseName, Suffix: 0x00, Group: false})
		case "messenger":
			names = append(names, nbstatNameEntry{Name: baseName, Suffix: 0x03, Group: false})
		case "fileserver":
			names = append(names, nbstatNameEntry{Name: baseName, Suffix: 0x20, Group: false})
		case "domainmaster":
			names = append(names, nbstatNameEntry{Name: baseName, Suffix: 0x1B, Group: true})
		case "masterbrowser":
			names = append(names, nbstatNameEntry{Name: baseName, Suffix: 0x1D, Group: true})
		case "browser":
			names = append(names, nbstatNameEntry{Name: baseName, Suffix: 0x1E, Group: true})
		case "msbrowse":
			names = append(names, nbstatNameEntry{Name: "__MSBROWSE__", Suffix: 0x01, Group: true})
		}
	}
	if cfg.MsBrowse {
		names = append(names, nbstatNameEntry{Name: "__MSBROWSE__", Suffix: 0x01, Group: true})
	}
	return names
}

func netbiosOwnerNodeType(device *config.Device) uint8 {
	if device == nil || device.NetBIOSConfig == nil {
		return 0
	}
	switch strings.ToUpper(device.NetBIOSConfig.NodeType) {
	case "P":
		return 1
	case "M":
		return 2
	case "H":
		return 3
	default:
		return 0
	}
}

// isValidDNSName validates DNS name length per RFC 1035
// SECURITY FIX MEDIUM-2: Prevents malformed DNS responses
func isValidDNSName(name []byte) bool {
	// RFC 1035: Maximum domain name length is 255 bytes
	if len(name) > 255 {
		return false
	}

	// Validate individual label lengths (max 63 bytes per label)
	labels := strings.Split(string(name), ".")
	for _, label := range labels {
		if len(label) > 63 {
			return false
		}
	}

	return true
}
