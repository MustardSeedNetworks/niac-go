// Package snmp implements SNMP agent functionality including MIB management and trap sending.
package snmp

import (
	"fmt"
	"log/slog"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/gosnmp/gosnmp"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
)

// Default OID constants.
const (
	// DefaultCiscoSysObjectID is the default sysObjectID for Cisco devices.
	DefaultCiscoSysObjectID = "1.3.6.1.4.1.9.1.1"

	// unknownPlaceholder is used as a fallback value when actual data is unavailable.
	unknownPlaceholder = "Unknown"
)

// Package-local aliases for exported constants.
const (
	millisecsPerCentisec = MillisecsPerCentisec
	maxUint32            = MaxUint32Value
	debugLevelVerbose    = DebugLevelVerbose
	debugLevelDebug      = DebugLevelDebug
	minRedactLen         = MinRedactLen
)

// Agent represents an SNMP agent instance for a device.
type Agent struct {
	device     *config.Device
	mib        *MIB
	community  string
	startTime  time.Time
	debugLevel int
	logger     *slog.Logger
	mu         sync.RWMutex
}

// NewAgent creates a new SNMP agent for a device using the device's community.
func NewAgent(device *config.Device, debugLevel int) *Agent {
	community := config.DefaultSNMPCommunity
	if device != nil && device.SNMPConfig.Community != "" {
		community = device.SNMPConfig.Community
	}

	return NewAgentWithCommunity(device, community, debugLevel)
}

// NewAgentWithCommunity creates a new SNMP agent for a device with a specific community.
func NewAgentWithCommunity(device *config.Device, community string, debugLevel int) *Agent {
	agent := &Agent{
		device:     device,
		mib:        NewMIB(),
		community:  community,
		startTime:  time.Now(),
		debugLevel: debugLevel,
		logger:     slog.Default(),
	}
	if agent.community == "" {
		agent.community = config.DefaultSNMPCommunity
	}

	// Initialize standard MIB-II system objects
	agent.initializeSystemMIB()

	// Initialize neighbor discovery MIBs (IF-MIB, LLDP-MIB, CDP-MIB)
	agent.initializeNeighborMIBs()

	return agent
}

// initializeSystemMIB initializes standard MIB-II system group OIDs.
func (a *Agent) initializeSystemMIB() {
	// sysDescr (1.3.6.1.2.1.1.1.0)
	sysDescr := a.device.Properties["sysDescr"]
	if sysDescr == "" {
		sysDescr = fmt.Sprintf("%s %s", a.device.Type, a.device.Name)
	}

	a.mib.Set("1.3.6.1.2.1.1.1.0", &OIDValue{
		Type:  gosnmp.OctetString,
		Value: sysDescr,
	})

	// sysObjectID (1.3.6.1.2.1.1.2.0)
	sysObjectID := a.device.Properties["sysObjectID"]
	if sysObjectID == "" {
		sysObjectID = DefaultCiscoSysObjectID // Default to generic Cisco
	}

	a.mib.Set("1.3.6.1.2.1.1.2.0", &OIDValue{
		Type:  gosnmp.ObjectIdentifier,
		Value: sysObjectID,
	})

	// sysUpTime (1.3.6.1.2.1.1.3.0) - TimeTicks (hundredths of second)
	a.mib.SetDynamic("1.3.6.1.2.1.1.3.0", func() *OIDValue {
		uptime := time.Since(a.startTime)

		ms := min(
			// Convert to hundredths of second
			uptime.Milliseconds()/millisecsPerCentisec, maxUint32)

		timeticks := safeUint32(ms)

		return &OIDValue{
			Type:  gosnmp.TimeTicks,
			Value: timeticks,
		}
	})

	// sysContact (1.3.6.1.2.1.1.4.0)
	sysContact := a.device.Properties["sysContact"]
	if sysContact == "" {
		sysContact = "admin@example.com"
	}

	a.mib.Set("1.3.6.1.2.1.1.4.0", &OIDValue{
		Type:  gosnmp.OctetString,
		Value: sysContact,
	})

	// sysName (1.3.6.1.2.1.1.5.0)
	sysName := a.device.Properties["sysName"]
	if sysName == "" {
		sysName = a.device.Name
	}

	a.mib.Set("1.3.6.1.2.1.1.5.0", &OIDValue{
		Type:  gosnmp.OctetString,
		Value: sysName,
	})

	// sysLocation (1.3.6.1.2.1.1.6.0)
	sysLocation := a.device.Properties["sysLocation"]
	if sysLocation == "" {
		sysLocation = unknownPlaceholder
	}

	a.mib.Set("1.3.6.1.2.1.1.6.0", &OIDValue{
		Type:  gosnmp.OctetString,
		Value: sysLocation,
	})

	// sysServices (1.3.6.1.2.1.1.7.0)
	// Bit 0 (LSB): physical (e.g., repeaters)
	// Bit 2: internet (e.g., IP gateways)
	// Bit 3: end-to-end  (e.g., IP hosts)
	// Bit 6: application (e.g., mail relays)
	sysServices := 72 // Typical for L3 device (bits 3 and 6)
	a.mib.Set("1.3.6.1.2.1.1.7.0", &OIDValue{
		Type:  gosnmp.Integer,
		Value: sysServices,
	})

	if a.debugLevel >= debugLevelVerbose {
		a.logger.Debug("Initialized system MIB", "device", a.device.Name)
	}
}

// LoadWalkFile loads SNMP walk file data into the MIB.
func (a *Agent) LoadWalkFile(filename string) error {
	if filename == "" {
		return ErrNoWalkFileSpecified
	}

	entries, err := ParseWalkFile(filename)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrFailedToParseWalkFile, err)
	}

	// When a device declares topology links (trunk_ports), NIAC synthesizes the
	// neighbour/forwarding tables from them (see initializeNeighborMIBs, run at
	// construction). A capture walk carries the *original* device's neighbours,
	// which are foreign to this lab — so skip those tables on load and let the
	// authored topology win. Without trunk_ports the walk's data stands.
	skipTopology := a.ownsSynthesizedTopology()

	loaded, skipped := 0, 0
	for _, entry := range entries {
		if skipTopology && isSynthesizedTopologyOID(entry.OID) {
			skipped++

			continue
		}
		a.mib.Set(entry.OID, &OIDValue{
			Type:  entry.Type,
			Value: entry.Value,
		})
		loaded++
	}

	if a.debugLevel >= 1 {
		a.logger.Debug(
			"Loaded OIDs from walk file",
			"count", loaded,
			"skipped_topology", skipped,
			"filename", filename,
			"device", a.device.Name,
		)
	}

	return nil
}

// ownsSynthesizedTopology reports whether this device's topology is authored
// via trunk_ports (in which case walk topology tables are skipped).
func (a *Agent) ownsSynthesizedTopology() bool {
	return a.device != nil && len(a.device.TrunkPorts) > 0
}

// isSynthesizedTopologyOID reports whether oid falls under a MIB subtree that
// trunk_ports synthesis owns — LLDP remote systems, CDP cache, and the bridge
// forwarding DB — and so must not be loaded from a capture walk. Handles the
// optional leading dot in walk OIDs.
func isSynthesizedTopologyOID(oid string) bool {
	prefixes := []string{
		lldpRemoteSystemsData, // 1.0.8802.1.1.2.1.4 — LLDP-MIB neighbours
		cdpCache,              // 1.3.6.1.4.1.9.9.23.1.2 — CDP cache neighbours
		dot1dTpFdbTable,       // 1.3.6.1.2.1.17.4.3 — bridge MAC→port
	}

	trimmed := strings.TrimPrefix(oid, ".")
	for _, prefix := range prefixes {
		if trimmed == prefix || strings.HasPrefix(trimmed, prefix+".") {
			return true
		}
	}

	return false
}

// HandleGet processes an SNMP GET request.
func (a *Agent) HandleGet(oid string) (*OIDValue, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	value := a.mib.Get(oid)
	if value == nil {
		return nil, fmt.Errorf("%w: %s", ErrNoSuchObject, oid)
	}

	if a.debugLevel >= debugLevelDebug {
		a.logger.Debug("SNMP GET", "oid", oid, "value", value.Value, "device", a.device.Name)
	}

	return value, nil
}

// HandleGetNext processes an SNMP GET-NEXT request.
func (a *Agent) HandleGetNext(oid string) (string, *OIDValue, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	nextOID, value := a.mib.GetNext(oid)
	if nextOID == "" || value == nil {
		return "", nil, ErrEndOfMIBView
	}

	if a.debugLevel >= debugLevelDebug {
		a.logger.Debug(
			"SNMP GET-NEXT",
			"oid",
			oid,
			"next_oid",
			nextOID,
			"value",
			value.Value,
			"device",
			a.device.Name,
		)
	}

	return nextOID, value, nil
}

// MaxBulkResponseBytes bounds the estimated size of a GET-BULK response's
// variable bindings so the marshaled datagram stays inside a single standard
// Ethernet frame (1500-byte MTU, less L2/L3/L4 and SNMP message headers). A
// conforming agent returns fewer than maxRepetitions bindings when the message
// would overflow and lets the manager continue the walk from the last OID.
// Without this cap, a run of large values (e.g. Cisco AGENT-CAPABILITIES
// description strings, ~250 bytes each) produces an oversized datagram that is
// silently dropped on the wire, stalling every bulk walk at that point.
const MaxBulkResponseBytes = 1400

// HandleGetBulk processes an SNMP GET-BULK request.
func (a *Agent) HandleGetBulk(oid string, maxRepetitions int) ([]OIDResult, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	results := make([]OIDResult, 0, maxRepetitions)
	currentOID := oid
	budget := MaxBulkResponseBytes

	for range maxRepetitions {
		nextOID, value := a.mib.GetNext(currentOID)
		if nextOID == "" || value == nil {
			break
		}

		// Always include at least one binding so the walk makes progress even
		// when a single value is larger than the whole budget; stop before
		// exceeding the frame budget thereafter.
		size := estimateVarbindSize(nextOID, value)
		if len(results) > 0 && size > budget {
			break
		}

		budget -= size

		results = append(results, OIDResult{
			OID:   nextOID,
			Value: value,
		})

		currentOID = nextOID
	}

	if a.debugLevel >= debugLevelDebug {
		a.logger.Debug(
			"SNMP GET-BULK",
			"oid",
			oid,
			"max_repetitions",
			maxRepetitions,
			"results",
			len(results),
			"device",
			a.device.Name,
		)
	}

	return results, nil
}

// estimateVarbindSize returns a deliberately conservative (over-)estimate of a
// variable binding's marshaled byte size: the OID string length is a safe proxy
// for its BER-encoded sub-identifiers, plus the value payload, plus TLV
// overhead. Over-estimating keeps GET-BULK datagrams comfortably under the MTU.
func estimateVarbindSize(oid string, value *OIDValue) int {
	const tlvOverhead = 8

	size := len(oid) + tlvOverhead

	switch v := value.Value.(type) {
	case string:
		size += len(v)
	case []byte:
		size += len(v)
	default:
		// Integers, counters, gauges, timeticks, IP addresses and OIDs all
		// encode to a small, bounded number of bytes.
		size += 16
	}

	return size
}

// SetOID sets an OID value (for SNMP SET operations).
func (a *Agent) SetOID(oid string, value *OIDValue) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Check if OID is writable
	// For now, allow setting any OID
	// In a full implementation, you'd check write permissions

	a.mib.Set(oid, value)

	if a.debugLevel >= debugLevelVerbose {
		a.logger.Debug("SNMP SET", "oid", oid, "value", value.Value, "device", a.device.Name)
	}

	return nil
}

// GetCommunity returns the agent's community string.
func (a *Agent) GetCommunity() string {
	return a.community
}

// RedactedCommunity returns a redacted version for safe logging
// SECURITY FIX MEDIUM-5: Prevent community string exposure in logs.
func (a *Agent) RedactedCommunity() string {
	if len(a.community) == 0 {
		return "[EMPTY]"
	}

	if len(a.community) <= minRedactLen {
		return "**"
	}
	// Show first and last character only
	return string(
		a.community[0],
	) + strings.Repeat(
		"*",
		len(a.community)-minRedactLen,
	) + string(
		a.community[len(a.community)-1],
	)
}

// RedactCommunityString redacts a community string for safe logging
// SECURITY FIX MEDIUM-5: Helper function to redact any community string.
func RedactCommunityString(community string) string {
	if len(community) == 0 {
		return "[EMPTY]"
	}

	if len(community) <= minRedactLen {
		return "**"
	}

	return string(community[0]) + strings.Repeat("*", len(community)-minRedactLen) + string(community[len(community)-1])
}

// ProcessPDU processes SNMP PDU variables and returns response variables
// This is typically called by an SNMP server implementation.
func (a *Agent) ProcessPDU(pduType gosnmp.PDUType, vars []gosnmp.SnmpPDU, maxRepetitions uint32) []gosnmp.SnmpPDU {
	switch pduType {
	case gosnmp.GetRequest:
		return a.processGetRequest(vars)
	case gosnmp.GetNextRequest:
		return a.processGetNextRequest(vars)
	case gosnmp.GetBulkRequest:
		reps := int(maxRepetitions)
		if reps <= 0 {
			reps = SNMPRetryCount
		}

		if reps > MaxOIDResultSize {
			reps = MaxOIDResultSize
		}

		return a.processGetBulkRequestVars(vars, reps)
	case gosnmp.Sequence,
		gosnmp.GetResponse,
		gosnmp.SetRequest,
		gosnmp.Trap,
		gosnmp.InformRequest,
		gosnmp.SNMPv2Trap,
		gosnmp.Report:
		// Return error PDU for unsupported PDU types
		name := ""
		if len(vars) > 0 {
			name = vars[0].Name
		}

		return []gosnmp.SnmpPDU{{
			Name:  name,
			Type:  gosnmp.NoSuchObject,
			Value: nil,
		}}
	default:
		// Unknown PDU type - return error response
		name := ""
		if len(vars) > 0 {
			name = vars[0].Name
		}

		return []gosnmp.SnmpPDU{{
			Name:  name,
			Type:  gosnmp.NoSuchObject,
			Value: nil,
		}}
	}
}

// processGetRequest processes GET request variables.
func (a *Agent) processGetRequest(vars []gosnmp.SnmpPDU) []gosnmp.SnmpPDU {
	response := make([]gosnmp.SnmpPDU, len(vars))

	for i, snmpVar := range vars {
		value, err := a.HandleGet(snmpVar.Name)
		if err != nil {
			response[i] = gosnmp.SnmpPDU{
				Name:  snmpVar.Name,
				Type:  gosnmp.NoSuchObject,
				Value: nil,
			}
		} else {
			response[i] = gosnmp.SnmpPDU{
				Name:  snmpVar.Name,
				Type:  value.Type,
				Value: value.Value,
			}
		}
	}

	return response
}

// processGetNextRequest processes GET-NEXT request variables.
func (a *Agent) processGetNextRequest(vars []gosnmp.SnmpPDU) []gosnmp.SnmpPDU {
	response := make([]gosnmp.SnmpPDU, len(vars))

	for i, snmpVar := range vars {
		nextOID, value, err := a.HandleGetNext(snmpVar.Name)
		if err != nil {
			response[i] = gosnmp.SnmpPDU{
				Name:  snmpVar.Name,
				Type:  gosnmp.EndOfMibView,
				Value: nil,
			}
		} else {
			response[i] = gosnmp.SnmpPDU{
				Name:  nextOID,
				Type:  value.Type,
				Value: value.Value,
			}
		}
	}

	return response
}

// processGetBulkRequestVars processes GET-BULK request variables.
func (a *Agent) processGetBulkRequestVars(vars []gosnmp.SnmpPDU, maxRepetitions int) []gosnmp.SnmpPDU {
	var response []gosnmp.SnmpPDU

	for _, snmpVar := range vars {
		results, err := a.HandleGetBulk(snmpVar.Name, maxRepetitions)
		if err != nil {
			response = append(response, gosnmp.SnmpPDU{
				Name:  snmpVar.Name,
				Type:  gosnmp.EndOfMibView,
				Value: nil,
			})

			continue
		}

		for _, result := range results {
			response = append(response, gosnmp.SnmpPDU{
				Name:  result.OID,
				Type:  result.Value.Type,
				Value: result.Value.Value,
			})
		}
	}

	return response
}

// OIDResult represents an OID and its value.
type OIDResult struct {
	OID   string
	Value *OIDValue
}

// FormatIP formats an IP address for display.
func FormatIP(ip net.IP) string {
	return ip.String()
}

// ParseOID parses an OID string and validates it.
func ParseOID(oid string) ([]int, error) {
	if oid == "" {
		return nil, ErrEmptyOID
	}

	// Remove leading dot if present
	oid = strings.TrimPrefix(oid, ".")

	parts := strings.Split(oid, ".")
	result := make([]int, len(parts))

	for i, part := range parts {
		var num int

		_, err := fmt.Sscanf(part, "%d", &num)
		if err != nil {
			return nil, fmt.Errorf("%w: %s", ErrInvalidOIDComponent, part)
		}

		result[i] = num
	}

	return result, nil
}
