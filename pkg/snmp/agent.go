// Package snmp implements SNMP agent functionality including MIB management and trap sending
package snmp

import (
	"fmt"
	"log"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/gosnmp/gosnmp"
	"github.com/krisarmstrong/niac-go/pkg/config"
)

// Agent represents an SNMP agent instance for a device
type Agent struct {
	device     *config.Device
	mib        *MIB
	community  string
	startTime  time.Time
	debugLevel int
	mu         sync.RWMutex
}

// NewAgent creates a new SNMP agent for a device
func NewAgent(device *config.Device, debugLevel int) *Agent {
	agent := &Agent{
		device:     device,
		mib:        NewMIB(),
		community:  "public",
		startTime:  time.Now(),
		debugLevel: debugLevel,
	}

	// Set community from device config if available
	if device.SNMPConfig.Community != "" {
		agent.community = device.SNMPConfig.Community
	}

	// Initialize standard MIB-II system objects
	agent.initializeSystemMIB()

	return agent
}

// initializeSystemMIB initializes standard MIB-II system group OIDs
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
		sysObjectID = "1.3.6.1.4.1.9.1.1" // Default to generic Cisco
	}
	a.mib.Set("1.3.6.1.2.1.1.2.0", &OIDValue{
		Type:  gosnmp.ObjectIdentifier,
		Value: sysObjectID,
	})

	// sysUpTime (1.3.6.1.2.1.1.3.0) - TimeTicks (hundredths of second)
	a.mib.SetDynamic("1.3.6.1.2.1.1.3.0", func() *OIDValue {
		uptime := time.Since(a.startTime)
		timeticks := uint32(uptime.Milliseconds() / 10) // Convert to hundredths of second
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
		sysLocation = "Unknown"
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

	if a.debugLevel >= 2 {
		log.Printf("Initialized system MIB for device %s", a.device.Name)
	}
}

// LoadWalkFile loads SNMP walk file data into the MIB
func (a *Agent) LoadWalkFile(filename string) error {
	if filename == "" {
		return fmt.Errorf("no walk file specified")
	}

	entries, err := ParseWalkFile(filename)
	if err != nil {
		return fmt.Errorf("failed to parse walk file: %v", err)
	}

	// Add all entries to MIB
	for _, entry := range entries {
		a.mib.Set(entry.OID, &OIDValue{
			Type:  entry.Type,
			Value: entry.Value,
		})
	}

	if a.debugLevel >= 1 {
		log.Printf("Loaded %d OIDs from walk file %s for device %s",
			len(entries), filename, a.device.Name)
	}

	return nil
}

// HandleGet processes an SNMP GET request
func (a *Agent) HandleGet(oid string) (*OIDValue, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	value := a.mib.Get(oid)
	if value == nil {
		return nil, fmt.Errorf("no such object: %s", oid)
	}

	if a.debugLevel >= 3 {
		log.Printf("SNMP GET %s = %v (device: %s)", oid, value.Value, a.device.Name)
	}

	return value, nil
}

// HandleGetNext processes an SNMP GET-NEXT request
func (a *Agent) HandleGetNext(oid string) (string, *OIDValue, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	nextOID, value := a.mib.GetNext(oid)
	if nextOID == "" || value == nil {
		return "", nil, fmt.Errorf("end of MIB view")
	}

	if a.debugLevel >= 3 {
		log.Printf("SNMP GET-NEXT %s -> %s = %v (device: %s)",
			oid, nextOID, value.Value, a.device.Name)
	}

	return nextOID, value, nil
}

// HandleGetBulk processes an SNMP GET-BULK request
func (a *Agent) HandleGetBulk(oid string, maxRepetitions int) ([]OIDResult, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	results := make([]OIDResult, 0, maxRepetitions)
	currentOID := oid

	for i := 0; i < maxRepetitions; i++ {
		nextOID, value := a.mib.GetNext(currentOID)
		if nextOID == "" || value == nil {
			break
		}

		results = append(results, OIDResult{
			OID:   nextOID,
			Value: value,
		})

		currentOID = nextOID
	}

	if a.debugLevel >= 3 {
		log.Printf("SNMP GET-BULK %s (max=%d) returned %d results (device: %s)",
			oid, maxRepetitions, len(results), a.device.Name)
	}

	return results, nil
}

// SetOID sets an OID value (for SNMP SET operations)
func (a *Agent) SetOID(oid string, value *OIDValue) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Check if OID is writable
	// For now, allow setting any OID
	// In a full implementation, you'd check write permissions

	a.mib.Set(oid, value)

	if a.debugLevel >= 2 {
		log.Printf("SNMP SET %s = %v (device: %s)", oid, value.Value, a.device.Name)
	}

	return nil
}

// GetCommunity returns the agent's community string
func (a *Agent) GetCommunity() string {
	return a.community
}

// ProcessPDU processes SNMP PDU variables and returns response variables
// This is typically called by an SNMP server implementation
func (a *Agent) ProcessPDU(pduType gosnmp.PDUType, vars []gosnmp.SnmpPDU) []gosnmp.SnmpPDU {
	switch pduType {
	case gosnmp.GetRequest:
		return a.processGetRequest(vars)
	case gosnmp.GetNextRequest:
		return a.processGetNextRequest(vars)
	case gosnmp.GetBulkRequest:
		// For bulk, we need maxRepetitions which would be passed separately
		return a.processGetBulkRequestVars(vars, 10) // Default to 10
	default:
		// Return error PDU
		return []gosnmp.SnmpPDU{{
			Name:  vars[0].Name,
			Type:  gosnmp.NoSuchObject,
			Value: nil,
		}}
	}
}

// processGetRequest processes GET request variables
func (a *Agent) processGetRequest(vars []gosnmp.SnmpPDU) []gosnmp.SnmpPDU {
	response := make([]gosnmp.SnmpPDU, len(vars))

	for i, v := range vars {
		value, err := a.HandleGet(v.Name)
		if err != nil {
			response[i] = gosnmp.SnmpPDU{
				Name:  v.Name,
				Type:  gosnmp.NoSuchObject,
				Value: nil,
			}
		} else {
			response[i] = gosnmp.SnmpPDU{
				Name:  v.Name,
				Type:  value.Type,
				Value: value.Value,
			}
		}
	}

	return response
}

// processGetNextRequest processes GET-NEXT request variables
func (a *Agent) processGetNextRequest(vars []gosnmp.SnmpPDU) []gosnmp.SnmpPDU {
	response := make([]gosnmp.SnmpPDU, len(vars))

	for i, v := range vars {
		nextOID, value, err := a.HandleGetNext(v.Name)
		if err != nil {
			response[i] = gosnmp.SnmpPDU{
				Name:  v.Name,
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

// processGetBulkRequestVars processes GET-BULK request variables
func (a *Agent) processGetBulkRequestVars(vars []gosnmp.SnmpPDU, maxRepetitions int) []gosnmp.SnmpPDU {
	var response []gosnmp.SnmpPDU

	for _, v := range vars {
		results, err := a.HandleGetBulk(v.Name, maxRepetitions)
		if err != nil {
			response = append(response, gosnmp.SnmpPDU{
				Name:  v.Name,
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

// OIDResult represents an OID and its value
type OIDResult struct {
	OID   string
	Value *OIDValue
}

// FormatIP formats an IP address for display
func FormatIP(ip net.IP) string {
	return ip.String()
}

// ParseOID parses an OID string and validates it
func ParseOID(oid string) ([]int, error) {
	if oid == "" {
		return nil, fmt.Errorf("empty OID")
	}

	// Remove leading dot if present
	oid = strings.TrimPrefix(oid, ".")

	parts := strings.Split(oid, ".")
	result := make([]int, len(parts))

	for i, part := range parts {
		var num int
		_, err := fmt.Sscanf(part, "%d", &num)
		if err != nil {
			return nil, fmt.Errorf("invalid OID component: %s", part)
		}
		result[i] = num
	}

	return result, nil
}
