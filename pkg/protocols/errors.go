package protocols

import "errors"

// Sentinel errors for ARP.
var ErrDeviceMissingMACOrIP = errors.New("device missing MAC or IP address")

// Sentinel errors for DHCP.
var (
	ErrDHCPPoolInvalid               = errors.New("invalid DHCP pool: end IP < start IP")
	ErrDHCPPoolSizeExceeded          = errors.New("DHCP pool size exceeds maximum")
	ErrNoAvailableIPAddresses        = errors.New("no available IP addresses")
	ErrTooManyDomainSearchEntries    = errors.New("too many domain search entries")
	ErrDomainTooLong                 = errors.New("domain too long")
	ErrDomainSearchListExceedsMaxLen = errors.New("domain search list exceeds DHCP option max size")
)

// Sentinel errors for DHCPv6.
var (
	ErrMessageTooShort      = errors.New("message too short")
	ErrOptionDataExceedsLen = errors.New("option data exceeds message length")
	ErrNoAvailableAddresses = errors.New("no available addresses")
)

// Sentinel errors for HTTP.
var ErrInvalidRequestLine = errors.New("invalid request line")

// Sentinel errors for ICMP.
var ErrOriginalIPLayerMissing = errors.New("original IP layer missing")

// Sentinel errors for packet decoding.
var ErrDecodingPacket = errors.New("error decoding packet")

// Sentinel errors for Stack.
var (
	ErrStackAlreadyRunning = errors.New("stack already running")
	ErrNilConfig           = errors.New("reload config: nil config")
)

// Sentinel errors for STP.
var ErrDeviceNoMACAddress = errors.New("device has no MAC address")
