package snmp

import (
	"slices"
	"strconv"
	"strings"

	"github.com/gosnmp/gosnmp"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
)

// POWER-ETHERNET-MIB OID prefixes (RFC 3621). Every column a real PSE answers
// is registered; the two index columns are not, because they are
// not-accessible in the MIB and the agents in the walk corpus do not return
// them either.
const (
	// pethObjects (1.3.6.1.2.1.105.1).
	pethObjects = "1.3.6.1.2.1.105.1"

	// pethPsePortTable (1.3.6.1.2.1.105.1.1), indexed group.port.
	pethPsePortEntry                    = pethObjects + ".1.1"
	pethPsePortAdminEnable              = pethPsePortEntry + ".3"
	pethPsePortPowerPairsControlAbility = pethPsePortEntry + ".4"
	pethPsePortPowerPairs               = pethPsePortEntry + ".5"
	pethPsePortDetectionStatus          = pethPsePortEntry + ".6"
	pethPsePortPowerPriority            = pethPsePortEntry + ".7"
	pethPsePortMPSAbsentCounter         = pethPsePortEntry + ".8"
	pethPsePortType                     = pethPsePortEntry + ".9"
	pethPsePortPowerClassifications     = pethPsePortEntry + ".10"
	pethPsePortInvalidSignatureCounter  = pethPsePortEntry + ".11"
	pethPsePortPowerDeniedCounter       = pethPsePortEntry + ".12"
	pethPsePortOverLoadCounter          = pethPsePortEntry + ".13"
	pethPsePortShortCounter             = pethPsePortEntry + ".14"

	// pethMainPseTable (1.3.6.1.2.1.105.1.3.1), indexed group. The extra arc
	// against the port table above is the MIB's own asymmetry: RFC 3621 hangs
	// pethPsePortTable straight off pethObjects but puts pethMainPseTable under
	// a pethMainPseObjects node first.
	pethMainPseEntry            = pethObjects + ".3.1.1"
	pethMainPsePower            = pethMainPseEntry + ".2"
	pethMainPseOperStatus       = pethMainPseEntry + ".3"
	pethMainPseConsumptionPower = pethMainPseEntry + ".4"
	pethMainPseUsageThreshold   = pethMainPseEntry + ".5"

	// pethNotificationControlTable (1.3.6.1.2.1.105.1.4), indexed group.
	pethNotificationControlEnable = pethObjects + ".4.1.1.2"
)

// POWER-ETHERNET-MIB enumerations (RFC 3621).
const (
	pethPortDetectionSearching       = 2
	pethPortDetectionDeliveringPower = 3
	pethPortDetectionFault           = 4

	pethMainPseOperStatusOn = 1

	pethPowerPairsSignal = 1

	pethPriorityCritical = 1
	pethPriorityHigh     = 2
	pethPriorityLow      = 3

	// Power classes are reported as class0(1) through class4(5).
	pethPowerClass0 = 1
	pethPowerClass1 = 2
	pethPowerClass2 = 3
	pethPowerClass3 = 4
	pethPowerClass4 = 5

	// 802.3af/at class ceilings at the powered device, in tenths of a watt.
	class1CeilingTenthWatts = 38
	class2CeilingTenthWatts = 64
	class3CeilingTenthWatts = 129

	// pethPseGroupIndex is the single PSE group NIAC models: one chassis, one
	// power supply. A stacked switch would carry one group per member, which
	// only a capture of that stack can say truthfully.
	pethPseGroupIndex = "1"
)

// poePower is the per-port power picture, published as one immutable snapshot.
// The MIB serves it from dynamic OIDs, which run while the MIB's own lock is
// held, so they must not reach for a lock the agent takes in the other order.
type poePower struct {
	// drawTenthWatts maps an ifIndex to what the powered device behind it
	// advertises drawing. A port absent from the map has nothing attached.
	drawTenthWatts map[string]int
	// priority maps an ifIndex to the PD's advertised power priority.
	priority map[string]int
	// interfaceName maps an ifIndex back to the authored interface, which is
	// what a PoE-loss fault is keyed by.
	interfaceName map[string]string
}

// initializePoEMIB registers POWER-ETHERNET-MIB for a device authored as power
// sourcing equipment. Which ports draw power is not known yet -- that needs the
// fleet roster, and SynthesizePoEPower supplies it -- so the columns that
// depend on an attached device are dynamic from the start.
//
// A walk-backed device gets nothing here. Whether its capture carries a real
// PSE table is only known once the walk has loaded, and the walk also owns the
// interface indexes these rows are keyed by, so refreshWalkedPoEMIB does it
// there.
func (a *Agent) initializePoEMIB() {
	if a.device == nil || a.device.PoEConfig == nil || a.hasWalkContent() {
		return
	}

	a.registerPoEMIB()
}

// refreshWalkedPoEMIB synthesizes the table for a PoE switch whose capture did
// not include one. A capture that did carry POWER-ETHERNET-MIB keeps it
// untouched: a real PSE reports its own group count, port numbering and
// consumption, and overwriting that with a synthesized single-group table would
// both lie and register as an unclassified substitution in the replay-fidelity
// contract.
func (a *Agent) refreshWalkedPoEMIB(walkOwnsPoE bool) {
	if a.device == nil || a.device.PoEConfig == nil || walkOwnsPoE {
		return
	}

	a.registerPoEMIB()
}

func (a *Agent) registerPoEMIB() {
	for _, index := range a.poePortIndexes() {
		a.registerPoEPortEntry(index)
	}
	a.registerPoEMainEntry()
}

// walkOwnsPoE reports whether a parsed capture carries POWER-ETHERNET-MIB of
// its own. Any object under the group counts: the corpus has captures with only
// the notification-control table, and a PSE that answers part of the MIB is
// still the authority on all of it.
func walkOwnsPoE(entries []WalkEntry) bool {
	for _, entry := range entries {
		if strings.HasPrefix(strings.TrimPrefix(entry.OID, "."), pethObjects+".") {
			return true
		}
	}

	return false
}

// poePortIndexes returns the ifIndex of every port that can source power, in
// ifIndex order. It reads the interface table rather than the authored device
// because that table is the authority on both the index space and which ports
// exist: a pack switch declares its access ports only as trunk_ports, and a
// walk-backed switch gets its ports and their numbering from the capture.
//
// Only Ethernet ports are PSE ports -- a VLAN, loopback or radio interface has
// no pair to put voltage on.
func (a *Agent) poePortIndexes() []string {
	prefix := ifType + "."
	indexes := make([]int, 0, len(a.device.TrunkPorts)+len(a.device.Interfaces))
	for _, oid := range a.mib.AllOIDs() {
		if !strings.HasPrefix(oid, prefix) {
			continue
		}
		entry := a.mib.Get(oid)
		if entry == nil || oidValueString(entry) != strconv.Itoa(interfaceTypeEthernet) {
			continue
		}
		index, err := strconv.Atoi(strings.TrimPrefix(oid, prefix))
		if err != nil {
			continue
		}
		indexes = append(indexes, index)
	}
	slices.Sort(indexes)

	suffixes := make([]string, 0, len(indexes))
	for _, index := range indexes {
		suffixes = append(suffixes, strconv.Itoa(index))
	}

	return suffixes
}

func (a *Agent) registerPoEPortEntry(ifIndex string) {
	suffix := "." + pethPseGroupIndex + "." + ifIndex

	a.mib.Set(pethPsePortAdminEnable+suffix,
		&OIDValue{Type: gosnmp.Integer, Value: TruthValueTrue})
	// NIAC does not model switching power between the signal and spare pairs,
	// so the ability is reported absent, as both PoE captures in the corpus do.
	a.mib.Set(pethPsePortPowerPairsControlAbility+suffix,
		&OIDValue{Type: gosnmp.Integer, Value: TruthValueFalse})
	a.mib.Set(pethPsePortPowerPairs+suffix,
		&OIDValue{Type: gosnmp.Integer, Value: pethPowerPairsSignal})
	a.mib.Set(pethPsePortType+suffix, &OIDValue{Type: gosnmp.OctetString, Value: ""})

	for _, counter := range []string{
		pethPsePortMPSAbsentCounter,
		pethPsePortInvalidSignatureCounter,
		pethPsePortPowerDeniedCounter,
		pethPsePortOverLoadCounter,
		pethPsePortShortCounter,
	} {
		a.mib.Set(counter+suffix, &OIDValue{Type: gosnmp.Counter32, Value: uint32(0)})
	}

	a.mib.SetDynamic(pethPsePortDetectionStatus+suffix, func() *OIDValue {
		return &OIDValue{Type: gosnmp.Integer, Value: a.poeDetectionStatus(ifIndex)}
	})
	a.mib.SetDynamic(pethPsePortPowerPriority+suffix, func() *OIDValue {
		return &OIDValue{Type: gosnmp.Integer, Value: a.poePortPriority(ifIndex)}
	})
	a.mib.SetDynamic(pethPsePortPowerClassifications+suffix, func() *OIDValue {
		return &OIDValue{Type: gosnmp.Integer, Value: a.poePortClass(ifIndex)}
	})
}

func (a *Agent) registerPoEMainEntry() {
	suffix := "." + pethPseGroupIndex
	poe := a.device.PoEConfig

	a.mib.Set(pethMainPsePower+suffix, &OIDValue{
		Type: gosnmp.Gauge32, Value: safeUint32(int64(poe.BudgetWatts)),
	})
	a.mib.Set(pethMainPseOperStatus+suffix, &OIDValue{
		Type: gosnmp.Integer, Value: pethMainPseOperStatusOn,
	})
	a.mib.Set(pethMainPseUsageThreshold+suffix, &OIDValue{
		Type: gosnmp.Integer, Value: poe.UsageThreshold(),
	})
	a.mib.SetDynamic(pethMainPseConsumptionPower+suffix, func() *OIDValue {
		return &OIDValue{Type: gosnmp.Gauge32, Value: safeUint32(int64(a.poeConsumptionWatts()))}
	})

	// NIAC sends no pethPsePortOnOffNotification, so the control says so
	// rather than advertising a notification nothing emits.
	a.mib.Set(pethNotificationControlEnable+suffix, &OIDValue{
		Type: gosnmp.Integer, Value: TruthValueFalse,
	})
}

// PoEFaultObservable reports whether a PoE fault on name would be visible: the
// port must actually be a PSE port. Cutting power on a port that supplies none
// would arm a fault indistinguishable from link_down, which is a success message
// with nothing behind it.
func (a *Agent) PoEFaultObservable(name string) bool {
	if a.device == nil || a.device.PoEConfig == nil {
		return false
	}
	index, ok := a.ifIndexForInterface(name)
	if !ok {
		return false
	}

	return a.mib.Get(pethPsePortAdminEnable+"."+pethPseGroupIndex+"."+index) != nil
}

// SynthesizePoEPower publishes which of this device's ports have a powered
// device behind them. Like the rest of the peer topology it can only be built
// once the whole roster is loaded: a switch alone does not know what its
// neighbours draw.
func (a *Agent) SynthesizePoEPower(resolve PeerResolver) {
	if a == nil || a.device == nil || a.device.PoEConfig == nil || resolve == nil {
		return
	}

	power := &poePower{
		drawTenthWatts: make(map[string]int),
		priority:       make(map[string]int),
		interfaceName:  make(map[string]string),
	}
	for _, index := range a.poePortIndexes() {
		power.interfaceName[index] = a.interfaceNameForIfIndex(index)
	}
	for _, trunk := range a.device.TrunkPorts {
		if trunk.RemoteDevice == "" || trunk.Interface == "" {
			continue
		}
		peer, ok := resolve(trunk.RemoteDevice, trunk.RemoteInterface)
		if !ok || peer.PoEDrawTenthWatts <= 0 {
			continue
		}
		index, ok := a.ifIndexForInterface(trunk.Interface)
		if !ok {
			continue
		}
		power.drawTenthWatts[index] = peer.PoEDrawTenthWatts
		power.priority[index] = poePriorityValue(peer.PoEPriority)
	}
	a.poe.Store(power)
}

// interfaceNameForIfIndex reads the name back out of the interface table, which
// is what a fault is keyed by. It cannot come from the authored device: a pack
// switch declares its access ports only as trunk_ports, so its ports have a
// name in ifDescr and no entry in Interfaces at all.
func (a *Agent) interfaceNameForIfIndex(ifIndex string) string {
	entry := a.mib.Get(ifDescr + "." + ifIndex)
	if entry == nil {
		return ""
	}

	return oidValueString(entry)
}

func poePriorityValue(priority string) int {
	switch priority {
	case "critical":
		return pethPriorityCritical
	case "high":
		return pethPriorityHigh
	default:
		return pethPriorityLow
	}
}

// poeDetectionStatus is what a tester reads to tell a powered port from an
// empty one, and a failed one from both. A PoE-loss fault reports fault(4)
// rather than searching(2): the difference is exactly what distinguishes a
// switch that stopped supplying power from a port with nothing plugged in.
func (a *Agent) poeDetectionStatus(ifIndex string) int {
	power := a.poe.Load()
	if power == nil {
		return pethPortDetectionSearching
	}
	if a.poeFaultActive(power.interfaceName[ifIndex]) {
		return pethPortDetectionFault
	}
	if power.drawTenthWatts[ifIndex] > 0 {
		return pethPortDetectionDeliveringPower
	}

	return pethPortDetectionSearching
}

func (a *Agent) poePortPriority(ifIndex string) int {
	power := a.poe.Load()
	if power == nil || power.priority[ifIndex] == 0 {
		return pethPriorityLow
	}

	return power.priority[ifIndex]
}

// poePortClass reports the 802.3af class the attached device signalled. A port
// with nothing attached, or one whose power was cut, has no class to report and
// answers class0.
func (a *Agent) poePortClass(ifIndex string) int {
	power := a.poe.Load()
	if power == nil || a.poeFaultActive(power.interfaceName[ifIndex]) {
		return pethPowerClass0
	}

	switch draw := power.drawTenthWatts[ifIndex]; {
	case draw <= 0:
		return pethPowerClass0
	case draw <= class1CeilingTenthWatts:
		return pethPowerClass1
	case draw <= class2CeilingTenthWatts:
		return pethPowerClass2
	case draw <= class3CeilingTenthWatts:
		return pethPowerClass3
	default:
		return pethPowerClass4
	}
}

// poeConsumptionWatts is the PSE's draw against its budget: the sum of every
// port still delivering power, so cutting power to a phone is visible in the
// closet total and not only on the port.
func (a *Agent) poeConsumptionWatts() int {
	power := a.poe.Load()
	if power == nil {
		return 0
	}

	total := 0
	for ifIndex, draw := range power.drawTenthWatts {
		if a.poeFaultActive(power.interfaceName[ifIndex]) {
			continue
		}
		total += draw
	}

	return total / config.PoETenthWattsPerWatt
}

func (a *Agent) poeFaultActive(interfaceName string) bool {
	if a.deviceState == nil || interfaceName == "" {
		return false
	}
	for _, fault := range a.deviceState.Snapshot().Faults {
		if fault.Interface == interfaceName &&
			fault.Type == devicestate.FaultPoELoss && fault.Value > 0 {
			return true
		}
	}

	return false
}
