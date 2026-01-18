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

// DefaultLeaseTime is the DHCP lease duration (24 hours).
const DefaultLeaseTime = 24 * time.Hour

// DHCP encoding constants.
const (
	dhcpMaxByte          = 255    // max byte value for subnet mask octets / max DHCP option length
	dhcpIPv4Len          = 4      // IPv4 address length in bytes (also used for uint32 encoding)
	dhcpHWAddrLen        = 6      // Ethernet hardware address length
	dhcpBroadcastFlag    = 0x8000 // DHCP broadcast flag
	dhcpMinParts         = 2      // minimum parts for option parsing
	dhcpBitShift         = 8      // bit shift for uint32 encoding
	dhcpDNSCountMax      = 10     // max DNS servers to include in response
	debugLevelInfo       = 2      // debug level for info/basic logging
	t1RenewalDivisor     = 2      // T1 renewal time = lease time / 2
	t2RebindNumerator    = 7      // T2 rebind time = lease time * 7 / 8
	t2RebindDivisor      = 8      // T2 rebind time divisor
	domainLabelBufferPad = 10     // extra buffer padding for domain label encoding
	maxDomainLabelLen    = 63     // RFC 1035: maximum domain label length
	dhcpServerPort       = 67     // DHCP server port
	dhcpClientPort       = 68     // DHCP client port
	ipv4Version          = 4      // IPv4 version field
	ipv4DefaultTTL       = 64     // IPv4 default time to live
)

// getDefaultSubnetMask returns the default /24 subnet mask.
func getDefaultSubnetMask() net.IP {
	return net.IPv4(dhcpMaxByte, dhcpMaxByte, dhcpMaxByte, 0)
}

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
		subnetMask: getDefaultSubnetMask(),
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
	h.subnetMask = getDefaultSubnetMask()
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
		ip := make(net.IP, dhcpIPv4Len)
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

// dhcpPacketInfo holds parsed DHCP packet information.
type dhcpPacketInfo struct {
	dhcp        *layers.DHCPv4
	messageType uint8
	hostname    string
}

// parseDHCPPacket parses a DHCP packet and extracts relevant information.
func (h *DHCPHandler) parseDHCPPacket(pkt *Packet) *dhcpPacketInfo {
	packet := gopacket.NewPacket(pkt.Buffer, layers.LayerTypeEthernet, gopacket.Default)

	dhcpLayer := packet.Layer(layers.LayerTypeDHCPv4)
	if dhcpLayer == nil {
		return nil
	}

	dhcp, ok := dhcpLayer.(*layers.DHCPv4)
	if !ok {
		return nil
	}

	info := &dhcpPacketInfo{dhcp: dhcp}

	// Extract message type and hostname from options
	for _, opt := range dhcp.Options {
		//exhaustive:ignore
		switch opt.Type {
		case layers.DHCPOptMessageType:
			if len(opt.Data) > 0 {
				info.messageType = opt.Data[0]
			}
		case layers.DHCPOptHostname:
			if len(opt.Data) > 0 {
				info.hostname = string(opt.Data)
			}
		default:
			// Ignore other DHCP options - only interested in message type and hostname
		}
	}

	return info
}

// findServerDevice finds a suitable server device from the device list.
func findServerDevice(devices []*config.Device) *config.Device {
	for _, dev := range devices {
		if len(dev.IPAddresses) > 0 {
			return dev
		}
	}

	return nil
}

// getRequestedIP extracts the requested IP from DHCP options.
func getRequestedIP(dhcp *layers.DHCPv4) net.IP {
	for _, opt := range dhcp.Options {
		if opt.Type == layers.DHCPOptRequestIP && len(opt.Data) == 4 {
			return net.IP(opt.Data)
		}
	}

	return nil
}

// handleDHCPDiscover handles a DHCP Discover message.
func (h *DHCPHandler) handleDHCPDiscover(
	info *dhcpPacketInfo,
	serverDevice *config.Device,
	serialNum int,
	debugLevel int,
) {
	logger := slog.Default()

	if debugLevel >= DebugLevelInfo {
		logger.Debug("DHCP: Processing Discover", "mac", info.dhcp.ClientHWAddr, "sn", serialNum)
	}

	lease, err := h.allocateLease(info.dhcp.ClientHWAddr, nil, info.hostname)
	if err != nil {
		if debugLevel >= 1 {
			logging.ProtocolDebugf("DHCP", debugLevel, 1, "Failed to allocate IP: %v sn=%d", err, serialNum)
		}

		return
	}

	offerErr := h.SendDHCPOffer(
		info.dhcp.Xid,
		info.dhcp.ClientHWAddr,
		lease.IP,
		serverDevice.IPAddresses[0],
		serverDevice.MACAddress,
	)
	if offerErr != nil {
		if debugLevel >= 1 {
			logging.ProtocolDebugf("DHCP", debugLevel, 1, "Failed to send Offer: %v sn=%d", offerErr, serialNum)
		}

		return
	}

	h.stack.IncrementStat("dhcp_offers")

	if debugLevel >= debugLevelInfo {
		logging.ProtocolDebugf("DHCP", debugLevel, debugLevelInfo, "Sent Offer IP=%s to %s sn=%d",
			lease.IP, info.dhcp.ClientHWAddr, serialNum)
	}
}

// handleDHCPRequest handles a DHCP Request message.
func (h *DHCPHandler) handleDHCPRequest(
	info *dhcpPacketInfo,
	serverDevice *config.Device,
	serialNum int,
	debugLevel int,
) {
	logger := slog.Default()

	if debugLevel >= DebugLevelInfo {
		logger.Debug("DHCP: Processing Request", "mac", info.dhcp.ClientHWAddr, "sn", serialNum)
	}

	requestedIP := getRequestedIP(info.dhcp)

	lease, err := h.allocateLease(info.dhcp.ClientHWAddr, requestedIP, info.hostname)
	if err != nil {
		if debugLevel >= 1 {
			logging.ProtocolDebugf("DHCP", debugLevel, 1, "Failed to confirm lease: %v sn=%d", err, serialNum)
		}

		return
	}

	ackErr := h.SendDHCPAck(
		info.dhcp.Xid,
		info.dhcp.ClientHWAddr,
		lease.IP,
		serverDevice.IPAddresses[0],
		serverDevice.MACAddress,
	)
	if ackErr != nil {
		if debugLevel >= 1 {
			logging.ProtocolDebugf("DHCP", debugLevel, 1, "Failed to send Ack: %v sn=%d", ackErr, serialNum)
		}

		return
	}

	h.stack.IncrementStat("dhcp_acks")
	h.updateFDBTables(info.dhcp.ClientHWAddr)

	if debugLevel >= debugLevelInfo {
		logging.ProtocolDebugf("DHCP", debugLevel, debugLevelInfo, "Sent Ack IP=%s to %s sn=%d",
			lease.IP, info.dhcp.ClientHWAddr, serialNum)
	}
}

// HandlePacket processes a DHCP packet.
func (h *DHCPHandler) HandlePacket(pkt *Packet, ipLayer *layers.IPv4, _ *layers.UDP, devices []*config.Device) {
	logger := slog.Default()
	debugLevel := h.stack.GetDebugLevel()

	h.stack.IncrementStat("dhcp_requests")

	info := h.parseDHCPPacket(pkt)
	if info == nil {
		if debugLevel >= DebugLevelInfo {
			logger.Debug("DHCP packet missing DHCP layer", "sn", pkt.SerialNumber)
		}

		return
	}

	if debugLevel >= DebugLevelVerbose {
		logger.Debug("DHCP message received",
			"type", h.dhcpMessageTypeString(info.messageType),
			"srcIP", ipLayer.SrcIP,
			"mac", info.dhcp.ClientHWAddr,
			"xid", info.dhcp.Xid,
			"sn", pkt.SerialNumber)
	}

	serverDevice := findServerDevice(devices)
	if serverDevice == nil {
		if debugLevel >= DebugLevelInfo {
			logger.Debug("DHCP: No server device configured", "sn", pkt.SerialNumber)
		}

		return
	}

	h.dispatchDHCPMessage(info, serverDevice, pkt.SerialNumber, debugLevel)
}

// dispatchDHCPMessage routes the DHCP message to the appropriate handler.
func (h *DHCPHandler) dispatchDHCPMessage(
	info *dhcpPacketInfo,
	serverDevice *config.Device,
	serialNum int,
	debugLevel int,
) {
	logger := slog.Default()

	switch info.messageType {
	case DHCPDiscover:
		h.handleDHCPDiscover(info, serverDevice, serialNum, debugLevel)
	case DHCPRequest:
		h.handleDHCPRequest(info, serverDevice, serialNum, debugLevel)
	case DHCPRelease:
		if debugLevel >= DebugLevelInfo {
			logger.Debug("DHCP: Release", "mac", info.dhcp.ClientHWAddr, "sn", serialNum)
		}

		h.mu.Lock()
		delete(h.leases, info.dhcp.ClientHWAddr.String())
		h.mu.Unlock()
	case DHCPInform:
		if debugLevel >= DebugLevelVerbose {
			logger.Debug("DHCP: Inform", "mac", info.dhcp.ClientHWAddr, "sn", serialNum)
		}
	default:
		if debugLevel >= DebugLevelInfo {
			logger.Debug("DHCP: Unhandled message type", "type", info.messageType, "sn", serialNum)
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

// buildDHCPOptions constructs the DHCP options for a response.
// Caller must hold h.mu (at least RLock).
func (h *DHCPHandler) buildDHCPOptions(
	msgType uint8,
	serverIP net.IP,
	clientMAC net.HardwareAddr,
) []layers.DHCPOption {
	options := []layers.DHCPOption{
		{Type: layers.DHCPOptMessageType, Length: 1, Data: []byte{msgType}},
		{Type: layers.DHCPOptServerID, Length: dhcpIPv4Len, Data: []byte(serverIP.To4())},
		{Type: layers.DHCPOptLeaseTime, Length: dhcpIPv4Len, Data: h.encodeUint32(uint32(DefaultLeaseTime.Seconds()))},
		{Type: layers.DHCPOptSubnetMask, Length: dhcpIPv4Len, Data: []byte(h.subnetMask.To4())},
	}

	options = h.appendNetworkOptions(options)
	options = h.appendTimingOptions(options)
	options = h.appendBootOptions(options)
	options = h.appendMiscOptions(options, clientMAC)

	options = append(options, layers.DHCPOption{Type: layers.DHCPOptEnd})

	return options
}

// appendNetworkOptions adds gateway, DNS, and domain options.
// Caller must hold h.mu (at least RLock).
func (h *DHCPHandler) appendNetworkOptions(options []layers.DHCPOption) []layers.DHCPOption {
	if h.gateway != nil {
		options = append(options, layers.DHCPOption{
			Type: layers.DHCPOptRouter, Length: dhcpIPv4Len, Data: []byte(h.gateway.To4()),
		})
	}

	if len(h.dnsServers) > 0 {
		dnsData := make([]byte, 0, len(h.dnsServers)*dhcpIPv4Len)
		for _, dns := range h.dnsServers {
			dnsData = append(dnsData, []byte(dns.To4())...)
		}

		dnsLen := min(len(dnsData), dhcpMaxByte)
		options = append(options, layers.DHCPOption{
			Type: layers.DHCPOptDNS, Length: safeUint8(dnsLen), Data: dnsData[:dnsLen],
		})
	}

	if h.domainName != "" {
		domainLen := min(len(h.domainName), dhcpMaxByte)
		options = append(options, layers.DHCPOption{
			Type: layers.DHCPOptDomainName, Length: safeUint8(domainLen), Data: []byte(h.domainName[:domainLen]),
		})
	}

	return options
}

// appendTimingOptions adds T1, T2, and NTP server options.
// Caller must hold h.mu (at least RLock).
func (h *DHCPHandler) appendTimingOptions(options []layers.DHCPOption) []layers.DHCPOption {
	// Add renewal time (T1) - 50% of lease time
	options = append(options, layers.DHCPOption{
		Type: layers.DHCPOptT1, Length: dhcpIPv4Len,
		Data: h.encodeUint32(uint32(DefaultLeaseTime.Seconds() / t1RenewalDivisor)),
	})

	// Add rebinding time (T2) - 87.5% of lease time
	options = append(options, layers.DHCPOption{
		Type: layers.DHCPOptT2, Length: dhcpIPv4Len,
		Data: h.encodeUint32(uint32(DefaultLeaseTime.Seconds() * t2RebindNumerator / t2RebindDivisor)),
	})

	// Add NTP servers if configured (Option 42)
	if len(h.ntpServers) > 0 {
		ntpData := make([]byte, 0, len(h.ntpServers)*dhcpIPv4Len)
		for _, ntp := range h.ntpServers {
			ntpData = append(ntpData, []byte(ntp.To4())...)
		}

		ntpLen := min(len(ntpData), dhcpMaxByte)
		options = append(options, layers.DHCPOption{
			Type: DHCPOptNTP, Length: safeUint8(ntpLen), Data: ntpData[:ntpLen],
		})
	}

	return options
}

// appendBootOptions adds TFTP, bootfile, and domain search options.
// Caller must hold h.mu (at least RLock).
func (h *DHCPHandler) appendBootOptions(options []layers.DHCPOption) []layers.DHCPOption {
	if h.tftpServerName != "" {
		tftpLen := min(len(h.tftpServerName), dhcpMaxByte)
		options = append(options, layers.DHCPOption{
			Type: DHCPOptTFTPServer, Length: safeUint8(tftpLen), Data: []byte(h.tftpServerName[:tftpLen]),
		})
	}

	if h.bootfileName != "" {
		bootLen := min(len(h.bootfileName), dhcpMaxByte)
		options = append(options, layers.DHCPOption{
			Type: DHCPOptBootfileName, Length: safeUint8(bootLen), Data: []byte(h.bootfileName[:bootLen]),
		})
	}

	if len(h.domainSearch) > 0 {
		searchData, err := h.encodeDomainSearchList(h.domainSearch)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: DHCP domain search encoding failed: %v\n", err)
		} else if len(searchData) > 0 {
			searchLen := min(len(searchData), dhcpMaxByte)
			options = append(options, layers.DHCPOption{
				Type: DHCPOptDomainSearch, Length: safeUint8(searchLen), Data: searchData[:searchLen],
			})
		}
	}

	return options
}

// appendMiscOptions adds vendor info and hostname options.
// Caller must hold h.mu (at least RLock).
func (h *DHCPHandler) appendMiscOptions(options []layers.DHCPOption, clientMAC net.HardwareAddr) []layers.DHCPOption {
	if len(h.vendorSpecificInfo) > 0 {
		vendorLen := min(len(h.vendorSpecificInfo), dhcpMaxByte)
		options = append(options, layers.DHCPOption{
			Type: layers.DHCPOptVendorOption, Length: safeUint8(vendorLen), Data: h.vendorSpecificInfo[:vendorLen],
		})
	}

	if lease, ok := h.leases[clientMAC.String()]; ok && lease.Hostname != "" {
		hostnameLen := min(len(lease.Hostname), dhcpMaxByte)
		options = append(options, layers.DHCPOption{
			Type: layers.DHCPOptHostname, Length: safeUint8(hostnameLen), Data: []byte(lease.Hostname[:hostnameLen]),
		})
	}

	return options
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

	dhcp := &layers.DHCPv4{
		Operation:    layers.DHCPOpReply,
		HardwareType: layers.LinkTypeEthernet,
		HardwareLen:  dhcpHWAddrLen,
		HardwareOpts: 0,
		Xid:          xid,
		Secs:         0,
		Flags:        dhcpBroadcastFlag,
		ClientIP:     net.IPv4zero,
		YourClientIP: assignedIP,
		NextServerIP: net.IPv4zero,
		RelayAgentIP: net.IPv4zero,
		ClientHWAddr: clientMAC,
	}
	dhcp.Options = h.buildDHCPOptions(msgType, serverIP, clientMAC)

	return h.serializeAndSendDHCP(dhcp, serverIP, serverMAC)
}

// serializeAndSendDHCP serializes and sends a DHCP packet.
func (h *DHCPHandler) serializeAndSendDHCP(
	dhcp *layers.DHCPv4,
	serverIP net.IP,
	serverMAC net.HardwareAddr,
) error {
	udp := &layers.UDP{SrcPort: dhcpServerPort, DstPort: dhcpClientPort}
	ip := &layers.IPv4{
		Version: ipv4Version, TTL: ipv4DefaultTTL, Protocol: layers.IPProtocolUDP,
		SrcIP: serverIP, DstIP: net.IPv4bcast,
	}
	eth := &layers.Ethernet{
		SrcMAC: serverMAC, DstMAC: net.HardwareAddr{0xff, 0xff, 0xff, 0xff, 0xff, 0xff},
		EthernetType: layers.EthernetTypeIPv4,
	}

	buf := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{ComputeChecksums: true, FixLengths: true}

	_ = udp.SetNetworkLayerForChecksum(ip)

	if err := gopacket.SerializeLayers(buf, opts, eth, ip, udp, dhcp); err != nil {
		return fmt.Errorf("failed to serialize DHCP response: %w", err)
	}

	return h.stack.SendRawPacket(buf.Bytes())
}

// encodeUint32 encodes a uint32 as big-endian bytes.
func (h *DHCPHandler) encodeUint32(val uint32) []byte {
	b := make([]byte, dhcpIPv4Len)
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
		labels := make([]byte, 0, len(domain)+domainLabelBufferPad)

		for _, label := range splitDomain(domain) {
			if len(label) == 0 || len(label) > maxDomainLabelLen {
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
