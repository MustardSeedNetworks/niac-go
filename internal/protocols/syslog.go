package protocols

import (
	"fmt"
	"strconv"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
)

const (
	syslogWarning       = 132 // local0.warning
	syslogNotice        = 133 // local0.notice
	syslogInformational = 134 // local0.info
	syslogTimestamp     = "2006-01-02T15:04:05.999999Z07:00"
	syslogHostnameLimit = 255
)

func (m *stateNotificationManager) sendSyslog(device *config.Device, event devicestate.Event) {
	if device.SyslogConfig == nil || !device.SyslogConfig.Enabled {
		return
	}
	message := formatSyslog(m.hostname(device), event)
	if message == "" {
		return
	}
	payload := []byte(message)
	for _, receiver := range device.SyslogConfig.Receivers {
		m.send(device, receiver, syslogPort, payload)
	}
}

func formatSyslog(hostname string, event devicestate.Event) string {
	priority, messageID := syslogEvent(event)
	if messageID == "" {
		return ""
	}
	return fmt.Sprintf("<%d>1 %s %s niac - %s - version=%d kind=%s target=%s",
		priority, event.Timestamp.UTC().Format(syslogTimestamp), syslogHostname(hostname), messageID,
		event.Version, event.Kind, strconv.QuoteToASCII(event.Target))
}

func syslogEvent(event devicestate.Event) (int, string) {
	switch event.Kind {
	case devicestate.EventFaultUpdated, devicestate.EventDeviceFaultUpdated:
		return syslogWarning, "FAULT_UPDATED"
	case devicestate.EventFaultCleared, devicestate.EventDeviceFaultCleared:
		return syslogNotice, "FAULT_CLEARED"
	case devicestate.EventInterfaceUpdated:
		if event.Interface == nil || event.PreviousInterface == nil ||
			event.Interface.OperUp == event.PreviousInterface.OperUp {
			return 0, ""
		}
		if event.Interface.OperUp {
			return syslogNotice, "LINK_UP"
		}
		return syslogWarning, "LINK_DOWN"
	// A station coming and going is routine, which is why Cisco logs its
	// association and roam at informational (%DOT11-6-ASSOC, %DOT11-6-ROAMED).
	case devicestate.EventStationAssociated:
		return syslogInformational, "STATION_ASSOCIATED"
	case devicestate.EventStationRoamed:
		return syslogInformational, "STATION_ROAMED"
	case devicestate.EventNetworkInstalled, devicestate.EventIdentityUpdated,
		devicestate.EventStartupSaved, devicestate.EventStartupReloaded, devicestate.EventStartupErased,
		devicestate.EventAuthoredReset, devicestate.EventCheckpointSaved, devicestate.EventCheckpointRestored,
		devicestate.EventVLANUpdated, devicestate.EventRouterUpdated, devicestate.EventRouteUpdated,
		devicestate.EventDeviceRebooted, devicestate.EventSTPTopologyChanged:
		return 0, ""
	// No captured AP logs its own transmit power changing, so a message for it
	// would be invented; a poller sees the change in IEEE802dot11-MIB instead.
	case devicestate.EventRadioUpdated:
		return 0, ""
	default:
		return 0, ""
	}
}

func syslogHostname(hostname string) string {
	if len(hostname) == 0 || len(hostname) > syslogHostnameLimit {
		return "-"
	}
	for index := range len(hostname) {
		if hostname[index] < '!' || hostname[index] > '~' {
			return "-"
		}
	}
	return hostname
}
