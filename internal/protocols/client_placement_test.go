package protocols

import (
	"net"
	"strconv"
	"strings"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/fabric"
	"github.com/MustardSeedNetworks/niac-go/internal/logging"
)

// A tester plugged into a pool port must show up where the cable is: in its
// own switch's forwarding table, at its own port, and nowhere else. Before
// AP-2 a client's MAC went into every forwarding device on the attachment
// network at a per-device authored constant port (attachment plan D3).

const (
	placementAccess     = "MED-ACC-SW01"
	placementCore       = "MED-CORE-SW01"
	placementNeighbour  = "MED-ACC-SW02"
	placementDataVLAN   = 210
	placementCommunity  = "public"
	placementPinnedPort = "GigabitEthernet1/0/45"
	dot1dTpFdbPortOID   = ".1.3.6.1.2.1.17.4.3.1.2"
	dot1qTpFdbPortOID   = ".1.3.6.1.2.1.17.7.1.2.2.1.2"
	ifDescrOID          = ".1.3.6.1.2.1.2.2.1.2"
	dot1dBasePortIfIdx  = ".1.3.6.1.2.1.17.1.4.1.2"
)

func placementPool() []string {
	return []string{"GigabitEthernet1/0/43", "GigabitEthernet1/0/44", placementPinnedPort}
}

func placementClient(last byte) net.HardwareAddr {
	return net.HardwareAddr{0x02, 0x00, 0x00, 0x00, 0x02, last}
}

func placementSNMP() config.SNMPConfig {
	return config.SNMPConfig{Community: placementCommunity}
}

func placementAccessSwitch() config.Device {
	interfaces := []config.Interface{
		{Name: "Vlan200", Network: "med-mgmt", Address: "10.51.200.21/24"},
		{Name: "HundredGigabitEthernet1/0/49", VLANs: []int{200, placementDataVLAN}},
	}
	for _, port := range placementPool() {
		interfaces = append(interfaces, config.Interface{Name: port, VLANs: []int{placementDataVLAN}})
	}
	return config.Device{
		Name:        placementAccess,
		Type:        "switch",
		MACAddress:  net.HardwareAddr{0x00, 0x51, 0x00, 0x00, 0x00, 0x21},
		IPAddresses: []net.IP{net.ParseIP("10.51.200.21")},
		Interfaces:  interfaces,
		SNMPConfig:  placementSNMP(),
		TrunkPorts: []config.TrunkPort{{
			Interface:    "HundredGigabitEthernet1/0/49",
			VLANs:        []int{200, placementDataVLAN},
			NativeVLAN:   200,
			RemoteDevice: placementCore,
		}},
	}
}

func placementNeighbourSwitch() config.Device {
	return config.Device{
		Name:        placementNeighbour,
		Type:        "switch",
		MACAddress:  net.HardwareAddr{0x00, 0x51, 0x00, 0x00, 0x00, 0x22},
		IPAddresses: []net.IP{net.ParseIP("10.51.200.22")},
		Interfaces: []config.Interface{
			{Name: "Vlan200", Network: "med-mgmt", Address: "10.51.200.22/24"},
			{Name: "HundredGigabitEthernet1/0/49", VLANs: []int{200, placementDataVLAN}},
		},
		SNMPConfig: placementSNMP(),
		TrunkPorts: []config.TrunkPort{{
			Interface:    "HundredGigabitEthernet1/0/49",
			VLANs:        []int{200, placementDataVLAN},
			NativeVLAN:   200,
			RemoteDevice: placementCore,
		}},
	}
}

// placementCoreSwitch owns the attachment network's SVI and authors the
// constant FDB injection the pool must override: before AP-2 a DHCP ACK put
// every client into its table at bridge port 1.
func placementCoreSwitch() config.Device {
	snmp := placementSNMP()
	snmp.Dot1DFdbTable = &config.FdbTableConfig{Port: 1}
	snmp.Dot1QFdbTable = &config.FdbTableConfig{Port: 1, VLAN: placementDataVLAN}
	return config.Device{
		Name:        placementCore,
		Type:        "layer3-switch",
		MACAddress:  net.HardwareAddr{0x00, 0x51, 0x00, 0x00, 0x00, 0x02},
		IPAddresses: []net.IP{net.ParseIP("10.51.210.2")},
		Interfaces: []config.Interface{
			{Name: "Vlan200", Network: "med-mgmt", Address: "10.51.200.2/24"},
			{Name: "Vlan210", Network: "med-data", Address: "10.51.210.2/24"},
			{Name: "HundredGigabitEthernet1/0/1", VLANs: []int{200, placementDataVLAN}},
			{Name: "HundredGigabitEthernet1/0/2", VLANs: []int{200, placementDataVLAN}},
		},
		SNMPConfig: snmp,
		TrunkPorts: []config.TrunkPort{
			{
				Interface:    "HundredGigabitEthernet1/0/1",
				VLANs:        []int{200, placementDataVLAN},
				NativeVLAN:   200,
				RemoteDevice: placementAccess,
			},
			{
				Interface:    "HundredGigabitEthernet1/0/2",
				VLANs:        []int{200, placementDataVLAN},
				NativeVLAN:   200,
				RemoteDevice: placementNeighbour,
			},
		},
	}
}

func placementStack(t *testing.T, pins ...config.AttachmentPin) *Stack {
	t.Helper()
	return placementStackOn(t, idleTransport{}, pins...)
}

func placementStackOn(t *testing.T, transport PacketTransport, pins ...config.AttachmentPin) *Stack {
	t.Helper()
	cfg := &config.Config{
		Networks: []config.Network{
			{Name: "med-mgmt", Subnet: "10.51.200.0/24", VirtualVLAN: 200},
			{Name: "med-data", Subnet: "10.51.210.0/24", VirtualVLAN: placementDataVLAN},
		},
		Attachments: []config.LogicalAttachment{{
			Name: "cyberscope",
			At:   &config.AttachmentPort{Device: placementAccess, Ports: placementPool()},
			Pins: pins,
		}},
		Devices: []config.Device{placementAccessSwitch(), placementCoreSwitch(), placementNeighbourSwitch()},
	}
	report := fabric.Compile(cfg, fabric.Binding{
		Attachment: "cyberscope", Interface: "eth0", Mode: fabric.ModeAccess,
		AccessVLAN: 200, PolicyApproved: true,
	})
	if !report.Safe {
		t.Fatalf("Compile() diagnostics = %#v", report.Diagnostics)
	}
	stack := NewStackWithTransport(transport, cfg, logging.NewDebugConfig(0))
	stack.ConfigureFabric(&report.Topology)
	return stack
}

// sendFrom drives one frame from mac through decodePacket, the funnel every
// received frame passes, exactly as a client on the wire would.
func sendFrom(stack *Stack, mac net.HardwareAddr, ip string) {
	stack.decodePacket(ipv4Frame(mac, deviceMAC(), ip, "10.51.210.2"))
}

func (s *Stack) agentFor(t *testing.T, name string) *snmpAgentGroup {
	t.Helper()
	for device, group := range s.snmpAgents {
		if device.Name == name {
			return group
		}
	}
	t.Fatalf("no SNMP agent for %s", name)
	return nil
}

func macIndex(mac net.HardwareAddr) string {
	parts := make([]string, len(mac))
	for i, b := range mac {
		parts[i] = strconv.Itoa(int(b))
	}
	return strings.Join(parts, ".")
}

// fdbPortName answers what a manager answers: the dot1q FDB port of mac,
// resolved through dot1dBasePortIfIndex to the ifDescr of the interface.
func fdbPortName(t *testing.T, group *snmpAgentGroup, mac net.HardwareAddr) (string, bool) {
	t.Helper()
	agent := group.Get(placementCommunity)
	port, err := agent.HandleGet(dot1qTpFdbPortOID + "." + strconv.Itoa(placementDataVLAN) + "." + macIndex(mac))
	if err != nil || port == nil || port.Value == nil {
		return "", false
	}
	dot1d, err := agent.HandleGet(dot1dTpFdbPortOID + "." + macIndex(mac))
	if err != nil || dot1d == nil || dot1d.Value != port.Value {
		t.Fatalf("dot1dTpFdbPort = %v, dot1qTpFdbPort = %v: the two tables disagree", dot1d, port)
	}
	ifIndex, err := agent.HandleGet(dot1dBasePortIfIdx + "." + strconv.Itoa(port.Value.(int)))
	if err != nil || ifIndex == nil {
		return "bridge port " + strconv.Itoa(port.Value.(int)), true
	}
	descr, err := agent.HandleGet(ifDescrOID + "." + strconv.Itoa(ifIndex.Value.(int)))
	if err != nil || descr == nil {
		t.Fatalf("ifIndex %v has no ifDescr", ifIndex.Value)
	}
	switch name := descr.Value.(type) {
	case string:
		return name, true
	case []byte:
		return string(name), true
	default:
		t.Fatalf("ifDescr is %T", descr.Value)
		return "", false
	}
}

func TestPlacementPutsEachClientOnItsOwnPortAndNowhereElse(t *testing.T) {
	stack := placementStack(t)
	first, second := placementClient(1), placementClient(2)
	sendFrom(stack, first, "10.51.210.101")
	sendFrom(stack, second, "10.51.210.102")
	sendFrom(stack, first, "10.51.210.101")

	access := stack.agentFor(t, placementAccess)
	for mac, want := range map[string]string{
		first.String():  "GigabitEthernet1/0/43",
		second.String(): "GigabitEthernet1/0/44",
	} {
		hw, _ := net.ParseMAC(mac)
		got, ok := fdbPortName(t, access, hw)
		if !ok || got != want {
			t.Errorf("%s FDB port on %s = %q (found %v), want %q", mac, placementAccess, got, ok, want)
		}
		for _, other := range []string{placementCore, placementNeighbour} {
			if port, found := fdbPortName(t, stack.agentFor(t, other), hw); found {
				t.Errorf("%s reports %s on %q; only its own switch may", other, mac, port)
			}
		}
	}

	// A DHCP ACK used to inject the client into every forwarding device on
	// the attachment network at the authored constant port; the core authors one.
	stack.updateFDBTables(first, stack.devices)
	if port, found := fdbPortName(t, stack.agentFor(t, placementCore), first); found {
		t.Errorf("after a DHCP ACK %s reports %s on %q", placementCore, first, port)
	}

	clients := stack.GetObservedClients()
	placed := make(map[string]string, len(clients))
	for _, client := range clients {
		placed[client.MAC] = client.Device + " " + client.Interface
	}
	if placed[first.String()] != placementAccess+" GigabitEthernet1/0/43" ||
		placed[second.String()] != placementAccess+" GigabitEthernet1/0/44" {
		t.Errorf("GetObservedClients() placement = %v", placed)
	}
}

func TestPlacementHoldsAPinnedPortForItsClient(t *testing.T) {
	pinned := placementClient(9)
	stack := placementStack(t, config.AttachmentPin{
		MAC: strings.ToUpper(pinned.String()), Device: placementAccess, Interface: placementPinnedPort,
	})
	first, second, third := placementClient(1), placementClient(2), placementClient(3)
	for _, mac := range []net.HardwareAddr{first, second, third, pinned} {
		sendFrom(stack, mac, "10.51.210.100")
	}

	access := stack.agentFor(t, placementAccess)
	if got, _ := fdbPortName(t, access, pinned); got != placementPinnedPort {
		t.Errorf("pinned client FDB port = %q, want its pin %q", got, placementPinnedPort)
	}
	if port, found := fdbPortName(t, access, third); found {
		t.Errorf("third unpinned client took %q; the pool's only free port is pinned", port)
	}
}

func TestPlacementStartsOverWhenTheSessionStops(t *testing.T) {
	stack := placementStack(t)
	if err := stack.Start(); err != nil {
		t.Fatalf("Start() = %v", err)
	}
	first, second := placementClient(1), placementClient(2)
	sendFrom(stack, first, "10.51.210.101")
	stack.Stop()
	sendFrom(stack, second, "10.51.210.102")

	if got, _ := stack.fabric.placement.lookup(second.String()); got.Interface != "GigabitEthernet1/0/43" {
		t.Errorf("after a reset the next client landed on %q, want the pool's first port", got.Interface)
	}
}

func TestNetworkScopedAttachmentPlacesNoClient(t *testing.T) {
	stack := observedClientStack(t)
	stack.decodePacket(arpRequestFrame(clientMAC()))

	for _, client := range stack.GetObservedClients() {
		if client.Device != "" || client.Interface != "" {
			t.Errorf("client %s placed on %s %s without a pool", client.MAC, client.Device, client.Interface)
		}
	}
}
