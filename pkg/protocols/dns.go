package protocols

import (
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
	stack      *Stack
	records    map[string][]net.IP // Hostname -> IPs
	ptrRecords map[string]string   // IP -> Hostname (reverse lookup)
	mu         sync.RWMutex
	domain     string // Default domain
}

// NewDNSHandler creates a new DNS handler
func NewDNSHandler(stack *Stack) *DNSHandler {
	return &DNSHandler{
		stack:      stack,
		records:    make(map[string][]net.IP),
		ptrRecords: make(map[string]string),
		domain:     "local",
	}
}

// AddRecord adds a DNS A/AAAA record
func (h *DNSHandler) AddRecord(hostname string, ip net.IP) {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Normalize hostname
	hostname = strings.ToLower(strings.TrimSuffix(hostname, "."))

	// Add forward record
	h.records[hostname] = append(h.records[hostname], ip)

	// Add reverse record (PTR)
	h.ptrRecords[ip.String()] = hostname
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

	response.Answers, response.ResponseCode = h.resolveQuestions(dns.Questions, pkt.SerialNumber, debugLevel)
	if len(response.Answers) == 0 && debugLevel >= 2 {
		fmt.Printf("DNS: NXDOMAIN for queries sn=%d\n", pkt.SerialNumber)
	} else if len(response.Answers) > 0 {
		response.ResponseCode = layers.DNSResponseCodeNoErr
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
func (h *DNSHandler) lookupHost(hostname string) []net.IP {
	h.mu.RLock()
	defer h.mu.RUnlock()

	hostname = strings.ToLower(strings.TrimSuffix(hostname, "."))

	// Try exact match first
	if ips, ok := h.records[hostname]; ok {
		return ips
	}

	// Try with default domain
	if !strings.Contains(hostname, ".") {
		fullname := hostname + "." + h.domain
		if ips, ok := h.records[fullname]; ok {
			return ips
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

	udp.SetNetworkLayerForChecksum(ip)

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

	udp.SetNetworkLayerForChecksum(ip)

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

	response.Answers, response.ResponseCode = h.resolveQuestions(dns.Questions, pkt.SerialNumber, debugLevel)
	if len(response.Answers) == 0 {
		if debugLevel >= 2 {
			fmt.Printf("DNS/IPv6: NXDOMAIN for queries sn=%d\n", pkt.SerialNumber)
		}
	} else {
		response.ResponseCode = layers.DNSResponseCodeNoErr
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

func (h *DNSHandler) resolveQuestions(questions []layers.DNSQuestion, debugLevel int, serial int) ([]layers.DNSResourceRecord, layers.DNSResponseCode) {
	answers := make([]layers.DNSResourceRecord, 0, len(questions))

	for _, q := range questions {
		hostname := strings.ToLower(strings.TrimSuffix(string(q.Name), "."))
		switch q.Type {
		case layers.DNSTypeA:
			for _, ip := range h.lookupHost(hostname) {
				if ip.To4() != nil {
					answers = append(answers, layers.DNSResourceRecord{
						Name:  q.Name,
						Type:  layers.DNSTypeA,
						Class: layers.DNSClassIN,
						TTL:   300,
						IP:    ip,
					})
					if debugLevel >= 2 {
						fmt.Printf("DNS: %s -> %s (A record) sn=%d\n", hostname, ip, serial)
					}
				}
			}
		case layers.DNSTypeAAAA:
			for _, ip := range h.lookupHost(hostname) {
				if ip.To4() == nil && ip.To16() != nil {
					answers = append(answers, layers.DNSResourceRecord{
						Name:  q.Name,
						Type:  layers.DNSTypeAAAA,
						Class: layers.DNSClassIN,
						TTL:   300,
						IP:    ip,
					})
					if debugLevel >= 2 {
						fmt.Printf("DNS: %s -> %s (AAAA record) sn=%d\n", hostname, ip, serial)
					}
				}
			}
		case layers.DNSTypePTR:
			if ip, ok := parsePTRName(q.Name); ok {
				if host := h.lookupPTR(ip); host != "" {
					ptr := host
					if !strings.HasSuffix(ptr, ".") {
						ptr += "."
					}
					answers = append(answers, layers.DNSResourceRecord{
						Name:  q.Name,
						Type:  layers.DNSTypePTR,
						Class: layers.DNSClassIN,
						TTL:   300,
						PTR:   []byte(ptr),
					})
					if debugLevel >= 2 {
						fmt.Printf("DNS: %s -> %s (PTR record) sn=%d\n", q.Name, ptr, serial)
					}
				}
			} else if debugLevel >= 2 {
				fmt.Printf("DNS: PTR query %s could not be parsed sn=%d\n", q.Name, serial)
			}
		}
	}

	if len(answers) == 0 {
		return answers, layers.DNSResponseCodeNXDomain
	}
	return answers, layers.DNSResponseCodeNoErr
}

func (h *DNSHandler) selectServerDevice(devices []*config.Device, wantIPv6 bool) (*config.Device, net.IP) {
	for _, dev := range devices {
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

func (h *DNSHandler) lookupPTR(ip net.IP) string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.ptrRecords[ip.String()]
}

func parsePTRName(name []byte) (net.IP, bool) {
	ptrName := strings.ToLower(strings.TrimSuffix(string(name), "."))
	switch {
	case strings.HasSuffix(ptrName, ".in-addr.arpa"):
		return parseIPv4PTRName(ptrName)
	case strings.HasSuffix(ptrName, ".ip6.arpa"):
		return parseIPv6PTRName(ptrName)
	default:
		return nil, false
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
