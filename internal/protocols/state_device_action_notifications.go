package protocols

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/gosnmp/gosnmp"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
	"github.com/MustardSeedNetworks/niac-go/internal/protocols/snmp"
	"github.com/MustardSeedNetworks/niac-go/internal/safeconv"
)

func (m *stateNotificationManager) sendEvent(device *config.Device, event devicestate.Event) {
	m.sendSyslog(device, event)
	m.sendDeviceActionNotification(device, event)
	if event.Kind != devicestate.EventInterfaceUpdated || event.Interface == nil || event.PreviousInterface == nil {
		return
	}
	if event.Interface.OperUp == event.PreviousInterface.OperUp {
		return
	}
	traps := device.SNMPConfig.Traps
	if traps == nil || traps.LinkState == nil || !traps.LinkState.Enabled {
		return
	}
	up := event.Interface.OperUp
	if (up && !traps.LinkState.LinkUp) || (!up && !traps.LinkState.LinkDown) {
		return
	}
	oid := snmp.OIDLinkDown
	operStatus := snmp.IfStatusDown
	if up {
		oid = snmp.OIDLinkUp
		operStatus = snmp.IfStatusUp
	}
	adminStatus := snmp.IfStatusDown
	if event.Interface.AdminUp {
		adminStatus = snmp.IfStatusUp
	}
	index := event.InterfaceIndex
	if registration := m.registrations[device]; registration != nil && registration.interfaceIndex != nil {
		resolved, found := registration.interfaceIndex(event.Interface.Name)
		if !found {
			slog.Warn("skip link notification without IF-MIB index", "device", device.Name,
				"interface", event.Interface.Name)
			return
		}
		index = resolved
	}
	variables := []gosnmp.SnmpPDU{
		{Name: fmt.Sprintf(".1.3.6.1.2.1.2.2.1.1.%d", index), Type: gosnmp.Integer, Value: index},
		{Name: fmt.Sprintf(".1.3.6.1.2.1.2.2.1.7.%d", index), Type: gosnmp.Integer, Value: adminStatus},
		{Name: fmt.Sprintf(".1.3.6.1.2.1.2.2.1.8.%d", index), Type: gosnmp.Integer, Value: operStatus},
		{Name: fmt.Sprintf(".1.3.6.1.2.1.2.2.1.2.%d", index), Type: gosnmp.OctetString, Value: event.Interface.Name},
	}
	m.sendTrap(device, oid, variables, event.Version)
}

func (m *stateNotificationManager) sendDeviceActionNotification(device *config.Device, event devicestate.Event) {
	if event.Kind == devicestate.EventDeviceRebooted {
		traps := device.SNMPConfig.Traps
		if traps != nil && traps.ColdStart != nil && traps.ColdStart.Enabled {
			m.sendTrap(device, snmp.OIDColdStart, nil, event.Version)
		}
		return
	}
	if event.Kind != devicestate.EventSTPTopologyChanged || m.stack == nil || m.stack.stpHandler == nil ||
		device.STPConfig == nil || !device.STPConfig.Enabled {
		return
	}
	if err := m.stack.stpHandler.SendTopologyChange(device); err != nil {
		slog.Warn("send topology change notification", "device", device.Name, "error", err)
	}
}

func (m *stateNotificationManager) notificationUptime(device *config.Device) uint32 {
	started := m.started
	if registration := m.registrations[device]; registration != nil {
		if registration.uptime != nil {
			return registration.uptime()
		}
		if rebooted := registration.store.DeviceTelemetry().RebootedAt; !rebooted.IsZero() {
			started = rebooted
		}
	}
	return safeconv.Uint32FromInt64(int64(time.Since(started) / snmpCentisecond))
}
