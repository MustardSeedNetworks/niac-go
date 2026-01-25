package protocols

import (
	"fmt"
	"net"
	"os"
	"sync"
	"time"

	"github.com/google/gopacket/layers"

	"github.com/krisarmstrong/niac-go/internal/config"
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
