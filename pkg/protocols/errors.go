package protocols

import "errors"

// ErrDeviceMissingMACOrIP is a sentinel error for ARP indicating device missing MAC or IP address.
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

// ErrInvalidRequestLine is a sentinel error for HTTP indicating an invalid request line.
var ErrInvalidRequestLine = errors.New("invalid request line")

// ErrOriginalIPLayerMissing is a sentinel error for ICMP indicating the original IP layer is missing.
var ErrOriginalIPLayerMissing = errors.New("original IP layer missing")

// ErrDecodingPacket is a sentinel error for packet decoding failures.
var ErrDecodingPacket = errors.New("error decoding packet")

// Sentinel errors for Stack.
var (
	ErrStackAlreadyRunning = errors.New("stack already running")
	ErrNilConfig           = errors.New("reload config: nil config")
)

// ErrDeviceNoMACAddress is a sentinel error for STP indicating device has no MAC address.
var ErrDeviceNoMACAddress = errors.New("device has no MAC address")
