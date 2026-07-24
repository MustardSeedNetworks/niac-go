package protocols

import (
	"bytes"
	"fmt"
	"net"
	"net/netip"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/layers"
	"github.com/gosnmp/gosnmp"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
	"github.com/MustardSeedNetworks/niac-go/internal/fabric"
	"github.com/MustardSeedNetworks/niac-go/internal/logging"
	"github.com/MustardSeedNetworks/niac-go/internal/protocols/snmp"
)

type recordedDatagram struct {
	device     *config.Device
	vlan       int
	address    string
	sourcePort uint16
	payload    []byte
}

type recordingDatagramSender struct {
	mu        sync.Mutex
	datagrams []recordedDatagram
}

func (s *recordingDatagramSender) Send(
	device *config.Device,
	vlan int,
	address string,
	sourcePort uint16,
	payload []byte,
) error {
	s.mu.Lock()
	s.datagrams = append(s.datagrams, recordedDatagram{
		device: device, vlan: vlan, address: address, sourcePort: sourcePort,
		payload: append([]byte(nil), payload...),
	})
	s.mu.Unlock()
	return nil
}

func TestStateTransitionEmitsMatchingSyslogAndSNMPLinkNotification(t *testing.T) {
	device := notificationTestDevice()
	store := devicestate.NewStore(devicestate.Identity{Hostname: "edge-1"})
	store.ReplaceNetwork(devicestate.Network{Interfaces: []devicestate.Interface{{
		Name: "Gi0/1", Address: netip.MustParsePrefix("10.0.0.1/24"), AdminUp: true, OperUp: true,
	}}})
	sender := &recordingDatagramSender{}
	manager := newStateNotificationManager(nil)
	manager.sender = sender
	manager.Register(device, store, func(name string) (int, bool) {
		return 47, name == "Gi0/1"
	}, 200)
	manager.dispatchPending()
	sender.datagrams = nil

	if err := store.UpdateInterface("Gi0/1", func(iface devicestate.Interface) (devicestate.Interface, error) {
		iface.AdminUp = false
		iface.OperUp = false
		return iface, nil
	}); err != nil {
		t.Fatalf("UpdateInterface() error = %v", err)
	}
	manager.dispatchPending()

	if len(sender.datagrams) != 2 {
		t.Fatalf("datagrams = %d, want SYSLOG and SNMP", len(sender.datagrams))
	}
	event := store.Events()[len(store.Events())-1]
	assertRecordedSyslog(t, sender.datagrams[0], device, event)
	decoder := gosnmp.GoSNMP{Transport: "udp", Version: gosnmp.Version2c}
	packet, err := decoder.SnmpDecodePacket(sender.datagrams[1].payload)
	if err != nil {
		t.Fatalf("decode trap: %v", err)
	}
	if !matchesLinkDownTrap(packet, event.Version) || sender.datagrams[1].sourcePort != snmp.DefaultSNMPTrapPort ||
		sender.datagrams[1].address != "192.0.2.20:162" {
		t.Fatalf("trap = %#v to %q", packet, sender.datagrams[1].address)
	}
}

func assertRecordedSyslog(
	t *testing.T,
	datagram recordedDatagram,
	device *config.Device,
	event devicestate.Event,
) {
	t.Helper()
	payload := string(datagram.payload)
	if datagram.device != device || datagram.vlan != 200 || datagram.sourcePort != syslogPort ||
		datagram.address != "192.0.2.10:514" ||
		!strings.Contains(payload, fmt.Sprintf(`version="%d"`, event.Version)) ||
		!strings.Contains(payload, `kind="interface.updated"`) {
		t.Fatalf("SYSLOG datagram = %q to %q", payload, datagram.address)
	}
}

func matchesLinkDownTrap(packet *gosnmp.SnmpPacket, version uint64) bool {
	return packet.PDUType == gosnmp.SNMPv2Trap && packet.RequestID == uint32(version) &&
		packet.Variables[1].Value == snmp.OIDLinkDown &&
		packet.Variables[2].Name == ".1.3.6.1.2.1.2.2.1.1.47" && packet.Variables[2].Value == 47 &&
		packet.Variables[3].Name == ".1.3.6.1.2.1.2.2.1.7.47" &&
		packet.Variables[3].Value == snmp.IfStatusDown && packet.Variables[4].Value == snmp.IfStatusDown &&
		string(packet.Variables[6].Value.([]byte)) == "edge-1" && packet.Variables[7].Value == "10.0.0.1"
}

func TestInterfaceDescriptionChangeDoesNotEmitLinkTrap(t *testing.T) {
	device := notificationTestDevice()
	device.SyslogConfig = nil
	store := devicestate.NewStore(devicestate.Identity{Hostname: "edge-1"})
	store.ReplaceNetwork(devicestate.Network{Interfaces: []devicestate.Interface{{
		Name: "Gi0/1", Address: netip.MustParsePrefix("10.0.0.1/24"), AdminUp: true, OperUp: true,
	}}})
	sender := &recordingDatagramSender{}
	manager := newStateNotificationManager(nil)
	manager.sender = sender
	manager.Register(device, store, nil, 0)
	manager.dispatchPending()

	if err := store.UpdateInterface("Gi0/1", func(iface devicestate.Interface) (devicestate.Interface, error) {
		iface.Description = "uplink"
		return iface, nil
	}); err != nil {
		t.Fatalf("UpdateInterface() error = %v", err)
	}
	manager.dispatchPending()
	if len(sender.datagrams) != 0 {
		t.Fatalf("description-only change emitted %d traps", len(sender.datagrams))
	}
}

func TestAdministrativeChangeWithoutOperationalTransitionDoesNotEmitLinkTrap(t *testing.T) {
	device := notificationTestDevice()
	device.SyslogConfig = nil
	store := devicestate.NewStore(devicestate.Identity{Hostname: "edge-1"})
	store.ReplaceNetwork(devicestate.Network{Interfaces: []devicestate.Interface{{
		Name: "Gi0/1", AdminUp: true, OperUp: true,
	}}})
	sender := &recordingDatagramSender{}
	manager := newStateNotificationManager(nil)
	manager.sender = sender
	manager.Register(device, store, nil, 0)
	manager.dispatchPending()
	if err := store.UpdateInterface("Gi0/1", func(iface devicestate.Interface) (devicestate.Interface, error) {
		iface.AdminUp = false
		return iface, nil
	}); err != nil {
		t.Fatal(err)
	}
	manager.dispatchPending()
	if len(sender.datagrams) != 0 {
		t.Fatalf("admin-only transition emitted %d link traps", len(sender.datagrams))
	}
}

func TestLinkTrapReportsIndependentAdminAndOperationalStatus(t *testing.T) {
	device := notificationTestDevice()
	device.SyslogConfig = nil
	store := devicestate.NewStore(devicestate.Identity{Hostname: "edge-1"})
	store.ReplaceNetwork(devicestate.Network{Interfaces: []devicestate.Interface{{
		Name: "Gi0/1", AdminUp: false, OperUp: false,
	}}})
	sender := &recordingDatagramSender{}
	manager := newStateNotificationManager(nil)
	manager.sender = sender
	manager.Register(device, store, nil, 0)
	manager.dispatchPending()
	if err := store.UpdateInterface("Gi0/1", func(iface devicestate.Interface) (devicestate.Interface, error) {
		iface.OperUp = true
		return iface, nil
	}); err != nil {
		t.Fatal(err)
	}
	manager.dispatchPending()
	decoder := gosnmp.GoSNMP{Transport: "udp", Version: gosnmp.Version2c}
	packet, err := decoder.SnmpDecodePacket(sender.datagrams[0].payload)
	if err != nil {
		t.Fatal(err)
	}
	statuses := make(map[string]int)
	for _, variable := range packet.Variables {
		if value, ok := variable.Value.(int); ok {
			statuses[variable.Name] = value
		}
	}
	if statuses[".1.3.6.1.2.1.2.2.1.7.1"] != snmp.IfStatusDown ||
		statuses[".1.3.6.1.2.1.2.2.1.8.1"] != snmp.IfStatusUp {
		t.Fatalf("link statuses = %#v", statuses)
	}
}

func TestFormatSyslogUsesNilValueForInvalidHostname(t *testing.T) {
	event := devicestate.Event{Timestamp: time.Date(2026, time.July, 22, 12, 0, 0, 0, time.UTC)}
	for _, hostname := range []string{"", "edge router", "edge\nrouter", strings.Repeat("a", 256), "routér"} {
		t.Run(hostname, func(t *testing.T) {
			message := formatSyslog(hostname, event)
			if !strings.HasPrefix(message, "<133>1 2026-07-22T12:00:00Z - niac ") {
				t.Fatalf("formatSyslog() = %q", message)
			}
		})
	}
}

func TestFormatSyslogPreservesValidHostname(t *testing.T) {
	event := devicestate.Event{Timestamp: time.Date(2026, time.July, 22, 12, 0, 0, 0, time.UTC)}
	message := formatSyslog("edge-1.example", event)
	if !strings.HasPrefix(message, "<133>1 2026-07-22T12:00:00Z edge-1.example niac ") {
		t.Fatalf("formatSyslog() = %q", message)
	}
}

func TestStackDatagramSenderInjectsAttributedNotificationFrame(t *testing.T) {
	device := notificationTestDevice()
	device.MACAddress = mustParseMAC(t, "02:00:00:00:00:01")
	device.IPAddresses = []net.IP{net.ParseIP("192.0.2.1")}
	device.Interfaces = []config.Interface{{Name: "Gi0/1", Address: "192.0.2.1/24"}}
	receiver := &config.Device{
		Name: "collector", MACAddress: mustParseMAC(t, "02:00:00:00:00:02"),
		IPAddresses: []net.IP{net.ParseIP("192.0.2.10")},
	}
	stack := NewStack(nil, &config.Config{Devices: []config.Device{*device, *receiver}}, logging.NewDebugConfig(0))
	registered := &stack.config.Devices[0]

	err := stack.notifications.sender.Send(registered, 0, "192.0.2.10:514", syslogPort, []byte("event"))
	if err != nil {
		t.Fatalf("Send() error = %v", err)
	}
	packet := <-stack.sendQueue
	decoded := gopacket.NewPacket(packet.Buffer, layers.LayerTypeEthernet, gopacket.Default)
	ethernet, _ := decoded.Layer(layers.LayerTypeEthernet).(*layers.Ethernet)
	ipv4, _ := decoded.Layer(layers.LayerTypeIPv4).(*layers.IPv4)
	udp, _ := decoded.Layer(layers.LayerTypeUDP).(*layers.UDP)
	if ethernet == nil || ipv4 == nil || udp == nil ||
		!bytes.Equal(ethernet.SrcMAC, registered.MACAddress) ||
		!bytes.Equal(ethernet.DstMAC, stack.config.Devices[1].MACAddress) ||
		!ipv4.SrcIP.Equal(net.ParseIP("192.0.2.1")) || !ipv4.DstIP.Equal(net.ParseIP("192.0.2.10")) ||
		udp.SrcPort != syslogPort || udp.DstPort != syslogPort || string(udp.Payload) != "event" {
		t.Fatalf("notification frame = ethernet %#v IPv4 %#v UDP %#v", ethernet, ipv4, udp)
	}
}

func TestStackDatagramSenderUsesConfiguredNextHopForOffLinkReceiver(t *testing.T) {
	device := notificationTestDevice()
	device.MACAddress = mustParseMAC(t, "02:00:00:00:00:01")
	device.IPAddresses = []net.IP{net.ParseIP("10.0.0.1")}
	device.Interfaces = []config.Interface{{Name: "Gi0/1", Address: "10.0.0.1/24"}}
	device.Routes = []config.Route{{
		Destination: "192.0.2.0/24", Via: "Gi0/1", NextHop: "10.0.0.2",
	}}
	gateway := config.Device{
		Name: "gateway", MACAddress: mustParseMAC(t, "02:00:00:00:00:02"),
		IPAddresses: []net.IP{net.ParseIP("10.0.0.2")},
		Interfaces:  []config.Interface{{Name: "Gi0/1", Address: "10.0.0.2/24"}},
	}
	stack := NewStack(nil, &config.Config{Devices: []config.Device{*device, gateway}}, logging.NewDebugConfig(0))
	registered := &stack.config.Devices[0]

	if err := stack.notifications.sender.Send(
		registered, 0, "192.0.2.10:514", syslogPort, []byte("event"),
	); err != nil {
		t.Fatalf("Send() error = %v", err)
	}
	packet := <-stack.sendQueue
	decoded := gopacket.NewPacket(packet.Buffer, layers.LayerTypeEthernet, gopacket.Default)
	ethernet, _ := decoded.Layer(layers.LayerTypeEthernet).(*layers.Ethernet)
	if ethernet == nil || !bytes.Equal(ethernet.DstMAC, gateway.MACAddress) {
		t.Fatalf("destination MAC = %v, want next hop %s", ethernet, gateway.MACAddress)
	}
}

func TestStackDatagramSenderRoutesThroughFabricToPhysicalCollector(t *testing.T) {
	routerMAC := mustForwardingMAC(t, "02:00:00:00:00:01")
	senderMAC := mustForwardingMAC(t, "02:00:00:00:00:10")
	collectorMAC := mustForwardingMAC(t, "02:00:00:00:00:fe")
	cfg := &config.Config{
		Networks: []config.Network{
			{Name: "attachment", Subnet: "10.10.200.0/24"},
			{Name: "management", Subnet: "10.20.0.0/24"},
		},
		Attachments: []config.LogicalAttachment{{Name: "tester", Network: "attachment"}},
		Devices: []config.Device{
			{
				Name: "edge", Type: "router", MACAddress: routerMAC,
				Interfaces: []config.Interface{
					{Name: "outside", Network: "attachment", Address: "10.10.200.1/24"},
					{Name: "inside", Network: "management", Address: "10.20.0.1/24"},
				},
			},
			{
				Name: "sender", Type: "server", MACAddress: senderMAC,
				Interfaces: []config.Interface{{
					Name: "eth0", Network: "management", Address: "10.20.0.10/24",
				}},
				Routes: []config.Route{{
					Destination: "0.0.0.0/0", Via: "eth0", NextHop: "10.20.0.1",
				}},
			},
		},
	}
	report := fabric.Compile(cfg, fabric.Binding{
		Attachment: "tester", Interface: "eth0", Mode: fabric.ModeAccess, AccessVLAN: 200,
		PolicyApproved: true,
	})
	if !report.Safe {
		t.Fatalf("Compile() diagnostics = %#v", report.Diagnostics)
	}
	stack := NewStack(nil, cfg, logging.NewDebugConfig(0))
	stack.ConfigureFabric(&report.Topology)
	sender := &cfg.Devices[1]
	datagrams := stack.notifications.sender.(*stackDatagramSender)
	defer datagrams.reset()

	if err := datagrams.Send(
		sender, config.UntaggedTag, "10.10.200.100:514", syslogPort, []byte("event"),
	); err != nil {
		t.Fatalf("Send() error = %v", err)
	}
	probe := <-stack.sendQueue
	assertPhysicalCollectorProbe(t, probe, routerMAC)

	reply, err := serializeARPPacket(
		&layers.Ethernet{
			SrcMAC: collectorMAC, DstMAC: routerMAC, EthernetType: layers.EthernetTypeARP,
		},
		buildARPLayer(
			layers.ARPReply, collectorMAC, net.ParseIP("10.10.200.100"),
			routerMAC, net.ParseIP("10.10.200.1"),
		),
	)
	if err != nil {
		t.Fatalf("serialize collector ARP reply: %v", err)
	}
	stack.arpHandler.HandlePacket(&Packet{Buffer: reply, Length: len(reply), VLAN: config.UntaggedTag})

	notification := <-stack.sendQueue
	assertPhysicalCollectorNotification(t, notification, routerMAC, collectorMAC)
	if got := stack.GetStats().FabricForwarded; got != 1 {
		t.Fatalf("FabricForwarded = %d, want 1", got)
	}
}

func assertPhysicalCollectorProbe(t *testing.T, probe *Packet, routerMAC net.HardwareAddr) {
	t.Helper()
	probePacket := gopacket.NewPacket(probe.Buffer, layers.LayerTypeEthernet, gopacket.Default)
	probeEthernet, _ := probePacket.Layer(layers.LayerTypeEthernet).(*layers.Ethernet)
	arpRequest, _ := probePacket.Layer(layers.LayerTypeARP).(*layers.ARP)
	if probeEthernet == nil || arpRequest == nil ||
		!bytes.Equal(probeEthernet.SrcMAC, routerMAC) ||
		!net.IP(arpRequest.SourceProtAddress).Equal(net.ParseIP("10.10.200.1")) ||
		!net.IP(arpRequest.DstProtAddress).Equal(net.ParseIP("10.10.200.100")) {
		t.Fatalf("physical collector probe = ethernet %#v ARP %#v", probeEthernet, arpRequest)
	}
}

func assertPhysicalCollectorNotification(
	t *testing.T,
	notification *Packet,
	routerMAC net.HardwareAddr,
	collectorMAC net.HardwareAddr,
) {
	t.Helper()
	decoded := gopacket.NewPacket(notification.Buffer, layers.LayerTypeEthernet, gopacket.Default)
	ethernet, _ := decoded.Layer(layers.LayerTypeEthernet).(*layers.Ethernet)
	ipv4, _ := decoded.Layer(layers.LayerTypeIPv4).(*layers.IPv4)
	udp, _ := decoded.Layer(layers.LayerTypeUDP).(*layers.UDP)
	if ethernet == nil || ipv4 == nil || udp == nil ||
		!bytes.Equal(ethernet.SrcMAC, routerMAC) ||
		!bytes.Equal(ethernet.DstMAC, collectorMAC) ||
		!ipv4.SrcIP.Equal(net.ParseIP("10.20.0.10")) ||
		!ipv4.DstIP.Equal(net.ParseIP("10.10.200.100")) ||
		ipv4.TTL != ipIPv4TTL-1 ||
		string(udp.Payload) != "event" {
		t.Fatalf("physical collector notification = ethernet %#v IPv4 %#v UDP %#v", ethernet, ipv4, udp)
	}
	if trace := notification.FabricTrace(); trace.RouteDecision != fabricRouteDecisionForwarded ||
		trace.EgressNetwork != "attachment" {
		t.Fatalf("notification FabricTrace() = %#v", trace)
	}
}

func TestStackDatagramSenderResolvesPhysicalNextHopBeforeNotification(t *testing.T) {
	device := notificationTestDevice()
	device.VLAN = 200
	device.MACAddress = mustParseMAC(t, "02:00:00:00:00:01")
	device.IPAddresses = []net.IP{net.ParseIP("10.0.0.1")}
	device.Interfaces = []config.Interface{{Name: "Gi0/1", Address: "10.0.0.1/24"}}
	device.Routes = []config.Route{{
		Destination: "192.0.2.0/24", Via: "Gi0/1", NextHop: "10.0.0.254",
	}}
	stack := NewStack(nil, &config.Config{Devices: []config.Device{*device}}, logging.NewDebugConfig(0))
	stack.fabric = &fabricRuntime{binding: fabric.CompiledBinding{WireTagged: false}}
	registered := &stack.config.Devices[0]

	if err := stack.notifications.sender.Send(
		registered, 0, "192.0.2.10:514", syslogPort, []byte("event"),
	); err != nil {
		t.Fatalf("Send() error = %v", err)
	}
	request := <-stack.sendQueue
	if request.VLAN != config.UntaggedTag {
		t.Fatalf("notification probe VLAN = %d, want untagged", request.VLAN)
	}
	requestPacket := gopacket.NewPacket(request.Buffer, layers.LayerTypeEthernet, gopacket.Default)
	arpRequest, _ := requestPacket.Layer(layers.LayerTypeARP).(*layers.ARP)
	if arpRequest == nil || arpRequest.Operation != layers.ARPRequest ||
		!net.IP(arpRequest.DstProtAddress).Equal(net.ParseIP("10.0.0.254")) {
		t.Fatalf("ARP request = %#v", arpRequest)
	}

	gatewayMAC := mustParseMAC(t, "02:00:00:00:00:fe")
	replyFrame, err := serializeARPPacket(
		&layers.Ethernet{
			SrcMAC: gatewayMAC, DstMAC: registered.MACAddress, EthernetType: layers.EthernetTypeARP,
		},
		buildARPLayer(
			layers.ARPReply, gatewayMAC, net.ParseIP("10.0.0.254"),
			registered.MACAddress, net.ParseIP("10.0.0.1"),
		),
	)
	if err != nil {
		t.Fatalf("serialize ARP reply: %v", err)
	}
	stack.arpHandler.HandlePacket(&Packet{Buffer: replyFrame, Length: len(replyFrame), VLAN: -1})

	notification := <-stack.sendQueue
	decoded := gopacket.NewPacket(notification.Buffer, layers.LayerTypeEthernet, gopacket.Default)
	ethernet, _ := decoded.Layer(layers.LayerTypeEthernet).(*layers.Ethernet)
	udp, _ := decoded.Layer(layers.LayerTypeUDP).(*layers.UDP)
	if ethernet == nil || udp == nil || !bytes.Equal(ethernet.DstMAC, gatewayMAC) ||
		string(udp.Payload) != "event" {
		t.Fatalf("resolved notification = ethernet %#v UDP %#v", ethernet, udp)
	}
}

func TestStackDatagramSenderRefreshesAndExpiresLearnedNeighbors(t *testing.T) {
	device := notificationTestDevice()
	device.MACAddress = mustParseMAC(t, "02:00:00:00:00:01")
	device.IPAddresses = []net.IP{net.ParseIP("10.0.0.1")}
	device.Interfaces = []config.Interface{{Name: "Gi0/1", Address: "10.0.0.1/24"}}
	stack := NewStack(nil, &config.Config{Devices: []config.Device{*device}}, logging.NewDebugConfig(0))
	registered := &stack.config.Devices[0]
	sender := stack.notifications.sender.(*stackDatagramSender)
	current := time.Date(2026, time.July, 22, 12, 0, 0, 0, time.UTC)
	sender.now = func() time.Time { return current }
	target := netip.MustParseAddr("10.0.0.100")
	firstMAC := mustParseMAC(t, "02:00:00:00:00:10")
	secondMAC := mustParseMAC(t, "02:00:00:00:00:20")

	sender.observeNeighbor(-1, target, firstMAC)
	got, err := sender.notificationDestinationMAC(registered, config.UntaggedTag, target, target)
	if err != nil || !bytes.Equal(got, firstMAC) {
		t.Fatalf("first learned neighbor = %s, %v; want %s", got, err, firstMAC)
	}

	current = current.Add(notificationNeighborTTL)
	got, err = sender.notificationDestinationMAC(registered, config.UntaggedTag, target, target)
	if err != nil || got != nil {
		t.Fatalf("expired neighbor = %s, %v; want nil", got, err)
	}

	sender.observeNeighbor(config.UntaggedTag, target, secondMAC)
	got, err = sender.notificationDestinationMAC(registered, -1, target, target)
	if err != nil || !bytes.Equal(got, secondMAC) {
		t.Fatalf("refreshed neighbor = %s, %v; want %s", got, err, secondMAC)
	}
}

func TestStackDatagramSenderResolvesPhysicalOnLinkCollector(t *testing.T) {
	device := notificationTestDevice()
	device.MACAddress = mustParseMAC(t, "02:00:00:00:00:01")
	device.IPAddresses = []net.IP{net.ParseIP("10.0.0.1")}
	device.Interfaces = []config.Interface{{Name: "Gi0/1", Address: "10.0.0.1/24"}}
	stack := NewStack(nil, &config.Config{Devices: []config.Device{*device}}, logging.NewDebugConfig(0))
	registered := &stack.config.Devices[0]

	if err := stack.notifications.sender.Send(
		registered, 0, "10.0.0.100:514", syslogPort, []byte("event"),
	); err != nil {
		t.Fatalf("Send() error = %v", err)
	}
	request := <-stack.sendQueue
	packet := gopacket.NewPacket(request.Buffer, layers.LayerTypeEthernet, gopacket.Default)
	arpRequest, _ := packet.Layer(layers.LayerTypeARP).(*layers.ARP)
	if arpRequest == nil || !net.IP(arpRequest.DstProtAddress).Equal(net.ParseIP("10.0.0.100")) {
		t.Fatalf("on-link ARP request = %#v", arpRequest)
	}
	stack.notifications.sender.(*stackDatagramSender).reset()
}

func TestStackDatagramSenderRetriesUnansweredNeighborResolution(t *testing.T) {
	device := notificationTestDevice()
	device.MACAddress = mustParseMAC(t, "02:00:00:00:00:01")
	device.IPAddresses = []net.IP{net.ParseIP("10.0.0.1")}
	device.Interfaces = []config.Interface{{Name: "Gi0/1", Address: "10.0.0.1/24"}}
	stack := NewStack(nil, &config.Config{Devices: []config.Device{*device}}, logging.NewDebugConfig(0))
	registered := &stack.config.Devices[0]
	defer stack.notifications.sender.(*stackDatagramSender).reset()

	if err := stack.notifications.sender.Send(
		registered, 0, "10.0.0.100:514", syslogPort, []byte("event"),
	); err != nil {
		t.Fatalf("Send() error = %v", err)
	}
	<-stack.sendQueue
	select {
	case retry := <-stack.sendQueue:
		packet := gopacket.NewPacket(retry.Buffer, layers.LayerTypeEthernet, gopacket.Default)
		arpRetry, _ := packet.Layer(layers.LayerTypeARP).(*layers.ARP)
		if arpRetry == nil || arpRetry.Operation != layers.ARPRequest {
			t.Fatalf("retry = %#v", arpRetry)
		}
	case <-time.After(2 * notificationNeighborRetry):
		t.Fatal("neighbor resolution was not retried")
	}
}

func TestStackStopCancelsNotificationNeighborResolution(t *testing.T) {
	device := notificationTestDevice()
	device.MACAddress = mustParseMAC(t, "02:00:00:00:00:01")
	device.IPAddresses = []net.IP{net.ParseIP("10.0.0.1")}
	device.Interfaces = []config.Interface{{Name: "Gi0/1", Address: "10.0.0.1/24"}}
	stack := NewStack(nil, &config.Config{Devices: []config.Device{*device}}, logging.NewDebugConfig(0))
	registered := &stack.config.Devices[0]

	if err := stack.notifications.sender.Send(
		registered, 0, "10.0.0.100:514", syslogPort, []byte("event"),
	); err != nil {
		t.Fatalf("Send() error = %v", err)
	}
	<-stack.sendQueue
	stack.running.Store(true)
	stack.Stop()

	sender := stack.notifications.sender.(*stackDatagramSender)
	sender.mu.Lock()
	pending := len(sender.pending)
	sender.mu.Unlock()
	if pending != 0 {
		t.Fatalf("pending neighbor resolutions after Stop() = %d", pending)
	}
	select {
	case packet := <-stack.sendQueue:
		t.Fatalf("neighbor retry queued after Stop(): %#v", packet)
	case <-time.After(2 * notificationNeighborRetry):
	}
}

func TestStackDatagramSenderUsesRouteInterfaceSource(t *testing.T) {
	device := notificationTestDevice()
	device.MACAddress = mustParseMAC(t, "02:00:00:00:00:01")
	device.IPAddresses = []net.IP{net.ParseIP("10.0.0.1"), net.ParseIP("198.51.100.1")}
	device.Interfaces = []config.Interface{
		{Name: "Gi0/1", Address: "10.0.0.1/24"},
		{Name: "Gi0/2", Address: "198.51.100.1/24"},
	}
	device.Routes = []config.Route{{
		Destination: "192.0.2.0/24", Via: "Gi0/2", NextHop: "198.51.100.2",
	}}
	gateway := config.Device{
		Name: "gateway", MACAddress: mustParseMAC(t, "02:00:00:00:00:02"),
		IPAddresses: []net.IP{net.ParseIP("198.51.100.2")},
		Interfaces:  []config.Interface{{Name: "Gi0/1", Address: "198.51.100.2/24"}},
	}
	stack := NewStack(nil, &config.Config{Devices: []config.Device{*device, gateway}}, logging.NewDebugConfig(0))
	registered := &stack.config.Devices[0]

	if err := stack.notifications.sender.Send(
		registered, 0, "192.0.2.10:514", syslogPort, []byte("event"),
	); err != nil {
		t.Fatalf("Send() error = %v", err)
	}
	packet := <-stack.sendQueue
	decoded := gopacket.NewPacket(packet.Buffer, layers.LayerTypeEthernet, gopacket.Default)
	ipv4, _ := decoded.Layer(layers.LayerTypeIPv4).(*layers.IPv4)
	if ipv4 == nil || !ipv4.SrcIP.Equal(net.ParseIP("198.51.100.1")) {
		t.Fatalf("source IP = %v, want 198.51.100.1", ipv4)
	}
}

func TestStackDatagramSenderPrefersMoreSpecificRouteOverConnectedNetwork(t *testing.T) {
	device := notificationTestDevice()
	device.MACAddress = mustParseMAC(t, "02:00:00:00:00:01")
	device.IPAddresses = []net.IP{net.ParseIP("10.0.0.1")}
	device.Interfaces = []config.Interface{{Name: "Gi0/1", Address: "10.0.0.1/8"}}
	device.Routes = []config.Route{{
		Destination: "10.100.0.0/16", Via: "Gi0/1", NextHop: "10.0.0.2",
	}}
	gateway := config.Device{
		Name: "gateway", MACAddress: mustParseMAC(t, "02:00:00:00:00:02"),
		IPAddresses: []net.IP{net.ParseIP("10.0.0.2")},
		Interfaces:  []config.Interface{{Name: "Gi0/1", Address: "10.0.0.2/8"}},
	}
	stack := NewStack(nil, &config.Config{Devices: []config.Device{*device, gateway}}, logging.NewDebugConfig(0))
	registered := &stack.config.Devices[0]

	if err := stack.notifications.sender.Send(
		registered, 0, "10.100.0.100:514", syslogPort, []byte("event"),
	); err != nil {
		t.Fatalf("Send() error = %v", err)
	}
	packet := <-stack.sendQueue
	decoded := gopacket.NewPacket(packet.Buffer, layers.LayerTypeEthernet, gopacket.Default)
	ethernet, _ := decoded.Layer(layers.LayerTypeEthernet).(*layers.Ethernet)
	if ethernet == nil || !bytes.Equal(ethernet.DstMAC, gateway.MACAddress) {
		t.Fatalf("destination MAC = %v, want next hop %s", ethernet, gateway.MACAddress)
	}
}

func TestStackDatagramSenderPreservesIPOnlyFlatNotifications(t *testing.T) {
	device := notificationTestDevice()
	device.MACAddress = mustParseMAC(t, "02:00:00:00:00:01")
	device.IPAddresses = []net.IP{net.ParseIP("10.0.0.1")}
	receiver := config.Device{
		Name: "collector", MACAddress: mustParseMAC(t, "02:00:00:00:00:02"),
		IPAddresses: []net.IP{net.ParseIP("10.0.0.100")},
	}
	stack := NewStack(nil, &config.Config{Devices: []config.Device{*device, receiver}}, logging.NewDebugConfig(0))
	registered := &stack.config.Devices[0]

	if err := stack.notifications.sender.Send(
		registered, 0, "10.0.0.100:162", snmp.DefaultSNMPTrapPort, []byte("trap"),
	); err != nil {
		t.Fatalf("Send() error = %v", err)
	}
	packet := <-stack.sendQueue
	decoded := gopacket.NewPacket(packet.Buffer, layers.LayerTypeEthernet, gopacket.Default)
	ethernet, _ := decoded.Layer(layers.LayerTypeEthernet).(*layers.Ethernet)
	ipv4, _ := decoded.Layer(layers.LayerTypeIPv4).(*layers.IPv4)
	if ethernet == nil || ipv4 == nil || !bytes.Equal(ethernet.DstMAC, receiver.MACAddress) ||
		!ipv4.SrcIP.Equal(net.ParseIP("10.0.0.1")) {
		t.Fatalf("IP-only notification = ethernet %#v IPv4 %#v", ethernet, ipv4)
	}
}

func TestStackDatagramSenderPreservesAddressBackedNamedInterfaceNotifications(t *testing.T) {
	device := notificationTestDevice()
	device.MACAddress = mustParseMAC(t, "02:00:00:00:00:01")
	device.IPAddresses = []net.IP{net.ParseIP("10.0.0.1")}
	device.Interfaces = []config.Interface{{Name: "Management"}}
	receiver := config.Device{
		Name: "collector", MACAddress: mustParseMAC(t, "02:00:00:00:00:02"),
		IPAddresses: []net.IP{net.ParseIP("10.0.0.100")},
	}
	stack := NewStack(nil, &config.Config{Devices: []config.Device{*device, receiver}}, logging.NewDebugConfig(0))
	registered := &stack.config.Devices[0]

	if err := stack.notifications.sender.Send(
		registered, 0, "10.0.0.100:162", snmp.DefaultSNMPTrapPort, []byte("trap"),
	); err != nil {
		t.Fatalf("Send() error = %v", err)
	}
	packet := <-stack.sendQueue
	decoded := gopacket.NewPacket(packet.Buffer, layers.LayerTypeEthernet, gopacket.Default)
	ipv4, _ := decoded.Layer(layers.LayerTypeIPv4).(*layers.IPv4)
	if ipv4 == nil || !ipv4.SrcIP.Equal(net.ParseIP("10.0.0.1")) {
		t.Fatalf("named-interface notification IPv4 = %#v", ipv4)
	}
}

func TestStackDatagramSenderEmitsIPv6TrapFrame(t *testing.T) {
	device := notificationTestDevice()
	device.MACAddress = mustParseMAC(t, "02:00:00:00:00:01")
	device.IPAddresses = []net.IP{net.ParseIP("2001:db8::1")}
	device.Interfaces = []config.Interface{{Name: "Gi0/1", Address: "2001:db8::1/64"}}
	receiver := config.Device{
		Name: "collector", MACAddress: mustParseMAC(t, "02:00:00:00:00:02"),
		IPAddresses: []net.IP{net.ParseIP("2001:db8::10")},
	}
	stack := NewStack(nil, &config.Config{Devices: []config.Device{*device, receiver}}, logging.NewDebugConfig(0))
	registered := &stack.config.Devices[0]

	if err := stack.notifications.sender.Send(
		registered, 0, "[2001:db8::10]:162", snmp.DefaultSNMPTrapPort, []byte("trap"),
	); err != nil {
		t.Fatalf("Send() error = %v", err)
	}
	packet := <-stack.sendQueue
	decoded := gopacket.NewPacket(packet.Buffer, layers.LayerTypeEthernet, gopacket.Default)
	ethernet, _ := decoded.Layer(layers.LayerTypeEthernet).(*layers.Ethernet)
	ipv6, _ := decoded.Layer(layers.LayerTypeIPv6).(*layers.IPv6)
	udp, _ := decoded.Layer(layers.LayerTypeUDP).(*layers.UDP)
	if ethernet == nil || ipv6 == nil || udp == nil || ethernet.EthernetType != layers.EthernetTypeIPv6 ||
		!bytes.Equal(ethernet.DstMAC, receiver.MACAddress) ||
		!ipv6.SrcIP.Equal(net.ParseIP("2001:db8::1")) || !ipv6.DstIP.Equal(net.ParseIP("2001:db8::10")) ||
		udp.DstPort != snmp.DefaultSNMPTrapPort || string(udp.Payload) != "trap" {
		t.Fatalf("IPv6 notification = ethernet %#v IPv6 %#v UDP %#v", ethernet, ipv6, udp)
	}
}

func TestStackDatagramSenderResolvesRoutedIPv6NextHop(t *testing.T) {
	device := notificationTestDevice()
	device.MACAddress = mustParseMAC(t, "02:00:00:00:00:01")
	device.IPAddresses = []net.IP{net.ParseIP("2001:db8:1::1")}
	device.Interfaces = []config.Interface{{Name: "Gi0/1", Address: "2001:db8:1::1/64"}}
	device.Routes = []config.Route{{
		Destination: "2001:db8:2::/64", Via: "Gi0/1", NextHop: "2001:db8:1::fe",
	}}
	stack := NewStack(nil, &config.Config{Devices: []config.Device{*device}}, logging.NewDebugConfig(0))
	registered := &stack.config.Devices[0]
	defer stack.notifications.sender.(*stackDatagramSender).reset()

	if err := stack.notifications.sender.Send(
		registered, 0, "[2001:db8:2::10]:162", snmp.DefaultSNMPTrapPort, []byte("trap"),
	); err != nil {
		t.Fatalf("Send() error = %v", err)
	}
	request := <-stack.sendQueue
	packet := gopacket.NewPacket(request.Buffer, layers.LayerTypeEthernet, gopacket.Default)
	ipv6, _ := packet.Layer(layers.LayerTypeIPv6).(*layers.IPv6)
	icmp, _ := packet.Layer(layers.LayerTypeICMPv6).(*layers.ICMPv6)
	if ipv6 == nil || icmp == nil || icmp.TypeCode.Type() != ICMPv6TypeNeighborSolicitation ||
		!ipv6.SrcIP.Equal(net.ParseIP("2001:db8:1::1")) {
		t.Fatalf("neighbor solicitation = IPv6 %#v ICMPv6 %#v", ipv6, icmp)
	}
	gatewayMAC := mustParseMAC(t, "02:00:00:00:00:fe")
	advertisement := serializeTestNeighborAdvertisement(
		t, gatewayMAC, registered.MACAddress,
		netip.MustParseAddr("2001:db8:1::fe"), netip.MustParseAddr("2001:db8:1::1"),
	)
	decodedAdvertisement := gopacket.NewPacket(advertisement, layers.LayerTypeEthernet, gopacket.Default)
	ipv6Advertisement, _ := decodedAdvertisement.Layer(layers.LayerTypeIPv6).(*layers.IPv6)
	stack.icmpv6Handler.HandlePacket(
		&Packet{Buffer: advertisement, Length: len(advertisement), VLAN: -1},
		decodedAdvertisement, ipv6Advertisement, nil,
	)
	notification := <-stack.sendQueue
	decoded := gopacket.NewPacket(notification.Buffer, layers.LayerTypeEthernet, gopacket.Default)
	ethernet, _ := decoded.Layer(layers.LayerTypeEthernet).(*layers.Ethernet)
	udp, _ := decoded.Layer(layers.LayerTypeUDP).(*layers.UDP)
	if ethernet == nil || udp == nil || !bytes.Equal(ethernet.DstMAC, gatewayMAC) ||
		string(udp.Payload) != "trap" {
		t.Fatalf("resolved IPv6 notification = ethernet %#v UDP %#v", ethernet, udp)
	}
}

func serializeTestNeighborAdvertisement(
	t *testing.T,
	sourceMAC, destinationMAC net.HardwareAddr,
	target, destination netip.Addr,
) []byte {
	t.Helper()
	ipv6 := &layers.IPv6{
		Version: 6, HopLimit: icmpv6NDPHopLimit, NextHeader: layers.IPProtocolICMPv6,
		SrcIP: target.AsSlice(), DstIP: destination.AsSlice(),
	}
	icmp := &layers.ICMPv6{TypeCode: layers.CreateICMPv6TypeCode(ICMPv6TypeNeighborAdvertisement, 0)}
	if err := icmp.SetNetworkLayerForChecksum(ipv6); err != nil {
		t.Fatalf("set neighbor advertisement checksum: %v", err)
	}
	payload := make([]byte, 28)
	copy(payload[4:20], target.AsSlice())
	payload[20], payload[21] = ICMPv6OptTargetLinkAddr, 1
	copy(payload[22:], sourceMAC)
	buffer := gopacket.NewSerializeBuffer()
	err := gopacket.SerializeLayers(buffer, gopacket.SerializeOptions{FixLengths: true, ComputeChecksums: true},
		&layers.Ethernet{SrcMAC: sourceMAC, DstMAC: destinationMAC, EthernetType: layers.EthernetTypeIPv6},
		ipv6, icmp, gopacket.Payload(payload))
	if err != nil {
		t.Fatalf("serialize neighbor advertisement: %v", err)
	}
	return buffer.Bytes()
}

func notificationTestDevice() *config.Device {
	return &config.Device{
		Name:         "edge-1",
		SyslogConfig: &config.SyslogConfig{Enabled: true, Receivers: []string{"192.0.2.10:514"}},
		SNMPConfig: config.SNMPConfig{Traps: &config.TrapConfig{
			Enabled: true, Receivers: []string{"192.0.2.20"}, Community: "traps",
			LinkState: &config.LinkStateTrapConfig{Enabled: true, LinkDown: true, LinkUp: true},
		}},
	}
}
