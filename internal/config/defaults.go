package config

// LLDP Chassis ID Type constants.
const (
	ChassisIDTypeMAC            = "mac"
	ChassisIDTypeLocal          = "local"
	ChassisIDTypeNetworkAddress = "network_address"
)

// Default configuration values.
const (
	// DefaultLLDPAdvertiseInterval is the default LLDP advertisement interval in seconds.
	DefaultLLDPAdvertiseInterval = 30
	// DefaultLLDPTTL is the default LLDP time-to-live in seconds.
	DefaultLLDPTTL = 120
	// DefaultCDPAdvertiseInterval is the default CDP advertisement interval in seconds.
	DefaultCDPAdvertiseInterval = 60
	// DefaultCDPHoldtime is the default CDP holdtime in seconds.
	DefaultCDPHoldtime = 180
	// DefaultCDPVersion is the default CDP protocol version.
	DefaultCDPVersion = 2
	// DefaultEDPAdvertiseInterval is the default EDP advertisement interval in seconds.
	DefaultEDPAdvertiseInterval = 30
	// DefaultFDPAdvertiseInterval is the default FDP advertisement interval in seconds.
	DefaultFDPAdvertiseInterval = 60
	// DefaultFDPHoldtime is the default FDP holdtime in seconds.
	DefaultFDPHoldtime = 180

	// DefaultSTPBridgePriority is the default STP bridge priority.
	DefaultSTPBridgePriority = 32768
	// DefaultSTPHelloTime is the default STP hello time in seconds.
	DefaultSTPHelloTime = 2
	// DefaultSTPMaxAge is the default STP max age in seconds.
	DefaultSTPMaxAge = 20
	// DefaultSTPForwardDelay is the default STP forward delay in seconds.
	DefaultSTPForwardDelay = 15

	// DefaultNetBIOSTTL is the default NetBIOS TTL in seconds.
	DefaultNetBIOSTTL = 300

	// DefaultICMPTTL is the default ICMP time-to-live.
	DefaultICMPTTL = 64
	// DefaultICMPv6HopLimit is the default ICMPv6 hop limit.
	DefaultICMPv6HopLimit = 64

	// DefaultDHCPv6PreferredLifetime is the default DHCPv6 preferred lifetime in seconds.
	DefaultDHCPv6PreferredLifetime = 604800
	// DefaultDHCPv6ValidLifetime is the default DHCPv6 valid lifetime in seconds.
	DefaultDHCPv6ValidLifetime = 2592000

	// DefaultARPAnnouncementInterval is the default ARP announcement interval in seconds.
	DefaultARPAnnouncementInterval = 60
	// DefaultPeriodicPingInterval is the default periodic ping interval in seconds.
	DefaultPeriodicPingInterval = 120
	// DefaultPeriodicPingPayloadSize is the default periodic ping payload size in bytes.
	DefaultPeriodicPingPayloadSize = 32
	// DefaultRandomTrafficInterval is the default random traffic interval in seconds.
	DefaultRandomTrafficInterval = 180
	// DefaultRandomTrafficPacketCount is the default packets per interval.
	DefaultRandomTrafficPacketCount = 5

	// DefaultSNMPCommunity is the default SNMP community string.
	DefaultSNMPCommunity = "public"

	// DefaultHighCPUThreshold is the default high CPU usage threshold in percent.
	DefaultHighCPUThreshold = 80
	// DefaultHighMemoryThreshold is the default high memory usage threshold in percent.
	DefaultHighMemoryThreshold = 90
	// DefaultInterfaceErrorThreshold is the default interface error threshold count.
	DefaultInterfaceErrorThreshold = 100
	// DefaultTrapCheckInterval is the default trap check interval in seconds.
	DefaultTrapCheckInterval = 300
	// DefaultInterfaceErrorInterval is the default interface error check interval in seconds.
	DefaultInterfaceErrorInterval = 60

	// DefaultDNSTTL is the default DNS TTL in seconds.
	DefaultDNSTTL = 3600
)

// Parser constants.
const (
	minDeviceDeclParts   = 2          // minimum parts for device declaration (device name)
	keyValueParts        = 2          // key-value pair after SplitN
	netbiosMaxNameLen    = 15         // NetBIOS name max length
	maxDNSTTL            = 2147483647 // max int32 (~68 years)
	minSimpleConfigParts = 4          // minimum fields: name type ip mac
	simpleConfigWithWalk = 5          // simple config with walkfile
	macAddressBytes      = 6          // MAC address length in bytes
	macPatternMultiplier = 17         // pattern multiplier for test MAC generation
	gbpsToMbps           = 1000       // conversion factor Gbps to Mbps
)
