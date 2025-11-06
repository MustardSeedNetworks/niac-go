package logging

import "sync"

// DebugConfig stores per-protocol debug levels
type DebugConfig struct {
	mu sync.RWMutex

	// Global debug level (fallback)
	Global int

	// Per-protocol debug levels
	protocols map[string]int
}

// NewDebugConfig creates a new debug configuration with a global default
func NewDebugConfig(globalLevel int) *DebugConfig {
	return &DebugConfig{
		Global:    globalLevel,
		protocols: make(map[string]int),
	}
}

// SetProtocolLevel sets the debug level for a specific protocol
func (d *DebugConfig) SetProtocolLevel(protocol string, level int) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.protocols[protocol] = level
}

// GetProtocolLevel returns the debug level for a specific protocol
// If not set, returns the global level
func (d *DebugConfig) GetProtocolLevel(protocol string) int {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if level, ok := d.protocols[protocol]; ok {
		return level
	}
	return d.Global
}

// SetGlobal sets the global debug level
func (d *DebugConfig) SetGlobal(level int) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.Global = level
}

// GetGlobal returns the global debug level
func (d *DebugConfig) GetGlobal() int {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.Global
}

// GetAllLevels returns a copy of all protocol-specific levels
func (d *DebugConfig) GetAllLevels() map[string]int {
	d.mu.RLock()
	defer d.mu.RUnlock()

	levels := make(map[string]int, len(d.protocols))
	for k, v := range d.protocols {
		levels[k] = v
	}
	return levels
}

// HasProtocolLevel returns true if a protocol has a specific level set
func (d *DebugConfig) HasProtocolLevel(protocol string) bool {
	d.mu.RLock()
	defer d.mu.RUnlock()
	_, ok := d.protocols[protocol]
	return ok
}

// Protocol names (constants for consistency)
const (
	ProtocolARP     = "ARP"
	ProtocolIP      = "IP"
	ProtocolICMP    = "ICMP"
	ProtocolIPv6    = "IPv6"
	ProtocolICMPv6  = "ICMPv6"
	ProtocolUDP     = "UDP"
	ProtocolTCP     = "TCP"
	ProtocolDNS     = "DNS"
	ProtocolDHCP    = "DHCP"
	ProtocolDHCPv6  = "DHCPv6"
	ProtocolHTTP    = "HTTP"
	ProtocolFTP     = "FTP"
	ProtocolNetBIOS = "NetBIOS"
	ProtocolSTP     = "STP"
	ProtocolLLDP    = "LLDP"
	ProtocolCDP     = "CDP"
	ProtocolEDP     = "EDP"
	ProtocolFDP     = "FDP"
	ProtocolSNMP    = "SNMP"
)
