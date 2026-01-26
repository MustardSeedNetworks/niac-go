package protocols

import (
	"fmt"
	"net"
	"time"

	"github.com/google/gopacket/layers"
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

// MaxPoolSize is the maximum number of IPs allowed in a DHCP pool.
const MaxPoolSize = 65536 // 2^16 IPs (reasonable for simulation)

// DHCP option constraints (RFC 2132, RFC 3397).
const (
	MaxDHCPOptionLen     = 255 // Maximum DHCP option length
	MaxDomainSearchCount = 10  // Reasonable limit for simulation
	MaxDomainLen         = 253 // RFC 1035: maximum domain name length
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

// dhcpPacketInfo holds parsed DHCP packet information.
type dhcpPacketInfo struct {
	dhcp        *layers.DHCPv4
	messageType uint8
	hostname    string
}

// dhcpMessageTypeString returns string representation of DHCP message type.
func dhcpMessageTypeString(msgType uint8) string {
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
