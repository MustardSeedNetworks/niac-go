package protocols

import (
	"fmt"
	"net"
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

	// Find server device to respond from
	var serverDevice *config.Device
	for _, dev := range devices {
		if len(dev.IPAddresses) > 0 {
			serverDevice = dev
			break
		}
	}

	if serverDevice == nil {
		if debugLevel >= 2 {
			fmt.Printf("DNS: No server device configured sn=%d\n", pkt.SerialNumber)
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

	// Process each question
	for _, q := range dns.Questions {
		hostname := strings.ToLower(strings.TrimSuffix(string(q.Name), "."))

		switch q.Type {
		case layers.DNSTypeA:
			// A record query (IPv4)
			ips := h.lookupHost(hostname)
			for _, ip := range ips {
				if ip.To4() != nil {
					response.Answers = append(response.Answers, layers.DNSResourceRecord{
						Name:  q.Name,
						Type:  layers.DNSTypeA,
						Class: layers.DNSClassIN,
						TTL:   300,
						IP:    ip,
					})
					if debugLevel >= 2 {
						fmt.Printf("DNS: %s -> %s (A record) sn=%d\n", hostname, ip, pkt.SerialNumber)
					}
				}
			}

		case layers.DNSTypeAAAA:
			// AAAA record query (IPv6)
			ips := h.lookupHost(hostname)
			for _, ip := range ips {
				if ip.To4() == nil && ip.To16() != nil {
					response.Answers = append(response.Answers, layers.DNSResourceRecord{
						Name:  q.Name,
						Type:  layers.DNSTypeAAAA,
						Class: layers.DNSClassIN,
						TTL:   300,
						IP:    ip,
					})
					if debugLevel >= 2 {
						fmt.Printf("DNS: %s -> %s (AAAA record) sn=%d\n", hostname, ip, pkt.SerialNumber)
					}
				}
			}

		case layers.DNSTypePTR:
			// PTR record query (reverse lookup)
			// Pending: parse reverse DNS (in-addr.arpa/ip6.arpa) - see issue #80
			if debugLevel >= 2 {
				fmt.Printf("DNS: PTR query for %s (reverse lookup pending, issue #80) sn=%d\n", hostname, pkt.SerialNumber)
			}
		}
	}

	// If no answers found, return NXDOMAIN
	if len(response.Answers) == 0 {
		response.ResponseCode = layers.DNSResponseCodeNXDomain
		if debugLevel >= 2 {
			fmt.Printf("DNS: NXDOMAIN for queries sn=%d\n", pkt.SerialNumber)
		}
	}

	// Get source MAC from Ethernet layer
	ethLayer := packet.Layer(layers.LayerTypeEthernet)
	var srcMAC net.HardwareAddr
	if eth, ok := ethLayer.(*layers.Ethernet); ok {
		srcMAC = eth.SrcMAC
	}

	// Send response
	if err := h.SendDNSResponse(response, serverDevice.IPAddresses[0], ipLayer.SrcIP, serverDevice.MACAddress, srcMAC, udpLayer.SrcPort); err != nil {
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

// HandleQueryV6 processes a DNS query over IPv6
func (h *DNSHandler) HandleQueryV6(pkt *Packet, packet gopacket.Packet, ipv6 *layers.IPv6, udpLayer *layers.UDP, devices []*config.Device) {
	debugLevel := h.stack.GetDebugLevel()

	if debugLevel >= 2 {
		fmt.Printf("DNS/IPv6 query from [%s] (stub - not fully implemented)\n", ipv6.SrcIP)
	}

	// Pending: implement full DNS over IPv6 (issue #80)
}
