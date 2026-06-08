package protocols

import (
	"net"
	"strings"
	"sync"

	"github.com/gopacket/gopacket/layers"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/safeconv"
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
			rcode: layers.DNSResponseCode(safeconv.DNSRCode(rec.RCode)),
		})
	}

	for _, rec := range device.DNSConfig.ReverseRecords {
		set.reverse[rec.IP.String()] = dnsPTR{
			name:  strings.ToLower(strings.TrimSuffix(rec.Name, ".")),
			ttl:   rec.TTL,
			rcode: layers.DNSResponseCode(safeconv.DNSRCode(rec.RCode)),
		}
	}

	h.mu.Lock()
	h.deviceRecords[device] = set
	h.mu.Unlock()
}
