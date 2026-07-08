// Package walkanalysis extracts a structured view of a captured SNMP walk:
// device identity, interface inventory, and LLDP/CDP neighbor adjacencies.
//
// It is built on snmp.ParseWalkFile, which yields typed (OID, value) entries,
// and walks the standard MIB trees directly — IF-MIB/ifXTable for interfaces,
// SNMPv2-MIB for system identity, and LLDP-MIB/CISCO-CDP-MIB remote tables for
// neighbors. The neighbor extraction is the exact inverse of the simulator's
// synthesis side (internal/protocols/snmp/mib_discovery.go); a round-trip test
// keeps the two in sync.
package walkanalysis

import (
	"maps"
	"math"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/MustardSeedNetworks/niac-go/internal/protocols/snmp"
)

// Analysis is the structured result of parsing a walk file. JSON/YAML tags are
// preserved from the original CLI analyzer so `niac analyze-walk` output is
// unchanged.
type Analysis struct {
	Device     Device      `json:"device"     yaml:"device"`
	Interfaces []Interface `json:"interfaces" yaml:"interfaces"`
	Neighbors  []Neighbor  `json:"neighbors"  yaml:"neighbors"`
	Statistics Stats       `json:"statistics" yaml:"statistics"`
}

// Device holds device identity parsed from SNMPv2-MIB (1.3.6.1.2.1.1).
type Device struct {
	SysName     string `json:"sysname"               yaml:"sysname"`
	SysDescr    string `json:"sysdescr"              yaml:"sysdescr"`
	SysObjectID string `json:"sysobjectid"           yaml:"sysobjectid"`
	SysContact  string `json:"syscontact,omitempty"  yaml:"syscontact,omitempty"`
	SysLocation string `json:"syslocation,omitempty" yaml:"syslocation,omitempty"`
}

// Interface holds one row of the IF-MIB interface table, enriched from ifXTable.
type Interface struct {
	Index       int    `json:"index"                  yaml:"index"`
	Name        string `json:"name"                   yaml:"name"`
	Description string `json:"description,omitempty"  yaml:"description,omitempty"`
	Type        string `json:"type"                   yaml:"type"`
	Speed       int64  `json:"speed,omitempty"        yaml:"speed,omitempty"`
	AdminStatus string `json:"admin_status,omitempty" yaml:"admin_status,omitempty"`
	OperStatus  string `json:"oper_status,omitempty"  yaml:"oper_status,omitempty"`
	MACAddress  string `json:"mac_address,omitempty"  yaml:"mac_address,omitempty"`
}

// Neighbor is a discovered adjacency from LLDP-MIB or CISCO-CDP-MIB.
type Neighbor struct {
	LocalInterface  string `json:"local_interface"  yaml:"local_interface"`
	RemoteDevice    string `json:"remote_device"    yaml:"remote_device"`
	RemoteInterface string `json:"remote_interface" yaml:"remote_interface"`
	Protocol        string `json:"protocol"         yaml:"protocol"`
}

// Stats summarizes the analysis.
type Stats struct {
	TotalInterfaces    int `json:"total_interfaces"    yaml:"total_interfaces"`
	PhysicalInterfaces int `json:"physical_interfaces" yaml:"physical_interfaces"`
	LogicalInterfaces  int `json:"logical_interfaces"  yaml:"logical_interfaces"`
	TotalNeighbors     int `json:"total_neighbors"     yaml:"total_neighbors"`
}

// Neighbor discovery protocol labels.
const (
	protocolLLDP = "lldp"
	protocolCDP  = "cdp"
)

// System scalar OIDs (SNMPv2-MIB, 1.3.6.1.2.1.1), instance .0.
const (
	oidSysDescr    = "1.3.6.1.2.1.1.1.0"
	oidSysObjectID = "1.3.6.1.2.1.1.2.0"
	oidSysContact  = "1.3.6.1.2.1.1.4.0"
	oidSysName     = "1.3.6.1.2.1.1.5.0"
	oidSysLocation = "1.3.6.1.2.1.1.6.0"
)

// Table prefixes (with trailing dot). An entry OID under a prefix is
// "<prefix><column>.<index...>".
const (
	ifTablePrefix     = "1.3.6.1.2.1.2.2.1."      // ifEntry
	ifXTablePrefix    = "1.3.6.1.2.1.31.1.1.1."   // ifXEntry
	lldpRemPrefix     = "1.0.8802.1.1.2.1.4.1.1." // lldpRemEntry
	lldpLocPortIDBase = "1.0.8802.1.1.2.1.3.7.1.3."
	cdpCachePrefix    = "1.3.6.1.4.1.9.9.23.1.2.1.1." // cdpCacheEntry
)

// IF-MIB / ifXTable column numbers.
const (
	ifColDescr       = "2"
	ifColType        = "3"
	ifColSpeed       = "5"
	ifColPhysAddress = "6"
	ifColAdminStatus = "7"
	ifColOperStatus  = "8"

	ifxColName      = "1"
	ifxColHighSpeed = "15"
	ifxColAlias     = "18"
)

// LLDP-MIB lldpRemTable / CISCO-CDP-MIB cdpCacheTable column numbers.
const (
	lldpColChassisID = "5"
	lldpColPortID    = "7"
	lldpColPortDesc  = "8"
	lldpColSysName   = "9"

	cdpColDeviceID   = "6"
	cdpColDevicePort = "7"
)

// bitsPerMbit converts ifHighSpeed (Mbps) to bits/second. uint32SpeedCeiling is
// the ifSpeed value net-snmp reports for links faster than the 32-bit counter
// (~4.29 Gbps), signalling that ifHighSpeed should be consulted instead.
const (
	bitsPerMbit        = 1_000_000
	uint32SpeedCeiling = int64(4294967295)
)

// ianaIfTypeNames maps common IANAifType values (IF-MIB ifType) to readable
// names. physicalIfTypes marks which of those are physical ports for the
// physical/logical interface tally.
var (
	ianaIfTypeNames = map[int]string{
		1:   "other",
		6:   "ethernetCsmacd",
		18:  "ds1",
		22:  "propPointToPointSerial",
		23:  "ppp",
		24:  "softwareLoopback",
		32:  "frameRelay",
		53:  "propVirtual",
		62:  "fastEther",
		69:  "fastEtherFX",
		117: "gigabitEthernet",
		131: "tunnel",
		135: "l2vlan",
		136: "l3ipvlan",
		161: "ieee8023adLag",
		209: "bridge",
	}
	physicalIfTypes = map[int]bool{
		6:   true,
		62:  true,
		69:  true,
		117: true,
	}
	adminStatusNames = map[int]string{1: "up", 2: "down", 3: "testing"}
	operStatusNames  = map[int]string{
		1: "up", 2: "down", 3: "testing", 4: "unknown",
		5: "dormant", 6: "notPresent", 7: "lowerLayerDown",
	}
)

// AnalyzeFile parses the walk file at path and returns its analysis. Path
// traversal, symlink, and regular-file checks are enforced by snmp.ParseWalkFile.
func AnalyzeFile(path string) (*Analysis, error) {
	entries, err := snmp.ParseWalkFile(path)
	if err != nil {
		return nil, err
	}

	return Analyze(entries), nil
}

// Analyze extracts device, interface, and neighbor data from parsed walk
// entries. It never returns nil; an empty walk yields an empty Analysis.
func Analyze(entries []snmp.WalkEntry) *Analysis {
	byOID := make(map[string]snmp.WalkEntry, len(entries))
	for _, e := range entries {
		byOID[normalizeOID(e.OID)] = e
	}

	interfaces, nameByIndex, physical := extractInterfaces(entries)
	neighbors := extractNeighbors(entries, nameByIndex)

	return &Analysis{
		Device:     extractDevice(byOID),
		Interfaces: interfaces,
		Neighbors:  neighbors,
		Statistics: Stats{
			TotalInterfaces:    len(interfaces),
			PhysicalInterfaces: physical,
			LogicalInterfaces:  len(interfaces) - physical,
			TotalNeighbors:     len(neighbors),
		},
	}
}

func extractDevice(byOID map[string]snmp.WalkEntry) Device {
	return Device{
		SysDescr:    lookupString(byOID, oidSysDescr),
		SysObjectID: lookupString(byOID, oidSysObjectID),
		SysContact:  lookupString(byOID, oidSysContact),
		SysName:     lookupString(byOID, oidSysName),
		SysLocation: lookupString(byOID, oidSysLocation),
	}
}

// ifAccum accumulates IF-MIB + ifXTable columns for a single ifIndex before the
// final Interface is built.
type ifAccum struct {
	descr         string
	name          string
	alias         string
	mac           string
	adminStatus   string
	operStatus    string
	ifType        int
	hasType       bool
	speedBps      int64
	hasSpeed      bool
	highSpeedMbps int64
	hasHighSpeed  bool
}

func extractInterfaces(entries []snmp.WalkEntry) ([]Interface, map[int]string, int) {
	accum := make(map[int]*ifAccum)

	for _, e := range entries {
		oid := normalizeOID(e.OID)
		if col, idx, ok := splitColIndex(oid, ifTablePrefix); ok {
			applyIfCol(accum, idx, col, e)
			continue
		}
		if col, idx, ok := splitColIndex(oid, ifXTablePrefix); ok {
			applyIfxCol(accum, idx, col, e)
		}
	}

	indices := slices.Sorted(maps.Keys(accum))

	nameByIndex := make(map[int]string, len(accum))
	list := make([]Interface, 0, len(accum))
	physical := 0
	for _, idx := range indices {
		iface, isPhysical := buildInterface(idx, accum[idx])
		if iface.Name == "" {
			continue
		}
		nameByIndex[idx] = iface.Name
		if isPhysical {
			physical++
		}
		list = append(list, iface)
	}

	return list, nameByIndex, physical
}

func applyIfCol(accum map[int]*ifAccum, idx, col string, e snmp.WalkEntry) {
	ifIdx, err := strconv.Atoi(idx)
	if err != nil {
		return
	}
	a := ensureAccum(accum, ifIdx)

	switch col {
	case ifColDescr:
		a.descr = entryString(e)
	case ifColType:
		if v, ok := entryInt(e); ok {
			a.ifType, a.hasType = v, true
		}
	case ifColSpeed:
		if v, ok := entryGauge(e); ok {
			a.speedBps, a.hasSpeed = v, true
		}
	case ifColPhysAddress:
		a.mac = formatMAC(entryString(e))
	case ifColAdminStatus:
		if v, ok := entryInt(e); ok {
			a.adminStatus = adminStatusNames[v]
		}
	case ifColOperStatus:
		if v, ok := entryInt(e); ok {
			a.operStatus = operStatusNames[v]
		}
	}
}

func applyIfxCol(accum map[int]*ifAccum, idx, col string, e snmp.WalkEntry) {
	ifIdx, err := strconv.Atoi(idx)
	if err != nil {
		return
	}
	a := ensureAccum(accum, ifIdx)

	switch col {
	case ifxColName:
		a.name = entryString(e)
	case ifxColAlias:
		a.alias = entryString(e)
	case ifxColHighSpeed:
		if v, ok := entryGauge(e); ok {
			a.highSpeedMbps, a.hasHighSpeed = v, true
		}
	}
}

func ensureAccum(accum map[int]*ifAccum, idx int) *ifAccum {
	a, ok := accum[idx]
	if !ok {
		a = &ifAccum{}
		accum[idx] = a
	}
	return a
}

func buildInterface(idx int, a *ifAccum) (Interface, bool) {
	name := a.name
	if name == "" {
		name = a.descr
	}

	typeName, physical := interfaceType(a, name)

	return Interface{
		Index:       idx,
		Name:        name,
		Description: a.alias,
		Type:        typeName,
		Speed:       interfaceSpeed(a),
		AdminStatus: a.adminStatus,
		OperStatus:  a.operStatus,
		MACAddress:  a.mac,
	}, physical
}

// interfaceSpeed prefers ifHighSpeed (Mbps) when ifSpeed is absent or pegged at
// the 32-bit ceiling (net-snmp reports 4294967295 for links faster than ~4Gbps).
func interfaceSpeed(a *ifAccum) int64 {
	if a.hasHighSpeed && a.highSpeedMbps > 0 &&
		(!a.hasSpeed || a.speedBps == 0 || a.speedBps == uint32SpeedCeiling) {
		return a.highSpeedMbps * bitsPerMbit
	}
	if a.hasSpeed {
		return a.speedBps
	}
	return 0
}

// interfaceType maps ifType (IANAifType) to a readable name and physical/logical
// class. When ifType is absent it falls back to a name-pattern heuristic.
func interfaceType(a *ifAccum, name string) (string, bool) {
	if a.hasType {
		if n, ok := ianaIfTypeNames[a.ifType]; ok {
			return n, physicalIfTypes[a.ifType]
		}
		return "ifType-" + strconv.Itoa(a.ifType), false
	}
	return typeFromName(name)
}

func extractNeighbors(entries []snmp.WalkEntry, nameByIndex map[int]string) []Neighbor {
	// Group table rows by index string: index -> column -> value.
	lldp := make(map[string]map[string]string)
	cdp := make(map[string]map[string]string)
	locPort := make(map[string]string) // lldp local port number -> interface name

	for _, e := range entries {
		oid := normalizeOID(e.OID)
		switch {
		case strings.HasPrefix(oid, lldpLocPortIDBase):
			locPort[strings.TrimPrefix(oid, lldpLocPortIDBase)] = entryString(e)
		case strings.HasPrefix(oid, lldpRemPrefix):
			collectRow(lldp, oid, lldpRemPrefix, e)
		case strings.HasPrefix(oid, cdpCachePrefix):
			collectRow(cdp, oid, cdpCachePrefix, e)
		}
	}

	neighbors := make([]Neighbor, 0, len(lldp)+len(cdp))
	neighbors = append(neighbors, buildLLDPNeighbors(lldp, locPort, nameByIndex)...)
	neighbors = append(neighbors, buildCDPNeighbors(cdp, nameByIndex)...)
	sortNeighbors(neighbors)

	return neighbors
}

func collectRow(rows map[string]map[string]string, oid, prefix string, e snmp.WalkEntry) {
	col, idx, ok := splitColIndex(oid, prefix)
	if !ok {
		return
	}
	row, exists := rows[idx]
	if !exists {
		row = make(map[string]string)
		rows[idx] = row
	}
	row[col] = entryString(e)
}

func buildLLDPNeighbors(
	rows map[string]map[string]string,
	locPort map[string]string,
	nameByIndex map[int]string,
) []Neighbor {
	out := make([]Neighbor, 0, len(rows))
	for index, row := range rows {
		remote := firstNonEmpty(row[lldpColSysName], row[lldpColChassisID])
		if remote == "" {
			continue
		}
		out = append(out, Neighbor{
			LocalInterface:  lldpLocalInterface(index, locPort, nameByIndex),
			RemoteDevice:    remote,
			RemoteInterface: firstNonEmpty(row[lldpColPortID], row[lldpColPortDesc]),
			Protocol:        protocolLLDP,
		})
	}
	return out
}

func buildCDPNeighbors(rows map[string]map[string]string, nameByIndex map[int]string) []Neighbor {
	out := make([]Neighbor, 0, len(rows))
	for index, row := range rows {
		remote := row[cdpColDeviceID]
		if remote == "" {
			continue
		}
		out = append(out, Neighbor{
			LocalInterface:  cdpLocalInterface(index, nameByIndex),
			RemoteDevice:    remote,
			RemoteInterface: row[cdpColDevicePort],
			Protocol:        protocolCDP,
		})
	}
	return out
}

// lldpLocalInterface resolves a neighbor's local interface. The lldpRemTable
// index is timeMark.localPortNum.remIndex; localPortNum resolves via
// lldpLocPortTable, falling back to the IF-MIB name at the same index.
func lldpLocalInterface(index string, locPort map[string]string, nameByIndex map[int]string) string {
	portNum := lldpLocalPortNum(index)
	if portNum == "" {
		return ""
	}
	if name, ok := locPort[portNum]; ok && name != "" {
		return name
	}
	if n, err := strconv.Atoi(portNum); err == nil {
		if name, ok := nameByIndex[n]; ok {
			return name
		}
	}
	return ""
}

// cdpLocalInterface resolves a CDP neighbor's local interface. The cdpCacheTable
// index is ifIndex.deviceIndex, so the first component is the local ifIndex.
func cdpLocalInterface(index string, nameByIndex map[int]string) string {
	parts := strings.Split(index, ".")
	if len(parts) == 0 {
		return ""
	}
	if n, err := strconv.Atoi(parts[0]); err == nil {
		if name, ok := nameByIndex[n]; ok {
			return name
		}
	}
	return ""
}

// lldpLocalPortNum extracts localPortNum from an lldpRemTable index. The index
// is timeMark.localPortNum.remIndex (or the abbreviated localPortNum.remIndex),
// so localPortNum is the component before the trailing remIndex.
func lldpLocalPortNum(index string) string {
	parts := strings.Split(index, ".")
	if len(parts) == 0 {
		return ""
	}
	if len(parts) == 1 {
		return parts[0]
	}
	return parts[len(parts)-2]
}

func sortNeighbors(n []Neighbor) {
	sort.Slice(n, func(i, j int) bool {
		a, b := n[i], n[j]
		switch {
		case a.LocalInterface != b.LocalInterface:
			return a.LocalInterface < b.LocalInterface
		case a.Protocol != b.Protocol:
			return a.Protocol < b.Protocol
		case a.RemoteDevice != b.RemoteDevice:
			return a.RemoteDevice < b.RemoteDevice
		default:
			return a.RemoteInterface < b.RemoteInterface
		}
	})
}

// typeFromName infers an interface type and physical class from its name, used
// only when the walk omits ifType.
func typeFromName(name string) (string, bool) {
	n := strings.ToLower(name)
	switch {
	case strings.Contains(n, "loopback"):
		return "loopback", false
	case strings.Contains(n, "null"):
		return "null", false
	case strings.Contains(n, "tunnel"):
		return "tunnel", false
	case strings.Contains(n, "port-channel"), strings.HasPrefix(n, "po"):
		return "port-channel", false
	case strings.Contains(n, "vlan"):
		return "vlan", false
	case strings.Contains(n, "ethernet"):
		return "ethernet", true
	case strings.Contains(n, "serial"):
		return "serial", false
	case strings.Contains(n, "async"):
		return "async", false
	default:
		return "unknown", false
	}
}

func normalizeOID(oid string) string {
	return strings.TrimPrefix(strings.TrimSpace(oid), ".")
}

// splitColIndex splits "<prefix><col>.<index>" into col and index. Both must be
// non-empty.
func splitColIndex(oid, prefix string) (string, string, bool) {
	if !strings.HasPrefix(oid, prefix) {
		return "", "", false
	}
	rest := oid[len(prefix):]
	dot := strings.IndexByte(rest, '.')
	if dot <= 0 || dot == len(rest)-1 {
		return "", "", false
	}
	return rest[:dot], rest[dot+1:], true
}

func lookupString(byOID map[string]snmp.WalkEntry, oid string) string {
	if e, ok := byOID[oid]; ok {
		return entryString(e)
	}
	return ""
}

func entryString(e snmp.WalkEntry) string {
	if s, ok := e.Value.(string); ok {
		return strings.TrimSpace(s)
	}
	return ""
}

// entryInt returns an SNMP Integer value (ifType, admin/oper status).
func entryInt(e snmp.WalkEntry) (int, bool) {
	v, ok := e.Value.(int)
	return v, ok
}

// entryGauge returns an SNMP Gauge32/Counter32 value (ifSpeed, ifHighSpeed) as
// int64. Both are 32-bit on the wire, so widening to int64 is lossless; the
// guard keeps gosec's overflow analysis satisfied where uint is 64-bit.
func entryGauge(e snmp.WalkEntry) (int64, bool) {
	var u uint64
	switch v := e.Value.(type) {
	case uint:
		u = uint64(v)
	case uint32:
		u = uint64(v)
	default:
		return 0, false
	}
	if u > math.MaxInt64 {
		return 0, false
	}
	return int64(u), true
}

// formatMAC normalizes an ifPhysAddress octet string into "AA:BB:CC:DD:EE:FF".
// A non-hex value (e.g. an empty logical-interface address) yields "".
func formatMAC(raw string) string {
	s := strings.ToLower(strings.TrimSpace(raw))
	s = strings.TrimPrefix(s, "0x")
	s = strings.ReplaceAll(s, " ", "")
	s = strings.ReplaceAll(s, ":", "")
	if s == "" || len(s)%2 != 0 || !isHexString(s) {
		return ""
	}

	var b strings.Builder
	for i := 0; i < len(s); i += 2 {
		if i > 0 {
			b.WriteByte(':')
		}
		b.WriteString(strings.ToUpper(s[i : i+2]))
	}
	return b.String()
}

func isHexString(s string) bool {
	for _, c := range s {
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') && (c < 'A' || c > 'F') {
			return false
		}
	}
	return true
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}
