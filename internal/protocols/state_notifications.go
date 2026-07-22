package protocols

import (
	"fmt"
	"log/slog"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gosnmp/gosnmp"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
	"github.com/MustardSeedNetworks/niac-go/internal/protocols/snmp"
	"github.com/MustardSeedNetworks/niac-go/internal/safeconv"
)

const (
	snmpCentisecond          = 10 * time.Millisecond
	notificationWriteTimeout = time.Second
)

type datagramSender interface {
	Send(address string, payload []byte) error
}

type udpDatagramSender struct{}

func (udpDatagramSender) Send(address string, payload []byte) error {
	remote, err := net.ResolveUDPAddr("udp", address)
	if err != nil {
		return fmt.Errorf("resolve UDP receiver: %w", err)
	}
	connection, err := net.DialUDP("udp", nil, remote)
	if err != nil {
		return fmt.Errorf("open UDP receiver: %w", err)
	}
	defer connection.Close()
	if err = connection.SetWriteDeadline(time.Now().Add(notificationWriteTimeout)); err != nil {
		return fmt.Errorf("set UDP write deadline: %w", err)
	}
	if _, err = connection.Write(payload); err != nil {
		return fmt.Errorf("send UDP datagram: %w", err)
	}
	return nil
}

type stateNotificationRegistration struct {
	device         *config.Device
	store          *devicestate.Store
	interfaceIndex func(string) (int, bool)
	cursor         uint64
}

type stateNotificationManager struct {
	mu            sync.Mutex
	wake          chan struct{}
	registrations map[*config.Device]*stateNotificationRegistration
	sender        datagramSender
	started       time.Time
}

func newStateNotificationManager() *stateNotificationManager {
	return &stateNotificationManager{
		wake: make(chan struct{}, 1), registrations: make(map[*config.Device]*stateNotificationRegistration),
		sender: udpDatagramSender{}, started: time.Now(),
	}
}

func (m *stateNotificationManager) Register(
	device *config.Device,
	store *devicestate.Store,
	interfaceIndex func(string) (int, bool),
) {
	if !notificationsEnabled(device) || store == nil {
		return
	}
	store.SetChangeSignal(m.wake)
	m.mu.Lock()
	m.registrations[device] = &stateNotificationRegistration{
		device: device, store: store, interfaceIndex: interfaceIndex,
	}
	m.mu.Unlock()
}

func notificationsEnabled(device *config.Device) bool {
	return device != nil && ((device.SyslogConfig != nil && device.SyslogConfig.Enabled) ||
		(device.SNMPConfig.Traps != nil && device.SNMPConfig.Traps.Enabled))
}

func (m *stateNotificationManager) Reset() {
	m.mu.Lock()
	for _, registration := range m.registrations {
		registration.store.SetChangeSignal(nil)
	}
	m.registrations = make(map[*config.Device]*stateNotificationRegistration)
	m.mu.Unlock()
}

func (m *stateNotificationManager) Run(stop <-chan struct{}) {
	m.sendColdStarts()
	m.dispatchPending()
	for {
		select {
		case <-m.wake:
			m.dispatchPending()
		case <-stop:
			return
		}
	}
}

func (m *stateNotificationManager) dispatchPending() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, registration := range m.registrations {
		events, complete := registration.store.EventsAfter(registration.cursor)
		if !complete {
			slog.Warn("device event history gap", "device", registration.device.Name,
				"cursor", registration.cursor, "oldestVersion", events[0].Version)
		}
		for _, event := range events {
			m.sendEvent(registration.device, event)
			registration.cursor = event.Version
		}
	}
}

func (m *stateNotificationManager) sendColdStarts() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, registration := range m.registrations {
		traps := registration.device.SNMPConfig.Traps
		if traps == nil || traps.ColdStart == nil || !traps.ColdStart.Enabled || !traps.ColdStart.OnStartup {
			continue
		}
		m.sendTrap(registration.device, snmp.OIDColdStart, nil, registration.store.Snapshot().Version)
	}
}

func (m *stateNotificationManager) sendEvent(device *config.Device, event devicestate.Event) {
	if device.SyslogConfig != nil && device.SyslogConfig.Enabled {
		payload := []byte(formatSyslog(m.hostname(device), event))
		for _, receiver := range device.SyslogConfig.Receivers {
			m.send(receiver, payload)
		}
	}
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

func (m *stateNotificationManager) sendTrap(
	device *config.Device,
	oid string,
	variables []gosnmp.SnmpPDU,
	version uint64,
) {
	traps := device.SNMPConfig.Traps
	if traps == nil || !traps.Enabled {
		return
	}
	community := traps.Community
	if community == "" {
		community = config.DefaultSNMPCommunity
	}
	uptime := safeconv.Uint32FromInt64(int64(time.Since(m.started) / snmpCentisecond))
	packet := &gosnmp.SnmpPacket{
		Version: gosnmp.Version2c, Community: community, PDUType: gosnmp.SNMPv2Trap,
		RequestID: safeconv.Uint32FromUint64(version), Variables: []gosnmp.SnmpPDU{
			{Name: ".1.3.6.1.2.1.1.3.0", Type: gosnmp.TimeTicks, Value: uptime},
			{Name: ".1.3.6.1.6.3.1.1.4.1.0", Type: gosnmp.ObjectIdentifier, Value: oid},
		},
	}
	packet.Variables = append(packet.Variables, variables...)
	packet.Variables = append(packet.Variables, m.trapIdentityVariables(device)...)
	payload, err := packet.MarshalMsg()
	if err != nil {
		slog.Warn("marshal SNMP notification", "device", device.Name, "error", err)
		return
	}
	for _, receiver := range traps.Receivers {
		m.send(normalizeReceiver(receiver, snmp.DefaultSNMPTrapPort), payload)
	}
}

func (m *stateNotificationManager) trapIdentityVariables(device *config.Device) []gosnmp.SnmpPDU {
	registration := m.registrations[device]
	if registration == nil {
		return nil
	}
	snapshot := registration.store.Snapshot()
	variables := []gosnmp.SnmpPDU{{
		Name: ".1.3.6.1.2.1.1.5.0", Type: gosnmp.OctetString, Value: snapshot.Identity.Hostname,
	}}
	for _, iface := range snapshot.Network.Interfaces {
		if iface.Address.IsValid() && iface.Address.Addr().Is4() {
			return append(variables, gosnmp.SnmpPDU{
				Name: ".1.3.6.1.6.3.18.1.3.0", Type: gosnmp.IPAddress,
				Value: iface.Address.Addr().String(),
			})
		}
	}
	return variables
}

func (m *stateNotificationManager) send(receiver string, payload []byte) {
	if err := m.sender.Send(receiver, payload); err != nil {
		slog.Warn("send management notification", "receiver", receiver, "error", err)
	}
}

func (m *stateNotificationManager) hostname(device *config.Device) string {
	registration := m.registrations[device]
	if registration == nil {
		return device.Name
	}
	return registration.store.Snapshot().Identity.Hostname
}

func normalizeReceiver(receiver string, defaultPort int) string {
	if _, _, err := net.SplitHostPort(receiver); err == nil {
		return receiver
	}
	return net.JoinHostPort(receiver, strconv.Itoa(defaultPort))
}

func formatSyslog(hostname string, event devicestate.Event) string {
	timestamp := event.Timestamp.UTC().Format(time.RFC3339Nano)
	data := fmt.Sprintf(
		`[niac version="%d" kind="%s" target="%s"]`,
		event.Version, syslogEscape(string(event.Kind)), syslogEscape(event.Target),
	)
	return fmt.Sprintf(
		"<133>1 %s %s niac - CONFIG %s configuration state changed",
		timestamp, syslogHostname(hostname), data,
	)
}

func syslogHostname(hostname string) string {
	if len(hostname) == 0 || len(hostname) > 255 {
		return "-"
	}
	for index := range len(hostname) {
		if hostname[index] < 33 || hostname[index] > 126 {
			return "-"
		}
	}
	return hostname
}

func syslogEscape(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, `"`, `\"`)
	return strings.ReplaceAll(value, `]`, `\]`)
}
