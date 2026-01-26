package protocols

// Queue buffer size constants.
const (
	// DefaultQueueBufferSize is the default buffer size for send/receive queues.
	// Increase this for high-traffic scenarios to prevent packet drops.
	// Decrease for memory-constrained environments.
	DefaultQueueBufferSize = 1000

	// Recommended sizes for different scenarios:
	// - Low traffic (< 100 pps): 500
	// - Normal traffic (100-1000 pps): 1000 (default)
	// - High traffic (1000-10000 pps): 5000
	// - Very high traffic (> 10000 pps): 10000.
)

// Stack internal constants.
const (
	stackReceiveBufferSize    = 65536 // Receive buffer size
	stackSelectTimeoutMs      = 100   // Select timeout in milliseconds
	stackBabbleIntervalSec    = 15    // Babble traffic interval in seconds
	stackBabbleDelayMs        = 10    // Delay between babble packets
	stackNeighborCleanupSec   = 30    // Neighbor cleanup interval in seconds
	stackBabbleTargetIPOctet3 = 10    // Babble target IP octet (10.1.1.1)
)

// Multicast MAC address bytes for protocol detection.
const (
	macAddrLen       = 6    // MAC address length in bytes
	ipv4AddrLen      = 4    // IPv4 address length in bytes
	ethMACCount      = 2    // Number of MAC addresses in Ethernet header (src + dst)
	macMulticastIEEE = 0x01 // IEEE multicast first byte (STP, LLDP, CDP, FDP)
	macUnicastZero   = 0x00 // Unicast/EDP first byte
	macSecondByteSTP = 0x80 // Second byte for STP/LLDP multicast
	macSecondByteEDP = 0xE0 // Second byte for EDP/FDP multicast
	macThirdByteSTP  = 0xC2 // Third byte for STP/LLDP (01:80:C2:...)
	macThirdByteCDP  = 0x0C // Third byte for CDP (01:00:0C:...)
	macThirdByteEDP  = 0x2B // Third byte for EDP (00:E0:2B:...)
	macThirdByteFDP  = 0x52 // Third byte for FDP (01:E0:52:...)
	macByteLLDP      = 0x0E // Last byte for LLDP multicast
	macByteCC        = 0xCC // CC byte used in CDP/FDP patterns
)

// Debug level thresholds for consistent logging.
const (
	DebugLevelBasic   = 1 // Basic debug output
	DebugLevelInfo    = 2 // Info-level debug output
	DebugLevelVerbose = 3 // Verbose debug output
	DebugLevelTrace   = 4 // Trace-level debug output
)
