package snmp

// SNMP numeric constants used across the package.
const (
	// OIDPartsMinPDU is the minimum OID parts for PDU validation.
	OIDPartsMinPDU = 2
	// OIDPartsMinV3 is the minimum parts for SNMPv3 validation.
	OIDPartsMinV3 = 3
	// SNMPRetryCount is the number of retries for SNMP operations.
	SNMPRetryCount = 10
	// MillisecsPerCentisec converts milliseconds to centiseconds (timeticks).
	MillisecsPerCentisec = 10
	// TrafficDivisor is the divisor for simulated traffic rate.
	TrafficDivisor = 10
	// MACAddressOctets is the number of octets in a MAC address.
	MACAddressOctets = 6
	// BytesPerInt is the number of bytes in an integer.
	BytesPerInt = 4
	// BitShiftByte is the bit shift for a byte.
	BitShiftByte = 8
	// SecondsPerMinute is the number of seconds in a minute.
	SecondsPerMinute = 60
	// ThresholdPercentage is the threshold percentage for calculations.
	ThresholdPercentage = 100
	// NanosPerSecond is the number of nanoseconds per second.
	NanosPerSecond = 1000000000
	// MaxUint32Value is the maximum uint32 value.
	MaxUint32Value = 0xFFFFFFFF
	// Uint32Overflow is the uint32 overflow value.
	Uint32Overflow = 4294967296
	// BERMaxShortLen is the maximum value for short-form BER length.
	BERMaxShortLen = 127
	// BERLongFormIndicator is the indicator for long-form BER length.
	BERLongFormIndicator = 128
	// MaxOIDResultSize is the max results for SNMP bulk operations.
	MaxOIDResultSize = 50

	// DebugLevelVerbose is the verbose debug level.
	DebugLevelVerbose = 2
	// DebugLevelDebug is the debug level.
	DebugLevelDebug = 3

	// MinRedactLen is the minimum length for partial redaction.
	MinRedactLen = 2

	// BERHighBitMask is the high bit mask for BER encoding.
	BERHighBitMask = 0x80
	// BERLowMask is the low mask for BER encoding.
	BERLowMask = 0x7F
	// BERFullByte is the full byte mask for BER encoding.
	BERFullByte = 0xFF

	// Speed100Gbps is 100 Gigabit in bits per second.
	Speed100Gbps = 100000000000
	// Speed40Gbps is 40 Gigabit in bits per second.
	Speed40Gbps = 40000000000
	// Speed25Gbps is 25 Gigabit in bits per second.
	Speed25Gbps = 25000000000
	// Speed10Gbps is 10 Gigabit in bits per second.
	Speed10Gbps = 10000000000
	// Speed5Gbps is 5 Gigabit in bits per second.
	Speed5Gbps = 5000000000
	// Speed2p5Gbps is 2.5 Gigabit in bits per second.
	Speed2p5Gbps = 2500000000
	// Speed100Mbps is 100 Megabit (Fast Ethernet) in bits per second.
	Speed100Mbps = 100000000
	// MicrosPerSec is the number of microseconds per second.
	MicrosPerSec = 1000000

	// CapabilityRouterBridge is the Router + Bridge capability bitmask.
	CapabilityRouterBridge = 0x14
	// CapabilityBridge is the Bridge (switch) capability bitmask.
	CapabilityBridge = 0x04
	// CapabilityWLANAP is the WLAN Access Point capability bitmask.
	CapabilityWLANAP = 0x08
	// CapabilityStationOnly is the Station Only capability bitmask.
	CapabilityStationOnly = 0x80

	// BERLenByteMask is the mask for length bytes count in BER.
	BERLenByteMask = 7

	// IPAddressOctets is the number of octets in an IPv4 address.
	IPAddressOctets = 4

	// MaxBERLengthBytes is the maximum bytes for BER length encoding.
	MaxBERLengthBytes = 4

	// OIDSubIDThreshold is the threshold for variable-length OID sub-ID encoding.
	OIDSubIDThreshold = 128

	// OIDShiftBits is the bit shift for OID variable-length encoding.
	OIDShiftBits = 7

	// EthernetCsmacdType is the interface type for Ethernet CSMA/CD.
	EthernetCsmacdType = 6

	// DefaultMTU is the default MTU for Ethernet interfaces.
	DefaultMTU = 1500

	// MinValidOIDParts is the minimum parts for a valid OID.
	MinValidOIDParts = 2

	// MaxIPPort is the maximum valid IP port number.
	MaxIPPort = 65535

	// ChassisIDSubtypeMAC is the LLDP chassis ID subtype for MAC address.
	ChassisIDSubtypeMAC = 4

	// PortIDSubtypeInterfaceName is the LLDP port ID subtype for interface name.
	PortIDSubtypeInterfaceName = 5

	// LLDPCapabilityOther is the LLDP other capability bitmask.
	LLDPCapabilityOther = 0x01

	// STPPriorityDefault is the default STP bridge priority.
	STPPriorityDefault = 32768

	// STPPortPriorityDefault is the default STP port priority.
	STPPortPriorityDefault = 128

	// STPPortStateForwarding is the STP port state for forwarding.
	STPPortStateForwarding = 5

	// STPPortPathCostDefault is the default STP port path cost.
	STPPortPathCostDefault = 4

	// DuplexFull is the value for full duplex.
	DuplexFull = 3

	// DefaultSNMPTrapPort is the default SNMP trap receiver port.
	DefaultSNMPTrapPort = 162

	// TrapTimeoutSeconds is the timeout for SNMP trap operations.
	TrapTimeoutSeconds = 2

	// SimulatedThresholdMultiplier is the multiplier for simulated threshold checks.
	SimulatedThresholdMultiplier = 2

	// DebugLevelMinimum is the minimum debug level for logging.
	DebugLevelMinimum = 2

	// DebugLevelTraps is the debug level for trap logging.
	DebugLevelTraps = 3

	// CentisecsPerSec is the number of centiseconds per second.
	CentisecsPerSec = 10

	// FrameRateInMultiplier is the multiplier for incoming frame rate simulation.
	FrameRateInMultiplier = 100

	// FrameRateOutMultiplier is the multiplier for outgoing frame rate simulation.
	FrameRateOutMultiplier = 80

	// FDBStatusSelf is the FDB status for self entry.
	FDBStatusSelf = 4

	// FDBAgingTimeDefault is the default FDB aging time in seconds.
	FDBAgingTimeDefault = 300

	// CDPDefaultInterval is the default CDP advertisement interval in seconds.
	CDPDefaultInterval = 60

	// CDPDefaultHoldtime is the default CDP holdtime in seconds.
	CDPDefaultHoldtime = 180

	// STPMaxAgeDefault is the default STP max age in seconds.
	STPMaxAgeDefault = 20

	// STPHelloTimeDefault is the default STP hello time in seconds.
	STPHelloTimeDefault = 2

	// STPForwardDelayDefault is the default STP forward delay in seconds.
	STPForwardDelayDefault = 15

	// HundredthsPerSecond is the conversion factor for STP timers (centiseconds).
	HundredthsPerSecond = 100

	// BridgeIDLength is the length of an 802.1D bridge ID in bytes.
	BridgeIDLength = 8

	// BaseDecimal is the base for decimal number parsing.
	BaseDecimal = 10

	// TruthValueFalse is SNMP TruthValue for false (2).
	TruthValueFalse = 2

	// TruthValueTrue is SNMP TruthValue for true (1).
	TruthValueTrue = 1

	// IPRouteTypeIndirect is the route type for indirect routes (4).
	IPRouteTypeIndirect = 4

	// IPRouteProtoLocal is the routing protocol for local routes (2).
	IPRouteProtoLocal = 2

	// BridgeTypeTransparent is the bridge type for transparent-only bridges (2).
	BridgeTypeTransparent = 2

	// STPProtocolIEEE is the STP protocol specification for IEEE 802.1d (3).
	STPProtocolIEEE = 3

	// ARPTypeStatic is the ARP entry type for static entries (4).
	ARPTypeStatic = 4

	// ARPTypeDynamic is the ARP entry type for dynamic entries (3).
	ARPTypeDynamic = 3

	// IfStatusUp is the interface status value for up (1).
	IfStatusUp = 1

	// IfStatusDown is the interface status value for down (2).
	IfStatusDown = 2
)
