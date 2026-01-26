package protocols

import (
	"net"
	"time"
)

// DHCPv6 message types (RFC 8415).
const (
	DHCPv6Solicit     = 1
	DHCPv6Advertise   = 2
	DHCPv6Request     = 3
	DHCPv6Confirm     = 4
	DHCPv6Renew       = 5
	DHCPv6Rebind      = 6
	DHCPv6Reply       = 7
	DHCPv6Release     = 8
	DHCPv6Decline     = 9
	DHCPv6Reconfigure = 10
	DHCPv6InfoRequest = 11
	DHCPv6RelayForw   = 12
	DHCPv6RelayRepl   = 13
)

// DHCPv6 option codes (RFC 8415).
const (
	DHCPv6OptClientID               = 1
	DHCPv6OptServerID               = 2
	DHCPv6OptIANA                   = 3 // Identity Association for Non-temporary Addresses
	DHCPv6OptIATA                   = 4 // Identity Association for Temporary Addresses
	DHCPv6OptIAAddr                 = 5 // IA Address
	DHCPv6OptORO                    = 6 // Option Request Option
	DHCPv6OptPreference             = 7
	DHCPv6OptElapsedTime            = 8
	DHCPv6OptRelayMsg               = 9
	DHCPv6OptAuth                   = 11
	DHCPv6OptUnicast                = 12
	DHCPv6OptStatusCode             = 13
	DHCPv6OptRapidCommit            = 14
	DHCPv6OptUserClass              = 15
	DHCPv6OptVendorClass            = 16
	DHCPv6OptVendorOpts             = 17
	DHCPv6OptInterfaceID            = 18
	DHCPv6OptReconfMsg              = 19
	DHCPv6OptReconfAccept           = 20
	DHCPv6OptSIPServers             = 21 // SIP Servers (Domain Name List)
	DHCPv6OptSIPServerAddrs         = 22 // SIP Servers (IPv6 Address List)
	DHCPv6OptDNSServers             = 23
	DHCPv6OptDomainList             = 24
	DHCPv6OptIAPD                   = 25 // Identity Association for Prefix Delegation
	DHCPv6OptIAPrefix               = 26
	DHCPv6OptSNTPServers            = 31 // SNTP Servers
	DHCPv6OptInformationRefreshTime = 32
	DHCPv6OptFQDN                   = 39 // Client FQDN
	DHCPv6OptNTPServer              = 56 // NTP Server
)

// DHCPv6 status codes.
const (
	DHCPv6StatusSuccess      = 0
	DHCPv6StatusUnspecFail   = 1
	DHCPv6StatusNoAddrsAvail = 2
	DHCPv6StatusNoBinding    = 3
	DHCPv6StatusNotOnLink    = 4
	DHCPv6StatusUseMulticast = 5
)

// DHCPv6 DUID types (RFC 8415).
const (
	DUIDTypeLLT = 1 // Link-layer address plus time
	DUIDTypeEN  = 2 // Vendor-assigned unique ID based on Enterprise Number
	DUIDTypeLL  = 3 // Link-layer address
)

// DHCPv6 lease duration (7 days preferred, 30 days valid).
const (
	DefaultPreferredLifetime = 7 * 24 * time.Hour
	DefaultValidLifetime     = 30 * 24 * time.Hour
)

// DHCPv6 ports.
const (
	DHCPv6ServerPort = 547
	DHCPv6ClientPort = 546
)

// DHCPv6 encoding constants.
const (
	duidLLSize          = 10    // DUID-LL format: Type(2) + HW Type(2) + MAC(6)
	macAddrSize         = 6     // MAC address size in bytes
	ipv6AddrSize        = 16    // IPv6 address size in bytes
	maxUint16Val        = 65535 // maximum uint16 value for option lengths
	udpHeaderSize       = 8     // UDP header size in bytes
	maxUDPPayload       = 65527 // max UDP payload (65535 - 8)
	ianaHeaderBase      = 12    // IANA header base size (IAID + T1 + T2)
	iaAddrDataSize      = 24    // IA Address data size (IP + preferred + valid)
	optionHeaderSize    = 4     // DHCPv6 option header size (Code 2 + Length 2)
	t1DivisorV6         = 2     // T1 = preferred lifetime / 2 (50%)
	t2MultiplicandV6    = 4     // T2 = preferred lifetime * 4 / 5 (80%)
	t2DivisorV6         = 5     // T2 divisor for 80% calculation
	dhcpv6MinMsgSize    = 4     // Minimum DHCPv6 message size (type + transaction ID)
	macUnicastMask      = 0xfe  // Mask to clear multicast bit in MAC address
	macLocalBitSet      = 0x02  // Bit to set for locally administered MAC address
	ipv6VersionVal      = 6     // IPv6 version field
	ipv6DefaultHopLimit = 64    // IPv6 default hop limit
)

// GetAllDHCPRelayAgentsAndServers returns the DHCPv6 all relay agents and servers multicast address.
func GetAllDHCPRelayAgentsAndServers() net.IP {
	return net.ParseIP("ff02::1:2")
}

// GetAllDHCPServers returns the DHCPv6 all servers multicast address.
func GetAllDHCPServers() net.IP {
	return net.ParseIP("ff05::1:3")
}

// DHCPv6Message represents a DHCPv6 message.
type DHCPv6Message struct {
	MessageType   uint8
	TransactionID [3]byte
	Options       []DHCPv6Option
}

// DHCPv6Option represents a DHCPv6 option.
type DHCPv6Option struct {
	Code   uint16
	Length uint16
	Data   []byte
}

// DHCPv6Lease represents an IPv6 address lease.
type DHCPv6Lease struct {
	Address           net.IP
	Prefix            *net.IPNet // For prefix delegation
	DUID              []byte
	IAID              uint32
	PreferredLifetime time.Time
	ValidLifetime     time.Time
	LastRenewal       time.Time
}
