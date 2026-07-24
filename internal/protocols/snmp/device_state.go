package snmp

import (
	"time"

	"github.com/gosnmp/gosnmp"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
)

// AgentOptions configures an SNMP agent backed by shared device state.
type AgentOptions struct {
	Community  string
	DebugLevel int
	Telemetry  *ProtocolTelemetry
}

// NewAgentWithState creates an agent that reads mutable values from state.
func NewAgentWithState(
	device *config.Device,
	state *devicestate.Store,
	options AgentOptions,
) *Agent {
	telemetry := options.Telemetry
	if telemetry == nil {
		telemetry = NewProtocolTelemetry()
	}
	agent := NewAgentWithCommunityAndTelemetry(
		device,
		options.Community,
		options.DebugLevel,
		telemetry,
	)
	agent.bindDeviceState(state)

	return agent
}

func (a *Agent) bindDeviceState(state *devicestate.Store) {
	a.deviceState = state
	identityValue := func() *OIDValue {
		return &OIDValue{
			Type:  gosnmp.OctetString,
			Value: state.Snapshot().Identity.Hostname,
		}
	}
	a.mib.SetDynamic(sysNameOID, identityValue)
	for _, oid := range []string{lldpLocSysName, cdpGlobalDeviceID} {
		if a.mib.Get(oid) != nil {
			a.mib.SetDynamic(oid, identityValue)
		}
	}
	a.refreshDeviceStateInterfaceMIBs()
	snapshot := state.Snapshot()
	a.refreshDeviceStateIPMIBs(snapshot)
	a.replaceTransportListeners(deviceStateIPv4Addresses(snapshot))
	a.stateMIBVersion.Store(snapshot.Version)
}

func (a *Agent) syncDeviceStateMIBs() {
	if a.deviceState == nil {
		return
	}
	snapshot := a.deviceState.Snapshot()
	a.protocolStats.AdvanceInterfaceFaults(
		time.Now(),
		snapshot.Faults,
		authoredInterfaceSpeeds(a.device),
	)
	if snapshot.Version == a.stateMIBVersion.Load() {
		return
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if snapshot.Version == a.stateMIBVersion.Load() {
		return
	}
	a.refreshDeviceStateInterfaceMIBs()
	a.refreshDeviceStateIPMIBs(snapshot)
	a.replaceTransportListeners(deviceStateIPv4Addresses(snapshot))
	a.stateMIBVersion.Store(snapshot.Version)
}

func deviceStateIPv4Addresses(snapshot devicestate.Snapshot) []string {
	result := make([]string, 0, len(snapshot.Network.Interfaces))
	for _, iface := range snapshot.Network.Interfaces {
		if iface.Address.IsValid() && iface.Address.Addr().Is4() {
			result = append(result, iface.Address.Addr().String())
		}
	}
	return result
}

func authoredInterfaceSpeeds(device *config.Device) map[string]int {
	result := make(map[string]int, len(device.Interfaces))
	for _, iface := range device.Interfaces {
		result[iface.Name] = iface.Speed
	}
	return result
}

func (a *Agent) refreshDeviceStateInterfaceMIBs() {
	if a.deviceState == nil {
		return
	}
	for _, iface := range a.deviceState.Snapshot().Network.Interfaces {
		index, ok := a.ifIndexForInterface(iface.Name)
		if !ok {
			continue
		}
		interfaceName := iface.Name
		a.mib.SetDynamic(ifAdminStatus+"."+index, func() *OIDValue {
			return &OIDValue{
				Type:  gosnmp.Integer,
				Value: a.deviceStateInterfaceStatus(interfaceName, true),
			}
		})
		a.mib.SetDynamic(ifOperStatus+"."+index, func() *OIDValue {
			return &OIDValue{
				Type:  gosnmp.Integer,
				Value: a.deviceStateInterfaceStatus(interfaceName, false),
			}
		})
		a.mib.SetDynamic(ifAlias+"."+index, func() *OIDValue {
			return &OIDValue{
				Type: gosnmp.OctetString, Value: a.deviceStateInterfaceDescription(interfaceName),
			}
		})
	}
}

func (a *Agent) deviceStateInterfaceDescription(name string) string {
	for _, iface := range a.deviceState.Snapshot().Network.Interfaces {
		if iface.Name == name {
			return iface.Description
		}
	}
	return ""
}

func (a *Agent) deviceStateInterfaceStatus(name string, admin bool) int {
	for _, iface := range a.deviceState.Snapshot().Network.Interfaces {
		if iface.Name == name && ((admin && iface.AdminUp) || (!admin && iface.OperUp)) {
			return interfaceStatusUp
		}
	}
	return interfaceStatusDown
}
