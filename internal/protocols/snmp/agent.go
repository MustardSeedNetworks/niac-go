// Package snmp implements SNMP agent functionality including MIB management and trap sending.
package snmp

import (
	"fmt"
	"log/slog"
	"net"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gosnmp/gosnmp"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
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
	device          *config.Device
	mib             *MIB
	community       string
	startTime       time.Time
	debugLevel      int
	logger          *slog.Logger
	mu              sync.RWMutex
	stats           snmpStats
	protocolStats   *ProtocolTelemetry
	deviceState     *devicestate.Store
	stateMIBVersion atomic.Uint64
	stateIPOIDs     map[string]struct{}
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
	return NewAgentWithCommunityAndTelemetry(device, community, debugLevel, NewProtocolTelemetry())
}

// NewAgentWithCommunityAndTelemetry creates an agent backed by shared per-device telemetry.
func NewAgentWithCommunityAndTelemetry(
	device *config.Device,
	community string,
	debugLevel int,
	telemetry *ProtocolTelemetry,
) *Agent {
	agent := &Agent{
		device:        device,
		mib:           NewMIB(),
		community:     community,
		startTime:     time.Now(),
		debugLevel:    debugLevel,
		logger:        slog.Default(),
		protocolStats: telemetry,
	}
	if agent.community == "" {
		agent.community = config.DefaultSNMPCommunity
	}

	// Initialize standard MIB-II system objects
	agent.initializeSystemMIB()
	agent.initializeSNMPMIB()
	agent.initializeMIBIIProtocolGroups()

	// Initialize neighbor discovery MIBs (IF-MIB, LLDP-MIB, CDP-MIB)
	agent.initializeNeighborMIBs()
	agent.protocolStats.attachMIB(agent.mib)

	return agent
}

// initializeSystemMIB initializes standard MIB-II system group OIDs.
func (a *Agent) initializeSystemMIB() {
	// sysDescr (1.3.6.1.2.1.1.1.0)
	sysDescr := a.device.SNMPConfig.SysDescr
	if sysDescr == "" {
		sysDescr = a.device.Properties["sysDescr"]
	}
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
	sysContact := a.device.SNMPConfig.SysContact
	if sysContact == "" {
		sysContact = a.device.Properties["sysContact"]
	}
	if sysContact == "" {
		sysContact = "admin@example.com"
	}

	a.mib.Set("1.3.6.1.2.1.1.4.0", &OIDValue{
		Type:  gosnmp.OctetString,
		Value: sysContact,
	})

	// sysName (1.3.6.1.2.1.1.5.0)
	sysName := a.device.SNMPConfig.SysName
	if sysName == "" {
		sysName = a.device.Properties["sysName"]
	}
	if sysName == "" {
		sysName = a.device.Name
	}

	a.mib.Set("1.3.6.1.2.1.1.5.0", &OIDValue{
		Type:  gosnmp.OctetString,
		Value: sysName,
	})

	// sysLocation (1.3.6.1.2.1.1.6.0)
	sysLocation := a.device.SNMPConfig.SysLocation
	if sysLocation == "" {
		sysLocation = a.device.Properties["sysLocation"]
	}
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

// Reindex eagerly builds the agent's sorted OID index. Call after all MIB
// content is loaded so the sort is paid at startup rather than lazily on the
// first GetNext (which runs on the stack's single decode goroutine). See
// MIB.Reindex.
func (a *Agent) Reindex() {
	a.mib.Reindex()
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
		// A device's identity (sysName) is authored from its config name, not the
		// captured device's name — otherwise every device sharing a walk collides
		// under one name and a discovery tool merges them. sysDescr/location stay
		// as the walk's authentic values.
		if a.isAuthoredIdentityOID(entry.OID) || isAgentOwnedSNMPOID(entry.OID) || isLiveProtocolOID(entry.OID) {
			skipped++

			continue
		}
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
	if a.mib.Get(dot1dBaseNumPorts) == nil {
		a.initializeBridgeMIB()
	}
	a.registerLiveMIBIIProtocolCounters()
	a.refreshBridgePortCounters()
	a.refreshAuthoredInterfaceMIBs()
	if a.ownsSynthesizedTopology() {
		a.refreshAuthoredDiscoveryMIBs()
	}
	// The walk owns the authoritative IF-MIB indexes. Rebuild the configured
	// route rows after loading it so ipRouteIfIndex references those indexes.
	a.registerConfiguredRoutes(a.device)

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

func isLiveProtocolOID(oid string) bool {
	oid = strings.TrimPrefix(oid, ".")
	for _, root := range []string{ipMIBBase, icmpMIBRoot, tcpMIBRoot, udpMIBRoot, egpMIBRoot} {
		if oid == root || strings.HasPrefix(oid, root+".") {
			return true
		}
	}
	return false
}

func isAgentOwnedSNMPOID(oid string) bool {
	oid = strings.TrimPrefix(oid, ".")
	return oid == snmpGroup || strings.HasPrefix(oid, snmpGroup+".")
}

// sysNameOID is SNMPv2-MIB::sysName.0 — a device's administrative identity,
// always authored from the config rather than the capture walk.
const (
	authoredSysDescrOID    = "1.3.6.1.2.1.1.1.0"
	authoredSysContactOID  = "1.3.6.1.2.1.1.4.0"
	sysNameOID             = "1.3.6.1.2.1.1.5.0"
	authoredSysLocationOID = "1.3.6.1.2.1.1.6.0"
)

// isAuthoredSysNameOID reports whether oid is sysName.0 (with or without the
// leading dot walks use).
func isAuthoredSysNameOID(oid string) bool {
	return strings.TrimPrefix(oid, ".") == sysNameOID
}

func (a *Agent) isAuthoredIdentityOID(oid string) bool {
	oid = strings.TrimPrefix(oid, ".")
	if isAuthoredSysNameOID(oid) {
		return true
	}
	return (oid == authoredSysDescrOID && a.device.SNMPConfig.SysDescr != "") ||
		(oid == authoredSysContactOID && a.device.SNMPConfig.SysContact != "") ||
		(oid == authoredSysLocationOID && a.device.SNMPConfig.SysLocation != "")
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

	return slices.ContainsFunc(prefixes, func(prefix string) bool {
		return trimmed == prefix || strings.HasPrefix(trimmed, prefix+".")
	})
}

// HandleGet processes an SNMP GET request.
func (a *Agent) HandleGet(oid string) (*OIDValue, error) {
	a.syncDeviceStateMIBs()
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
	a.syncDeviceStateMIBs()
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
	a.syncDeviceStateMIBs()
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
	oid = strings.TrimPrefix(oid, ".")
	if oid == snmpGroup+".30.0" {
		enabled, ok := snmpInteger(value.Value)
		if !ok || (enabled != authenticationTrapsOn && enabled != authenticationTrapsOff) {
			return fmt.Errorf("%w: snmpEnableAuthenTraps must be enabled(1) or disabled(2)", ErrInvalidValue)
		}
		a.stats.enableAuthenTraps.Store(uint32(enabled))
		return nil
	}

	// Check if OID is writable
	// For now, allow setting any OID
	// In a full implementation, you'd check write permissions

	a.mib.Set(oid, value)

	if a.debugLevel >= debugLevelVerbose {
		a.logger.Debug("SNMP SET", "oid", oid, "value", value.Value, "device", a.device.Name)
	}

	return nil
}

func snmpInteger(value any) (int, bool) {
	switch n := value.(type) {
	case int:
		return n, true
	case int32:
		return int(n), true
	case int64:
		return int(n), true
	default:
		return 0, false
	}
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
func (a *Agent) ProcessPDU(
	pduType gosnmp.PDUType,
	vars []gosnmp.SnmpPDU,
	nonRepeaters int,
	maxRepetitions uint32,
) []gosnmp.SnmpPDU {
	a.recordInboundPDU(pduType, len(vars))

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

		reps = min(reps, MaxOIDResultSize)

		return a.processGetBulk(vars, nonRepeaters, reps)
	case gosnmp.SetRequest:
		return a.processSetRequest(vars)
	case gosnmp.Sequence,
		gosnmp.GetResponse,
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

// ProcessRequest processes a v1/v2c request and derives the response status
// required by the version's error model.
func (a *Agent) ProcessRequest(request *gosnmp.SnmpPacket) ([]gosnmp.SnmpPDU, gosnmp.SNMPError, uint8) {
	variables := a.ProcessPDU(
		request.PDUType, request.Variables, int(request.NonRepeaters), request.MaxRepetitions,
	)
	for i, variable := range variables {
		if request.PDUType == gosnmp.SetRequest && variable.Type == gosnmp.NoSuchObject {
			return variables, gosnmp.BadValue, uint8(i + 1)
		}
		if request.Version == gosnmp.Version1 &&
			(variable.Type == gosnmp.NoSuchObject || variable.Type == gosnmp.EndOfMibView) {
			return request.Variables, gosnmp.NoSuchName, uint8(i + 1)
		}
	}
	return variables, gosnmp.NoError, 0
}

func (a *Agent) processSetRequest(vars []gosnmp.SnmpPDU) []gosnmp.SnmpPDU {
	response := make([]gosnmp.SnmpPDU, len(vars))
	for i, variable := range vars {
		value := &OIDValue{Type: variable.Type, Value: variable.Value}
		if err := a.SetOID(variable.Name, value); err != nil {
			response[i] = gosnmp.SnmpPDU{Name: variable.Name, Type: gosnmp.NoSuchObject}
			continue
		}
		response[i] = variable
	}
	return response
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

// processGetBulk implements SNMP GET-BULK per RFC 3416 §4.2.3.
//
// The first nonRepeaters varbinds are advanced once (like GET-NEXT); the rest
// are "repeaters" walked up to maxRepetitions times. Crucially, the repeated
// bindings are emitted row-major — for each repetition, the next binding of
// every repeater in request order — not column-major (each repeater walked to
// completion before the next). A manager reconstructs a conceptual table by
// position, so the earlier column-major layout made a multi-column ifTable walk
// (what a NetAlly CyberScope issues) unparseable: it reported a single interface
// even though the agent served the whole table. All bindings share one datagram
// byte budget so the response stays under the MTU.
func (a *Agent) processGetBulk(vars []gosnmp.SnmpPDU, nonRepeaters, maxRepetitions int) []gosnmp.SnmpPDU {
	a.mu.RLock()
	defer a.mu.RUnlock()

	nonRepeaters = max(0, min(nonRepeaters, len(vars)))

	response := make([]gosnmp.SnmpPDU, 0, len(vars))
	budget := MaxBulkResponseBytes

	// Non-repeaters: advance each once, in order.
	for _, v := range vars[:nonRepeaters] {
		nextOID, value := a.mib.GetNext(v.Name)
		response = append(response, bulkBinding(v.Name, nextOID, value))

		if value != nil {
			budget -= estimateVarbindSize(nextOID, value)
		}
	}

	repeaters := vars[nonRepeaters:]
	cursors := make([]string, len(repeaters))
	exhausted := make([]bool, len(repeaters))

	for i, v := range repeaters {
		cursors[i] = v.Name
	}

	// Repeaters: interleave repetitions (row-major).
	for range maxRepetitions {
		progressed := false

		for i := range repeaters {
			if exhausted[i] {
				continue
			}

			nextOID, value := a.mib.GetNext(cursors[i])
			if nextOID == "" || value == nil {
				exhausted[i] = true

				response = append(response, gosnmp.SnmpPDU{
					Name:  cursors[i],
					Type:  gosnmp.EndOfMibView,
					Value: nil,
				})

				continue
			}

			// Keep the datagram under the MTU; the manager resumes the walk from
			// the last returned OIDs. Always allow the very first binding so a
			// single oversized value never deadlocks the walk.
			size := estimateVarbindSize(nextOID, value)
			if len(response) > 0 && size > budget {
				return response
			}

			budget -= size
			response = append(response, bulkBinding(cursors[i], nextOID, value))
			cursors[i] = nextOID
			progressed = true
		}

		if !progressed {
			break
		}
	}

	return response
}

// bulkBinding builds a response binding for a GET-BULK step, mapping an
// exhausted subtree (no next OID) to the endOfMibView exception.
func bulkBinding(requestedOID, nextOID string, value *OIDValue) gosnmp.SnmpPDU {
	if nextOID == "" || value == nil {
		return gosnmp.SnmpPDU{Name: requestedOID, Type: gosnmp.EndOfMibView, Value: nil}
	}

	return gosnmp.SnmpPDU{Name: nextOID, Type: value.Type, Value: value.Value}
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
