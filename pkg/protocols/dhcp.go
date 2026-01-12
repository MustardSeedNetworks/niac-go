package protocols

import (
	"encoding/binary"
	"fmt"
	"log/slog"
	"net"
	"os"
	"sync"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/krisarmstrong/niac-go/pkg/config"
	"github.com/krisarmstrong/niac-go/pkg/logging"
)

// DHCP message types.
const (
	DHCPDiscover = 1
	DHCPOffer    = 2
	DHCPRequest  = 3
	DHCPDecline  = 4
	DHCPAck      = 5
	DHCPNak      = 6
	DHCPRelease  = 7
	DHCPInform   = 8
)

// DHCP option types not defined in gopacket.
const (
	DHCPOptNTP          layers.DHCPOpt = 42  // NTP servers
	DHCPOptTFTPServer   layers.DHCPOpt = 66  // TFTP server name
	DHCPOptBootfileName layers.DHCPOpt = 67  // Bootfile name
	DHCPOptDomainSearch layers.DHCPOpt = 119 // Domain search list
)

// DHCP lease duration (24 hours).
const DefaultLeaseTime = 24 * time.Hour

// DHCPLease represents an IP address lease.
type DHCPLease struct {
	IP        net.IP
	MAC       net.HardwareAddr
	Hostname  string
	Expiry    time.Time
	LeaseTime time.Duration
}

// DHCPHandler handles DHCP server functionality.
type DHCPHandler struct {
	stack              *Stack
	leases             map[string]*DHCPLease // Key: MAC address string
	ipPool             []net.IP
	poolStart          net.IP
	poolEnd            net.IP
	serverIP           net.IP
	subnetMask         net.IP
	gateway            net.IP
	dnsServers         []net.IP
	domainName         string
	ntpServers         []net.IP // Option 42: NTP servers
	domainSearch       []string // Option 119: Domain search list
	tftpServerName     string   // Option 66: TFTP server name
	bootfileName       string   // Option 67: Bootfile name (for PXE)
	vendorSpecificInfo []byte   // Option 43: Vendor-specific information
	staticLeases       []config.DHCPLease
	mu                 sync.RWMutex
}

// NewDHCPHandler creates a new DHCP handler.
func NewDHCPHandler(stack *Stack) *DHCPHandler {
	return &DHCPHandler{
		stack:      stack,
		leases:     make(map[string]*DHCPLease),
		ipPool:     make([]net.IP, 0),
		subnetMask: net.IPv4(255, 255, 255, 0),
	}
}

// SetPool configures the DHCP IP address pool.
func (h *DHCPHandler) SetPool(start, end net.IP) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.poolStart = start
	h.poolEnd = end

	pool, err := h.generateIPPool(start, end)
	if err != nil {
		// Log error but don't fail initialization - just use empty pool
		fmt.Fprintf(os.Stderr, "Warning: DHCP pool generation failed: %v\n", err)

		h.ipPool = []net.IP{}
	} else {
		h.ipPool = pool
	}
}

// SetServerConfig configures DHCP server parameters.
func (h *DHCPHandler) SetServerConfig(serverIP, gateway net.IP, dnsServers []net.IP, domain string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.serverIP = serverIP
	h.gateway = gateway
	h.dnsServers = dnsServers
	h.domainName = domain
}

// SetAdvancedOptions configures advanced DHCP options.
func (h *DHCPHandler) SetAdvancedOptions(
	ntpServers []net.IP,
	domainSearch []string,
	tftpServer, bootfile string,
	vendorInfo []byte,
) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.ntpServers = ntpServers
	h.domainSearch = domainSearch
	h.tftpServerName = tftpServer
	h.bootfileName = bootfile
	h.vendorSpecificInfo = vendorInfo
}

// Reset clears all DHCP server state while preserving the associated stack.
func (h *DHCPHandler) Reset() {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.leases = make(map[string]*DHCPLease)
	h.ipPool = nil
	h.poolStart = nil
	h.poolEnd = nil
	h.serverIP = nil
	h.subnetMask = net.IPv4(255, 255, 255, 0)
	h.gateway = nil
	h.dnsServers = nil
	h.domainName = ""
	h.ntpServers = nil
	h.domainSearch = nil
	h.tftpServerName = ""
	h.bootfileName = ""
	h.vendorSpecificInfo = nil
	h.staticLeases = nil
}

// SetStaticLeases configures static DHCP leases.
func (h *DHCPHandler) SetStaticLeases(leases []config.DHCPLease) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.staticLeases = leases
}

// MaxPoolSize is the maximum number of IPs allowed in a DHCP pool.
const MaxPoolSize = 65536 // 2^16 IPs (reasonable for simulation)

// generateIPPool creates a list of available IPs
// Returns error if pool size exceeds MaxPoolSize or if range is invalid
// SECURITY FIX MEDIUM-1: Validates range to prevent integer overflow.
func (h *DHCPHandler) generateIPPool(start, end net.IP) ([]net.IP, error) {
	startInt := binary.BigEndian.Uint32(start.To4())
	endInt := binary.BigEndian.Uint32(end.To4())

	// SECURITY: Validate range to prevent integer overflow
	// This check ensures endInt >= startInt before subtraction
	if endInt < startInt {
		return nil, fmt.Errorf("%w: end IP (%s) < start IP (%s)", ErrDHCPPoolInvalid, end, start)
	}

	// Calculate pool size (safe because endInt >= startInt)
	// Using uint64 to prevent overflow even for max range (2^32 - 1)
	size := uint64(endInt) - uint64(startInt) + 1
	if size > MaxPoolSize {
		return nil, fmt.Errorf("%w: size %d exceeds maximum %d (range: %s to %s)",
			ErrDHCPPoolSizeExceeded, size, MaxPoolSize, start, end)
	}

	// Pre-allocate slice with exact capacity
	pool := make([]net.IP, 0, size)

	for i := startInt; i <= endInt; i++ {
		ip := make(net.IP, 4)
		binary.BigEndian.PutUint32(ip, i)
		pool = append(pool, ip)
	}

	return pool, nil
}

// findAvailableIP finds an available IP address
// Note: Caller must hold h.mu lock.
func (h *DHCPHandler) findAvailableIP() net.IP {
	// Check each IP in pool
	for _, ip := range h.ipPool {
		inUse := false

		for _, lease := range h.leases {
			if lease.IP.Equal(ip) && time.Now().Before(lease.Expiry) {
				inUse = true

				break
			}
		}

		if !inUse {
			return ip
		}
	}

	return nil
}

// allocateLease allocates or renews a lease.
func (h *DHCPHandler) allocateLease(mac net.HardwareAddr, requestedIP net.IP, hostname string) (*DHCPLease, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	macStr := mac.String()

	// Check static leases first
	if staticIP := h.matchStaticLease(mac); staticIP != nil {
		if existing, ok := h.leases[macStr]; ok {
			existing.Expiry = time.Now().Add(DefaultLeaseTime)

			existing.IP = staticIP

			if hostname != "" {
				existing.Hostname = hostname
			}

			return existing, nil
		}

		lease := &DHCPLease{
			IP:        staticIP,
			MAC:       mac,
			Hostname:  hostname,
			Expiry:    time.Now().Add(DefaultLeaseTime),
			LeaseTime: DefaultLeaseTime,
		}
		h.leases[macStr] = lease

		return lease, nil
	}

	// Check if client already has a lease
	if existing, ok := h.leases[macStr]; ok {
		// Renew existing lease
		existing.Expiry = time.Now().Add(DefaultLeaseTime)
		// Update hostname if provided
		if hostname != "" {
			existing.Hostname = hostname
		}

		return existing, nil
	}

	// Find available IP
	var ip net.IP
	if requestedIP != nil && h.isIPInPool(requestedIP) && !h.isIPLeased(requestedIP) {
		ip = requestedIP
	} else {
		ip = h.findAvailableIP()
	}

	if ip == nil {
		return nil, ErrNoAvailableIPAddresses
	}

	// Create new lease
	lease := &DHCPLease{
		IP:        ip,
		MAC:       mac,
		Hostname:  hostname,
		Expiry:    time.Now().Add(DefaultLeaseTime),
		LeaseTime: DefaultLeaseTime,
	}

	h.leases[macStr] = lease

	return lease, nil
}

// matchStaticLease finds a matching static lease IP for the given MAC.
func (h *DHCPHandler) matchStaticLease(mac net.HardwareAddr) net.IP {
	for _, lease := range h.staticLeases {
		if lease.MACAddress == nil {
			continue
		}

		if macMatchesMask(mac, lease.MACAddress, lease.MACMask) {
			return lease.ClientIP
		}
	}

	return nil
}

func macMatchesMask(mac, match, mask net.HardwareAddr) bool {
	if len(mac) == 0 || len(match) == 0 {
		return false
	}

	if len(mask) == 0 {
		return mac.String() == match.String()
	}

	if len(mask) != len(mac) || len(match) != len(mac) {
		return false
	}

	for i := range mac {
		if (mac[i] & mask[i]) != (match[i] & mask[i]) {
			return false
		}
	}

	return true
}

// isIPInPool checks if IP is in the pool.
func (h *DHCPHandler) isIPInPool(ip net.IP) bool {
	for _, poolIP := range h.ipPool {
		if poolIP.Equal(ip) {
			return true
		}
	}

	return false
}

// isIPLeased checks if IP is currently leased.
func (h *DHCPHandler) isIPLeased(ip net.IP) bool {
	for _, lease := range h.leases {
		if lease.IP.Equal(ip) && time.Now().Before(lease.Expiry) {
			return true
		}
	}

	return false
}

// HandlePacket processes a DHCP packet.
func (h *DHCPHandler) HandlePacket(pkt *Packet, ipLayer *layers.IPv4, udpLayer *layers.UDP, devices []*config.Device) {
	debugLevel := h.stack.GetDebugLevel()

	h.stack.IncrementStat("dhcp_requests")

	// Parse DHCP layer
	packet := gopacket.NewPacket(pkt.Buffer, layers.LayerTypeEthernet, gopacket.Default)

	dhcpLayer := packet.Layer(layers.LayerTypeDHCPv4)
	if dhcpLayer == nil {
		if debugLevel >= 2 {
			slog.Debug("DHCP packet missing DHCP layer", "sn", pkt.SerialNumber)
		}

		return
	}

	dhcp, ok := dhcpLayer.(*layers.DHCPv4)
	if !ok {
		return
	}

	// Determine DHCP message type
	var messageType uint8

	for _, opt := range dhcp.Options {
		if opt.Type == layers.DHCPOptMessageType && len(opt.Data) > 0 {
			messageType = opt.Data[0]

			break
		}
	}

	if debugLevel >= 3 {
		msgTypeStr := h.dhcpMessageTypeString(messageType)
		slog.Debug(
			"DHCP message received",
			"type",
			msgTypeStr,
			"srcIP",
			ipLayer.SrcIP,
			"mac",
			dhcp.ClientHWAddr,
			"xid",
			dhcp.Xid,
			"sn",
			pkt.SerialNumber,
		)
	}

	// Get source device (DHCP server)
	var serverDevice *config.Device

	for _, dev := range devices {
		if len(dev.IPAddresses) > 0 {
			serverDevice = dev

			break
		}
	}

	if serverDevice == nil {
		if debugLevel >= 2 {
			slog.Debug("DHCP: No server device configured", "sn", pkt.SerialNumber)
		}

		return
	}

	// Extract hostname from options if present (Option 12)
	var hostname string

	for _, opt := range dhcp.Options {
		if opt.Type == layers.DHCPOptHostname && len(opt.Data) > 0 {
			hostname = string(opt.Data)

			break
		}
	}

	// Handle based on message type
	switch messageType {
	case DHCPDiscover:
		// Handle DHCP Discover -> send Offer
		if debugLevel >= 2 {
			slog.Debug("DHCP: Processing Discover", "mac", dhcp.ClientHWAddr, "sn", pkt.SerialNumber)
		}

		// Allocate IP for client
		lease, err := h.allocateLease(dhcp.ClientHWAddr, nil, hostname)
		if err != nil {
			if debugLevel >= 1 {
				logging.ProtocolDebug("DHCP", debugLevel, 1, "Failed to allocate IP: %v sn=%d", err, pkt.SerialNumber)
			}

			return
		}

		// Send DHCP Offer
		if err := h.SendDHCPOffer(
			dhcp.Xid,
			dhcp.ClientHWAddr,
			lease.IP,
			serverDevice.IPAddresses[0],
			serverDevice.MACAddress,
		); err != nil {
			if debugLevel >= 1 {
				logging.ProtocolDebug("DHCP", debugLevel, 1, "Failed to send Offer: %v sn=%d", err, pkt.SerialNumber)
			}
		} else {
			h.stack.IncrementStat("dhcp_offers")

			if debugLevel >= 2 {
				logging.ProtocolDebug(
					"DHCP",
					debugLevel,
					2,
					"Sent Offer IP=%s to %s sn=%d",
					lease.IP,
					dhcp.ClientHWAddr,
					pkt.SerialNumber,
				)
			}
		}

	case DHCPRequest:
		// Handle DHCP Request -> send Ack
		if debugLevel >= 2 {
			slog.Debug("DHCP: Processing Request", "mac", dhcp.ClientHWAddr, "sn", pkt.SerialNumber)
		}

		// Get requested IP from options
		var requestedIP net.IP

		for _, opt := range dhcp.Options {
			if opt.Type == layers.DHCPOptRequestIP && len(opt.Data) == 4 {
				requestedIP = net.IP(opt.Data)

				break
			}
		}

		// Allocate/confirm lease
		lease, err := h.allocateLease(dhcp.ClientHWAddr, requestedIP, hostname)
		if err != nil {
			if debugLevel >= 1 {
				logging.ProtocolDebug("DHCP", debugLevel, 1, "Failed to confirm lease: %v sn=%d", err, pkt.SerialNumber)
			}

			return
		}

		// Send DHCP Ack
		if err := h.SendDHCPAck(
			dhcp.Xid,
			dhcp.ClientHWAddr,
			lease.IP,
			serverDevice.IPAddresses[0],
			serverDevice.MACAddress,
		); err != nil {
			if debugLevel >= 1 {
				logging.ProtocolDebug("DHCP", debugLevel, 1, "Failed to send Ack: %v sn=%d", err, pkt.SerialNumber)
			}
		} else {
			h.stack.IncrementStat("dhcp_acks")
			h.updateFDBTables(dhcp.ClientHWAddr)

			if debugLevel >= 2 {
				logging.ProtocolDebug(
					"DHCP",
					debugLevel,
					2,
					"Sent Ack IP=%s to %s sn=%d",
					lease.IP,
					dhcp.ClientHWAddr,
					pkt.SerialNumber,
				)
			}
		}

	case DHCPRelease:
		if debugLevel >= 2 {
			slog.Debug("DHCP: Release", "mac", dhcp.ClientHWAddr, "sn", pkt.SerialNumber)
		}
		// Remove lease
		h.mu.Lock()
		delete(h.leases, dhcp.ClientHWAddr.String())
		h.mu.Unlock()

	case DHCPInform:
		if debugLevel >= 3 {
			slog.Debug("DHCP: Inform", "mac", dhcp.ClientHWAddr, "sn", pkt.SerialNumber)
		}

	default:
		if debugLevel >= 2 {
			slog.Debug("DHCP: Unhandled message type", "type", messageType, "sn", pkt.SerialNumber)
		}
	}
}

// dhcpMessageTypeString returns string representation of DHCP message type.
func (h *DHCPHandler) dhcpMessageTypeString(msgType uint8) string {
	switch msgType {
	case DHCPDiscover:
		return "DISCOVER"
	case DHCPOffer:
		return "OFFER"
	case DHCPRequest:
		return "REQUEST"
	case DHCPDecline:
		return "DECLINE"
	case DHCPAck:
		return "ACK"
	case DHCPNak:
		return "NAK"
	case DHCPRelease:
		return "RELEASE"
	case DHCPInform:
		return "INFORM"
	default:
		return fmt.Sprintf("UNKNOWN(%d)", msgType)
	}
}

func (h *DHCPHandler) updateFDBTables(mac net.HardwareAddr) {
	if h == nil || h.stack == nil {
		return
	}

	h.stack.updateFDBTables(mac)
}

// SendDHCPOffer sends a DHCP Offer message.
func (h *DHCPHandler) SendDHCPOffer(
	xid uint32,
	clientMAC net.HardwareAddr,
	offeredIP, serverIP net.IP,
	serverMAC net.HardwareAddr,
) error {
	return h.sendDHCPResponse(xid, clientMAC, offeredIP, serverIP, serverMAC, DHCPOffer)
}

// SendDHCPAck sends a DHCP Ack message.
func (h *DHCPHandler) SendDHCPAck(
	xid uint32,
	clientMAC net.HardwareAddr,
	assignedIP, serverIP net.IP,
	serverMAC net.HardwareAddr,
) error {
	return h.sendDHCPResponse(xid, clientMAC, assignedIP, serverIP, serverMAC, DHCPAck)
}

// sendDHCPResponse sends a DHCP Offer or Ack response.
func (h *DHCPHandler) sendDHCPResponse(
	xid uint32,
	clientMAC net.HardwareAddr,
	assignedIP, serverIP net.IP,
	serverMAC net.HardwareAddr,
	msgType uint8,
) error {
	h.mu.RLock()
	defer h.mu.RUnlock()

	// Build DHCP layer
	dhcp := &layers.DHCPv4{
		Operation:    layers.DHCPOpReply,
		HardwareType: layers.LinkTypeEthernet,
		HardwareLen:  6,
		HardwareOpts: 0,
		Xid:          xid,
		Secs:         0,
		Flags:        0x8000, // Broadcast flag
		ClientIP:     net.IPv4zero,
		YourClientIP: assignedIP,
		NextServerIP: net.IPv4zero,
		RelayAgentIP: net.IPv4zero,
		ClientHWAddr: clientMAC,
	}

	// Build DHCP options
	options := []layers.DHCPOption{
		{
			Type:   layers.DHCPOptMessageType,
			Length: 1,
			Data:   []byte{msgType},
		},
		{
			Type:   layers.DHCPOptServerID,
			Length: 4,
			Data:   []byte(serverIP.To4()),
		},
		{
			Type:   layers.DHCPOptLeaseTime,
			Length: 4,
			Data:   h.encodeUint32(uint32(DefaultLeaseTime.Seconds())),
		},
		{
			Type:   layers.DHCPOptSubnetMask,
			Length: 4,
			Data:   []byte(h.subnetMask.To4()),
		},
	}

	// Add router/gateway if configured
	if h.gateway != nil {
		options = append(options, layers.DHCPOption{
			Type:   layers.DHCPOptRouter,
			Length: 4,
			Data:   []byte(h.gateway.To4()),
		})
	}

	// Add DNS servers if configured
	if len(h.dnsServers) > 0 {
		dnsData := make([]byte, 0, len(h.dnsServers)*4)
		for _, dns := range h.dnsServers {
			dnsData = append(dnsData, []byte(dns.To4())...)
		}

		dnsLen := min(len(dnsData),
			// Cap at max uint8
			255)

		options = append(options, layers.DHCPOption{
			Type:   layers.DHCPOptDNS,
			Length: uint8(dnsLen), // #nosec G115 -- dnsLen capped at 255 above
			Data:   dnsData[:dnsLen],
		})
	}

	// Add domain name if configured
	if h.domainName != "" {
		domainLen := min(len(h.domainName), 255)

		options = append(options, layers.DHCPOption{
			Type:   layers.DHCPOptDomainName,
			Length: uint8(domainLen), // #nosec G115 -- domainLen capped at 255 above
			Data:   []byte(h.domainName[:domainLen]),
		})
	}

	// Add renewal time (T1) - 50% of lease time
	options = append(options, layers.DHCPOption{
		Type:   layers.DHCPOptT1,
		Length: 4,
		Data:   h.encodeUint32(uint32(DefaultLeaseTime.Seconds() / 2)),
	})

	// Add rebinding time (T2) - 87.5% of lease time
	options = append(options, layers.DHCPOption{
		Type:   layers.DHCPOptT2,
		Length: 4,
		Data:   h.encodeUint32(uint32(DefaultLeaseTime.Seconds() * 7 / 8)),
	})

	// Add NTP servers if configured (Option 42)
	if len(h.ntpServers) > 0 {
		ntpData := make([]byte, 0, len(h.ntpServers)*4)
		for _, ntp := range h.ntpServers {
			ntpData = append(ntpData, []byte(ntp.To4())...)
		}

		ntpLen := min(len(ntpData), 255)

		options = append(options, layers.DHCPOption{
			Type:   DHCPOptNTP,
			Length: uint8(ntpLen), // #nosec G115 -- ntpLen capped at 255 above
			Data:   ntpData[:ntpLen],
		})
	}

	// Add TFTP server name if configured (Option 66)
	if h.tftpServerName != "" {
		tftpLen := min(len(h.tftpServerName), 255)

		options = append(options, layers.DHCPOption{
			Type:   DHCPOptTFTPServer,
			Length: uint8(tftpLen), // #nosec G115 -- tftpLen capped at 255 above
			Data:   []byte(h.tftpServerName[:tftpLen]),
		})
	}

	// Add bootfile name if configured (Option 67)
	if h.bootfileName != "" {
		bootLen := min(len(h.bootfileName), 255)

		options = append(options, layers.DHCPOption{
			Type:   DHCPOptBootfileName,
			Length: uint8(bootLen), // #nosec G115 -- bootLen capped at 255 above
			Data:   []byte(h.bootfileName[:bootLen]),
		})
	}

	// Add domain search list if configured (Option 119)
	if len(h.domainSearch) > 0 {
		searchData, err := h.encodeDomainSearchList(h.domainSearch)
		if err != nil {
			// Log error but don't fail - just skip this option
			fmt.Fprintf(os.Stderr, "Warning: DHCP domain search encoding failed: %v\n", err)
		} else if len(searchData) > 0 {
			searchLen := min(len(searchData), 255)

			options = append(options, layers.DHCPOption{
				Type:   DHCPOptDomainSearch,
				Length: uint8(searchLen), // #nosec G115 -- searchLen capped at 255 above
				Data:   searchData[:searchLen],
			})
		}
	}

	// Add vendor-specific information if configured (Option 43)
	if len(h.vendorSpecificInfo) > 0 {
		vendorLen := min(len(h.vendorSpecificInfo), 255)

		options = append(options, layers.DHCPOption{
			Type:   layers.DHCPOptVendorOption,
			Length: uint8(vendorLen), // #nosec G115 -- vendorLen capped at 255 above
			Data:   h.vendorSpecificInfo[:vendorLen],
		})
	}

	// Add hostname from lease if available (Option 12)
	if lease, ok := h.leases[clientMAC.String()]; ok && lease.Hostname != "" {
		hostnameLen := min(len(lease.Hostname), 255)

		options = append(options, layers.DHCPOption{
			Type:   layers.DHCPOptHostname,
			Length: uint8(hostnameLen), // #nosec G115 -- hostnameLen capped at 255 above
			Data:   []byte(lease.Hostname[:hostnameLen]),
		})
	}

	// End option
	options = append(options, layers.DHCPOption{
		Type: layers.DHCPOptEnd,
	})

	dhcp.Options = options

	// Build UDP layer
	udp := &layers.UDP{
		SrcPort: 67, // DHCP server port
		DstPort: 68, // DHCP client port
	}

	// Build IP layer
	ip := &layers.IPv4{
		Version:  4,
		TTL:      64,
		Protocol: layers.IPProtocolUDP,
		SrcIP:    serverIP,
		DstIP:    net.IPv4bcast, // Broadcast
	}

	// Build Ethernet layer
	eth := &layers.Ethernet{
		SrcMAC:       serverMAC,
		DstMAC:       net.HardwareAddr{0xff, 0xff, 0xff, 0xff, 0xff, 0xff}, // Broadcast
		EthernetType: layers.EthernetTypeIPv4,
	}

	// Serialize packet
	buf := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{
		ComputeChecksums: true,
		FixLengths:       true,
	}

	_ = udp.SetNetworkLayerForChecksum(ip) // error is non-critical for simulation

	err := gopacket.SerializeLayers(buf, opts, eth, ip, udp, dhcp)
	if err != nil {
		return fmt.Errorf("failed to serialize DHCP response: %w", err)
	}

	// Send packet
	return h.stack.SendRawPacket(buf.Bytes())
}

// encodeUint32 encodes a uint32 as big-endian bytes.
func (h *DHCPHandler) encodeUint32(val uint32) []byte {
	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, val)

	return b
}

// DHCP option constraints (RFC 2132, RFC 3397).
const (
	MaxDHCPOptionLen     = 255 // Maximum DHCP option length
	MaxDomainSearchCount = 10  // Reasonable limit for simulation
	MaxDomainLen         = 253 // RFC 1035: maximum domain name length
)

// encodeDomainSearchList encodes a domain search list in DNS label format (RFC 1035)
// Used for DHCP Option 119 (Domain Search)
// Returns error if constraints are violated.
func (h *DHCPHandler) encodeDomainSearchList(domains []string) ([]byte, error) {
	if len(domains) > MaxDomainSearchCount {
		return nil, fmt.Errorf("%w: %d > %d", ErrTooManyDomainSearchEntries, len(domains), MaxDomainSearchCount)
	}

	result := make([]byte, 0, MaxDHCPOptionLen)

	for _, domain := range domains {
		// Validate domain length
		if len(domain) > MaxDomainLen {
			return nil, fmt.Errorf("%w: %d > %d (domain: %s)", ErrDomainTooLong, len(domain), MaxDomainLen, domain)
		}

		// Split domain into labels (e.g., "example.com" -> ["example", "com"])
		labels := make([]byte, 0, len(domain)+10)

		for _, label := range splitDomain(domain) {
			if len(label) == 0 || len(label) > 63 {
				continue // Invalid label
			}
			// Add label length byte followed by label bytes
			labels = append(labels, byte(len(label)))
			labels = append(labels, []byte(label)...)
		}
		// Add null terminator (0x00)
		labels = append(labels, 0)

		// Check total size before adding
		if len(result)+len(labels) > MaxDHCPOptionLen {
			return nil, fmt.Errorf("%w: %d bytes", ErrDomainSearchListExceedsMaxLen, MaxDHCPOptionLen)
		}

		result = append(result, labels...)
	}

	return result, nil
}

// splitDomain splits a domain name into labels.
func splitDomain(domain string) []string {
	if domain == "" {
		return nil
	}
	// Remove trailing dot if present
	if domain[len(domain)-1] == '.' {
		domain = domain[:len(domain)-1]
	}

	labels := []string{}
	start := 0

	for i := range len(domain) {
		if domain[i] == '.' {
			if i > start {
				labels = append(labels, domain[start:i])
			}

			start = i + 1
		}
	}
	// Add last label
	if start < len(domain) {
		labels = append(labels, domain[start:])
	}

	return labels
}
