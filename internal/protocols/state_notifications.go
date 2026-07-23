package protocols

import (
	"fmt"
	"log/slog"
	"net"
	"net/netip"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/layers"
	"github.com/gosnmp/gosnmp"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
	"github.com/MustardSeedNetworks/niac-go/internal/protocols/snmp"
	"github.com/MustardSeedNetworks/niac-go/internal/safeconv"
)

const (
	snmpCentisecond             = 10 * time.Millisecond
	syslogPort                  = 514
	maxPendingNeighborDatagrams = 64
	notificationNeighborRetries = 4
	notificationNeighborRetry   = 500 * time.Millisecond
	notificationNeighborTTL     = 5 * time.Minute
)

type datagramSender interface {
	Send(device *config.Device, vlan int, address string, sourcePort uint16, payload []byte) error
}

type notificationNeighborKey struct {
	vlan    int
	address netip.Addr
}

type notificationNeighbor struct {
	mac       net.HardwareAddr
	expiresAt time.Time
}

type pendingNotification struct {
	device                      *config.Device
	vlan                        int
	source, destination         netip.Addr
	sourcePort, destinationPort uint16
	payload                     []byte
}

type notificationNeighborResolution struct {
	notifications []pendingNotification
	attempts      int
	timer         *time.Timer
}

type stackDatagramSender struct {
	stack     *Stack
	mu        sync.Mutex
	neighbors map[notificationNeighborKey]notificationNeighbor
	pending   map[notificationNeighborKey]*notificationNeighborResolution
	now       func() time.Time
}

func newStackDatagramSender(stack *Stack) *stackDatagramSender {
	return &stackDatagramSender{
		stack: stack, neighbors: make(map[notificationNeighborKey]notificationNeighbor),
		pending: make(map[notificationNeighborKey]*notificationNeighborResolution),
		now:     time.Now,
	}
}

type serializableNetworkLayer interface {
	gopacket.NetworkLayer
	gopacket.SerializableLayer
}

func (s *stackDatagramSender) Send(
	device *config.Device,
	vlan int,
	address string,
	sourcePort uint16,
	payload []byte,
) error {
	vlan = s.notificationWireVLAN(vlan)
	destination, port, err := parseNotificationReceiver(address)
	if err != nil {
		return err
	}
	source, target := s.notificationRoute(device, destination)
	if !source.IsValid() || len(device.MACAddress) != SizeOfMac {
		return fmt.Errorf("device %q has no active notification source for %s", device.Name, destination)
	}
	destinationMAC, err := s.notificationDestinationMAC(device, vlan, destination, target)
	if err != nil {
		return err
	}
	if destinationMAC == nil {
		return s.resolveNeighbor(pendingNotification{
			device: device, vlan: vlan, source: source, destination: destination,
			sourcePort: sourcePort, destinationPort: port, payload: append([]byte(nil), payload...),
		}, target)
	}
	return s.emitNotification(device, vlan, source, destination, sourcePort, port, destinationMAC, payload)
}

func (s *stackDatagramSender) notificationWireVLAN(vlan int) int {
	if s.stack.fabric != nil && !s.stack.fabric.binding.WireTagged {
		return config.UntaggedTag
	}
	if vlan < config.UntaggedTag {
		return config.UntaggedTag
	}
	return vlan
}

func parseNotificationReceiver(address string) (netip.Addr, uint16, error) {
	host, portText, err := net.SplitHostPort(address)
	if err != nil {
		return netip.Addr{}, 0, fmt.Errorf("parse UDP receiver: %w", err)
	}
	destination, err := netip.ParseAddr(host)
	if err != nil {
		return netip.Addr{}, 0, fmt.Errorf("parse UDP receiver address: %w", err)
	}
	port, err := strconv.ParseUint(portText, 10, 16)
	if err != nil || port == 0 {
		return netip.Addr{}, 0, fmt.Errorf("parse UDP receiver port %q", portText)
	}
	return destination.Unmap(), uint16(port), nil
}

func (s *stackDatagramSender) notificationRoute(
	device *config.Device,
	destination netip.Addr,
) (netip.Addr, netip.Addr) {
	state := s.stack.deviceStates[device]
	if state == nil {
		return netip.Addr{}, netip.Addr{}
	}
	snapshot := state.Snapshot()
	source, bestBits := notificationConnectedSource(snapshot.Network.Interfaces, destination)
	route := notificationStaticRoute(snapshot.Network.Routes, destination, bestBits)
	if route.NextHop.IsValid() {
		routeSource := notificationRouteSource(snapshot.Network.Interfaces, route, destination)
		if routeSource.IsValid() {
			return routeSource, route.NextHop
		}
		return netip.Addr{}, netip.Addr{}
	}
	if source.IsValid() {
		return source, destination
	}
	if !hasNotificationSubnet(snapshot.Network.Interfaces, destination) {
		for _, iface := range snapshot.Network.Interfaces {
			if iface.AdminUp && iface.OperUp && iface.Address.IsValid() &&
				iface.Address.Addr().Is4() == destination.Is4() {
				return iface.Address.Addr(), destination
			}
		}
	}
	return netip.Addr{}, netip.Addr{}
}

func notificationConnectedSource(
	interfaces []devicestate.Interface,
	destination netip.Addr,
) (netip.Addr, int) {
	bestBits := -1
	var source netip.Addr
	for _, iface := range interfaces {
		if iface.AdminUp && iface.OperUp && iface.Address.IsValid() &&
			iface.Address.Addr().Is4() == destination.Is4() && iface.Address.Contains(destination) &&
			iface.Address.Bits() > bestBits {
			bestBits, source = iface.Address.Bits(), iface.Address.Addr()
		}
	}
	return source, bestBits
}

func notificationStaticRoute(
	routes []devicestate.Route,
	destination netip.Addr,
	bestBits int,
) devicestate.Route {
	var route devicestate.Route
	for _, candidate := range routes {
		if candidate.NextHop.IsValid() && candidate.Destination.Contains(destination) &&
			candidate.Destination.Bits() > bestBits {
			bestBits, route = candidate.Destination.Bits(), candidate
		}
	}
	return route
}

func notificationRouteSource(
	interfaces []devicestate.Interface,
	route devicestate.Route,
	destination netip.Addr,
) netip.Addr {
	for _, iface := range interfaces {
		if iface.Name == route.Via && iface.AdminUp && iface.OperUp && iface.Address.IsValid() &&
			iface.Address.Addr().Is4() == destination.Is4() {
			return iface.Address.Addr()
		}
	}
	return netip.Addr{}
}

func hasNotificationSubnet(interfaces []devicestate.Interface, destination netip.Addr) bool {
	for _, iface := range interfaces {
		if iface.AdminUp && iface.OperUp && iface.Address.IsValid() &&
			iface.Address.Addr().Is4() == destination.Is4() &&
			iface.Address.Bits() < iface.Address.Addr().BitLen() {
			return true
		}
	}
	return false
}

func (s *stackDatagramSender) notificationDestinationMAC(
	device *config.Device,
	vlan int,
	destination netip.Addr,
	target netip.Addr,
) (net.HardwareAddr, error) {
	if !target.IsValid() {
		return nil, fmt.Errorf("device %q has no route to notification receiver %s", device.Name, destination)
	}
	if resolved := s.notificationTargetDevice(vlan, target); resolved != nil &&
		len(resolved.MACAddress) == SizeOfMac {
		return resolved.MACAddress, nil
	}
	key := newNotificationNeighborKey(vlan, target)
	s.mu.Lock()
	neighbor, found := s.neighbors[key]
	if found && !s.now().Before(neighbor.expiresAt) {
		delete(s.neighbors, key)
		found = false
	}
	s.mu.Unlock()
	if !found {
		return nil, nil
	}
	return append(net.HardwareAddr(nil), neighbor.mac...), nil
}

func (s *stackDatagramSender) resolveNeighbor(notification pendingNotification, target netip.Addr) error {
	key := newNotificationNeighborKey(notification.vlan, target)
	s.mu.Lock()
	resolution := s.pending[key]
	if resolution != nil && len(resolution.notifications) >= maxPendingNeighborDatagrams {
		s.mu.Unlock()
		return fmt.Errorf("notification neighbor queue for %s is full", target)
	}
	if resolution != nil {
		resolution.notifications = append(resolution.notifications, notification)
		s.mu.Unlock()
		return nil
	}
	s.pending[key] = &notificationNeighborResolution{notifications: []pendingNotification{notification}}
	s.mu.Unlock()
	s.probeNeighbor(key)
	return nil
}

func (s *stackDatagramSender) probeNeighbor(key notificationNeighborKey) {
	s.mu.Lock()
	resolution := s.pending[key]
	if resolution == nil {
		s.mu.Unlock()
		return
	}
	if resolution.attempts >= notificationNeighborRetries {
		delete(s.pending, key)
		s.mu.Unlock()
		slog.Warn("notification neighbor resolution timed out", "target", key.address, "vlan", key.vlan)
		return
	}
	notification := resolution.notifications[0]
	resolution.attempts++
	s.mu.Unlock()

	frame, err := s.neighborProbeFrame(notification, key.address)
	if err != nil {
		s.mu.Lock()
		delete(s.pending, key)
		s.mu.Unlock()
		slog.Warn("build notification neighbor probe", "target", key.address, "error", err)
		return
	}
	s.queueFrame(notification.device, notification.vlan, frame)

	s.mu.Lock()
	if current := s.pending[key]; current == resolution {
		current.timer = time.AfterFunc(notificationNeighborRetry, func() { s.probeNeighbor(key) })
	}
	s.mu.Unlock()
}

func (s *stackDatagramSender) neighborProbeFrame(
	notification pendingNotification,
	target netip.Addr,
) ([]byte, error) {
	if target.Is4() {
		zeroMAC := net.HardwareAddr{0, 0, 0, 0, 0, 0}
		return serializeARPPacket(
			&layers.Ethernet{
				SrcMAC:       notification.device.MACAddress,
				DstMAC:       net.HardwareAddr{0xff, 0xff, 0xff, 0xff, 0xff, 0xff},
				EthernetType: layers.EthernetTypeARP,
			},
			buildARPLayer(
				layers.ARPRequest, notification.device.MACAddress, net.IP(notification.source.AsSlice()),
				zeroMAC, net.IP(target.AsSlice()),
			),
		)
	}
	return serializeNeighborSolicitation(notification.device.MACAddress, notification.source, target)
}

func serializeNeighborSolicitation(sourceMAC net.HardwareAddr, source, target netip.Addr) ([]byte, error) {
	targetBytes := target.As16()
	destinationBytes := [16]byte{0xff, 0x02}
	destinationBytes[11], destinationBytes[12] = 0x01, 0xff
	copy(destinationBytes[13:], targetBytes[13:])
	destination := netip.AddrFrom16(destinationBytes)
	ipv6 := &layers.IPv6{
		Version: icmpv6IPv6Version, HopLimit: icmpv6NDPHopLimit, NextHeader: layers.IPProtocolICMPv6,
		SrcIP: source.AsSlice(), DstIP: destination.AsSlice(),
	}
	icmp := &layers.ICMPv6{TypeCode: layers.CreateICMPv6TypeCode(ICMPv6TypeNeighborSolicitation, 0)}
	if err := icmp.SetNetworkLayerForChecksum(ipv6); err != nil {
		return nil, fmt.Errorf("set neighbor solicitation checksum: %w", err)
	}
	payload := make([]byte, icmpv6NAPayloadSize-icmpv6Uint32Size)
	copy(payload[4:20], targetBytes[:])
	payload[20], payload[21] = ICMPv6OptSourceLinkAddr, 1
	copy(payload[22:], sourceMAC)
	buffer := gopacket.NewSerializeBuffer()
	err := gopacket.SerializeLayers(buffer, gopacket.SerializeOptions{FixLengths: true, ComputeChecksums: true},
		&layers.Ethernet{
			SrcMAC:       sourceMAC,
			DstMAC:       net.HardwareAddr{0x33, 0x33, 0xff, targetBytes[13], targetBytes[14], targetBytes[15]},
			EthernetType: layers.EthernetTypeIPv6,
		},
		ipv6, icmp, gopacket.Payload(payload))
	if err != nil {
		return nil, fmt.Errorf("serialize neighbor solicitation: %w", err)
	}
	return buffer.Bytes(), nil
}

func (s *stackDatagramSender) observeNeighbor(vlan int, address netip.Addr, mac net.HardwareAddr) {
	if len(mac) != SizeOfMac {
		return
	}
	key := newNotificationNeighborKey(vlan, address)
	s.mu.Lock()
	resolution := s.pending[key]
	if resolution != nil && resolution.timer != nil {
		resolution.timer.Stop()
	}
	s.neighbors[key] = notificationNeighbor{
		mac:       append(net.HardwareAddr(nil), mac...),
		expiresAt: s.now().Add(notificationNeighborTTL),
	}
	delete(s.pending, key)
	s.mu.Unlock()
	if resolution == nil {
		return
	}
	for _, notification := range resolution.notifications {
		if err := s.emitNotification(
			notification.device, notification.vlan, notification.source, notification.destination,
			notification.sourcePort, notification.destinationPort, mac, notification.payload,
		); err != nil {
			slog.Warn("send resolved management notification", "receiver", notification.destination, "error", err)
		}
	}
}

func newNotificationNeighborKey(vlan int, address netip.Addr) notificationNeighborKey {
	if vlan < config.UntaggedTag {
		vlan = config.UntaggedTag
	}
	return notificationNeighborKey{vlan: vlan, address: address}
}

func (s *stackDatagramSender) reset() {
	s.mu.Lock()
	for _, resolution := range s.pending {
		if resolution.timer != nil {
			resolution.timer.Stop()
		}
	}
	clear(s.neighbors)
	clear(s.pending)
	s.mu.Unlock()
}

func (s *stackDatagramSender) notificationTargetDevice(vlan int, target netip.Addr) *config.Device {
	if target.Is4() {
		if devices := s.stack.devicesForStateIPv4(vlan, net.IP(target.AsSlice())); len(devices) > 0 {
			return devices[0]
		}
		return nil
	}
	if devices := s.stack.devicesFor(vlan).GetByIP(net.IP(target.AsSlice())); len(devices) > 0 {
		return devices[0]
	}
	return nil
}

func (s *stackDatagramSender) emitNotification(
	device *config.Device,
	vlan int,
	source, destination netip.Addr,
	sourcePort, destinationPort uint16,
	destinationMAC net.HardwareAddr,
	payload []byte,
) error {
	buffer := gopacket.NewSerializeBuffer()
	udp := &layers.UDP{SrcPort: layers.UDPPort(sourcePort), DstPort: layers.UDPPort(destinationPort)}
	var network serializableNetworkLayer
	etherType := layers.EthernetTypeIPv4
	if source.Is4() {
		network = &layers.IPv4{
			Version: ipIPv4Version, IHL: ipIPv4IHL, TTL: ipIPv4TTL, Protocol: layers.IPProtocolUDP,
			SrcIP: source.AsSlice(), DstIP: destination.AsSlice(),
		}
	} else {
		etherType = layers.EthernetTypeIPv6
		network = &layers.IPv6{
			Version: icmpv6IPv6Version, HopLimit: icmpv6DefaultHopLimit, NextHeader: layers.IPProtocolUDP,
			SrcIP: source.AsSlice(), DstIP: destination.AsSlice(),
		}
	}
	if err := udp.SetNetworkLayerForChecksum(network); err != nil {
		return fmt.Errorf("set notification checksum: %w", err)
	}
	err := gopacket.SerializeLayers(buffer, gopacket.SerializeOptions{FixLengths: true, ComputeChecksums: true},
		&layers.Ethernet{SrcMAC: device.MACAddress, DstMAC: destinationMAC, EthernetType: etherType},
		network, udp, gopacket.Payload(payload))
	if err != nil {
		return fmt.Errorf("serialize management notification: %w", err)
	}
	s.queueFrame(device, vlan, buffer.Bytes())
	return nil
}

func (s *stackDatagramSender) queueFrame(device *config.Device, vlan int, frame []byte) {
	s.stack.mu.Lock()
	s.stack.serialNumber++
	serial := s.stack.serialNumber
	s.stack.mu.Unlock()
	s.stack.Send(&Packet{
		Buffer: append([]byte(nil), frame...), Length: len(frame), SerialNumber: serial,
		Device: device, VLAN: vlan,
	})
}

type stateNotificationRegistration struct {
	device         *config.Device
	store          *devicestate.Store
	interfaceIndex func(string) (int, bool)
	vlan           int
	cursor         uint64
}

type stateNotificationManager struct {
	mu            sync.Mutex
	wake          chan struct{}
	registrations map[*config.Device]*stateNotificationRegistration
	sender        datagramSender
	started       time.Time
}

func newStateNotificationManager(stack *Stack) *stateNotificationManager {
	return &stateNotificationManager{
		wake: make(chan struct{}, 1), registrations: make(map[*config.Device]*stateNotificationRegistration),
		sender: newStackDatagramSender(stack), started: time.Now(),
	}
}

func (m *stateNotificationManager) Register(
	device *config.Device,
	store *devicestate.Store,
	interfaceIndex func(string) (int, bool),
	vlan int,
) {
	if !notificationsEnabled(device) || store == nil {
		return
	}
	store.SetChangeSignal(m.wake)
	m.mu.Lock()
	m.registrations[device] = &stateNotificationRegistration{
		device: device, store: store, interfaceIndex: interfaceIndex, vlan: vlan,
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
	if sender, ok := m.sender.(*stackDatagramSender); ok {
		sender.reset()
	}
}

func (m *stateNotificationManager) observeNeighbor(vlan int, address netip.Addr, mac net.HardwareAddr) {
	if sender, ok := m.sender.(*stackDatagramSender); ok {
		sender.observeNeighbor(vlan, address, mac)
	}
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
			m.send(device, receiver, syslogPort, payload)
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
		m.send(device, normalizeReceiver(receiver, snmp.DefaultSNMPTrapPort), snmp.DefaultSNMPTrapPort, payload)
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

func (m *stateNotificationManager) send(
	device *config.Device,
	receiver string,
	sourcePort uint16,
	payload []byte,
) {
	registration := m.registrations[device]
	if registration == nil {
		return
	}
	if err := m.sender.Send(device, registration.vlan, receiver, sourcePort, payload); err != nil {
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
