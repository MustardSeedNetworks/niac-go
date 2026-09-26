//go:build linux && integration

package wiretest_test

import (
	"bytes"
	"net"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/layers"
	"github.com/gosnmp/gosnmp"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/converter"
	"github.com/MustardSeedNetworks/niac-go/internal/protocols"
	"github.com/MustardSeedNetworks/niac-go/internal/scenario"
)

// AP-2: a client plugged into a pool port is learned where the cable is. Two
// clients share the wire here: the test end itself, placed by its own SNMP
// traffic, and a second MAC that only ever sends one ARP request. Each must
// sit in the tester's switch's forwarding table at its own pool port, and no
// other device on the attachment network may report either of them.
const (
	oidDot1dTpFdbPort       = ".1.3.6.1.2.1.17.4.3.1.2"
	oidDot1qTpFdbPort       = ".1.3.6.1.2.1.17.7.1.2.2.1.2"
	oidDot1dBasePortIfIndex = ".1.3.6.1.2.1.17.1.4.1.2"
	oidSysName              = ".1.3.6.1.2.1.1.5.0"
)

// secondClient is a MAC with no kernel behind it: it sends only what a test
// writes for it, so its placement is decided by that frame alone.
func secondClient() net.HardwareAddr {
	return net.HardwareAddr{0x02, 0x00, 0x00, 0x00, 0xa2, 0x02}
}

func TestTwoClientsAreLearnedOnTheirOwnPoolPorts(t *testing.T) {
	authored, attachment := startPack(t, "hospital")
	pool := authored.Attachments[0].At
	access := dialDevice(t, authored, pool.Device)

	// The test end's first frames place it, so it takes the pool's first port.
	if _, err := access.Get([]string{oidSysName}); err != nil {
		t.Fatalf("GET sysName on %s: %v", pool.Device, err)
	}
	testEnd := clientMAC(t)
	second := secondClient()
	announce(t, second, attachment)

	vlan := portVLAN(t, deviceNamed(t, authored, pool.Device), pool.Ports[0])
	want := map[string]string{testEnd.String(): pool.Ports[0], second.String(): pool.Ports[1]}
	for _, mac := range []net.HardwareAddr{testEnd, second} {
		got := awaitFDBPort(t, access, vlan, mac)
		if got != want[mac.String()] {
			t.Errorf("%s dot1qTpFdbPort on %s resolves to %q, want %q", mac, pool.Device, got, want[mac.String()])
		}
		t.Logf("%s: %s learned on %s", pool.Device, mac, got)
	}

	checked := 0
	for _, device := range attachment.onNetwork {
		if device.Name == pool.Device || !config.SNMPv2Enabled(device.SNMPConfig) {
			continue
		}
		checked++
		other := dialDevice(t, authored, device.Name)
		for _, mac := range []net.HardwareAddr{testEnd, second} {
			for _, column := range []string{oidDot1dTpFdbPort, oidDot1qTpFdbPort} {
				if row := fdbRow(t, other, column, mac); row != "" {
					t.Errorf("%s reports %s at %s; only %s may", device.Name, mac, row, pool.Device)
				}
			}
		}
	}
	if checked == 0 {
		t.Fatal("no other SNMP device on the attachment network, so \"nowhere else\" asserted nothing")
	}
	t.Logf("%d other devices on %s report neither client", checked, attachment.network)
}

// LLDP is link-local: a tester hears the switch at the other end of its cable
// and nothing further away. Before AP-2 the speakers were every device with an
// interface on the attachment network, which on a pack is the core switch three
// tiers up and never the access switch the tester is plugged into. Both clients
// share this wire, so each hears exactly the neighbours the wire carries: that
// must be one, their switch, naming the first-placed client's port.
func TestPoolClientsHearOnlyTheirOwnSwitch(t *testing.T) {
	authored, attachment := startPack(t, "hospital")
	pool := authored.Attachments[0].At
	access := deviceNamed(t, authored, pool.Device)

	handle := openClient(t)
	packets := gopacket.NewPacketSource(handle, handle.LinkType()).Packets()
	if _, err := dialDevice(t, authored, pool.Device).Get([]string{oidSysName}); err != nil {
		t.Fatalf("GET sysName on %s: %v", pool.Device, err)
	}
	second := secondClient()
	announce(t, second, attachment)

	// LLDP and CDP beacon every 15 s: two intervals and slack.
	want := advertisedName(access) + " " + pool.Ports[0]
	neighbours := map[string]int{}
	deadline := time.After(35 * time.Second)
	for waiting := true; waiting; {
		select {
		case packet := <-packets:
			ethernet, ok := packet.Layer(layers.LayerTypeEthernet).(*layers.Ethernet)
			if !ok || !discoveryDestination(ethernet.DstMAC) {
				continue
			}
			if !bytes.Equal(ethernet.SrcMAC, access.MACAddress) {
				t.Errorf("advertisement to %s from %s; only %s (%s) is at the tester's cable",
					ethernet.DstMAC, ethernet.SrcMAC, access.Name, access.MACAddress)
			}
			lldp, isLLDP := packet.Layer(layers.LayerTypeLinkLayerDiscovery).(*layers.LinkLayerDiscovery)
			info, hasInfo := packet.Layer(layers.LayerTypeLinkLayerDiscoveryInfo).(*layers.LinkLayerDiscoveryInfo)
			if isLLDP && hasInfo {
				neighbours[info.SysName+" "+string(lldp.PortID.ID)]++
			}
		case <-deadline:
			waiting = false
		}
	}
	if len(neighbours) != 1 || neighbours[want] == 0 {
		t.Fatalf("LLDP neighbours on the wire = %v, want exactly one: %q", neighbours, want)
	}
	t.Logf("both clients hear one LLDP neighbour, %q (%d frames)", want, neighbours[want])
}

// Each client leases from the scope of the network its own port lands it on,
// with that network's gateway as its first hop.
func TestPoolClientsLeaseFromTheirPortsScope(t *testing.T) {
	authored, attachment := startPack(t, "hospital")
	pool := authored.Attachments[0].At
	handle := openClient(t)

	clients := []net.HardwareAddr{clientMAC(t), secondClient()}
	for index, mac := range clients {
		port := attachment.ports[index]
		_, scope := attachmentScope(t, attachment.topology, port.Network)
		xid := 0x9e57c0e0 + uint32(index)
		offer := dhcpExchange(t, handle, mac, xid, layers.DHCPMsgTypeDiscover, nil)
		offered := append(net.IP(nil), offer.YourClientIP...)
		ack := dhcpExchange(t, handle, mac, xid, layers.DHCPMsgTypeRequest, offered)

		if !addrWithin(ack.YourClientIP, scope.Start.AsSlice(), scope.End.AsSlice()) {
			t.Errorf("%s on %s leased %s, outside %s's scope %s-%s",
				mac, port.Interface, ack.YourClientIP, port.Network, scope.Start, scope.End)
		}
		if router := dhcpOptionIP(ack, layers.DHCPOptRouter); !router.Equal(scope.Router.AsSlice()) {
			t.Errorf("%s on %s has first hop %s, want %s's gateway %s",
				mac, port.Interface, router, port.Network, scope.Router)
		}
		t.Logf("%s: %s lease %s via %s (%s)", mac, port.Interface, ack.YourClientIP, scope.Router, port.Network)
	}

	access := dialDevice(t, authored, pool.Device)
	vlan := portVLAN(t, deviceNamed(t, authored, pool.Device), pool.Ports[0])
	for index, mac := range clients {
		if got := awaitFDBPort(t, access, vlan, mac); got != attachment.ports[index].Interface {
			t.Errorf("%s leased as the client on %s but %s learned it on %s",
				mac, attachment.ports[index].Interface, pool.Device, got)
		}
	}
}

// A pinned MAC lands on its pin even when an unpinned client arrives first.
// The pin takes the pool's first port, the one the test end would otherwise
// be given.
func TestAPinnedClientLandsOnItsPin(t *testing.T) {
	pool := packPool(t, "hospital")
	pinned := net.HardwareAddr{0x02, 0x00, 0x00, 0x00, 0xa2, 0x09}
	authored, attachment := startPack(t, "hospital", converter.AttachmentPin{
		MAC: pinned.String(), Device: pool.Device, Interface: pool.Ports[0],
	})
	access := dialDevice(t, authored, pool.Device)

	if _, err := access.Get([]string{oidSysName}); err != nil {
		t.Fatalf("GET sysName on %s: %v", pool.Device, err)
	}
	announce(t, pinned, attachment)

	vlan := portVLAN(t, deviceNamed(t, authored, pool.Device), pool.Ports[0])
	testEnd := clientMAC(t)
	for mac, want := range map[string]string{pinned.String(): pool.Ports[0], testEnd.String(): pool.Ports[1]} {
		hw, _ := net.ParseMAC(mac)
		if got := awaitFDBPort(t, access, vlan, hw); got != want {
			t.Errorf("%s landed on %s, want %s", mac, got, want)
		}
		t.Logf("%s: %s on %s", pool.Device, mac, want)
	}
}

// packPool is the tester attachment's pool a pack generates, read before the
// pack is started so a pin can name one of its ports.
func packPool(t *testing.T, id string) *config.AttachmentPort {
	t.Helper()
	for _, pack := range scenario.Packs() {
		if pack.ID == id {
			result, err := scenario.Generate(pack.Request)
			if err != nil {
				t.Fatalf("scenario.Generate(%s): %v", id, err)
			}
			return result.Config.Attachments[0].At
		}
	}
	t.Fatalf("no pack with id %q", id)
	return nil
}

func discoveryDestination(mac net.HardwareAddr) bool {
	for _, group := range []string{
		protocols.LLDPMulticastMAC, protocols.STPMulticastMAC, protocols.CDPMulticastMAC,
		protocols.EDPMulticastMAC, protocols.FDPMulticastMAC,
	} {
		if strings.EqualFold(mac.String(), group) {
			return true
		}
	}
	return false
}

// announce sends one ARP request from mac for the attachment gateway, the one
// frame a client that never DHCPs or advertises discovery is guaranteed to send.
func announce(t *testing.T, mac net.HardwareAddr, attachment packAttachment) {
	t.Helper()
	sender := attachment.client.Addr().Next().AsSlice()
	frame := serialize(t,
		&layers.Ethernet{
			SrcMAC: mac, DstMAC: layers.EthernetBroadcast, EthernetType: layers.EthernetTypeARP,
		},
		&layers.ARP{
			AddrType: layers.LinkTypeEthernet, Protocol: layers.EthernetTypeIPv4,
			HwAddressSize: 6, ProtAddressSize: 4, Operation: layers.ARPRequest,
			SourceHwAddress: mac, SourceProtAddress: sender,
			DstHwAddress: make([]byte, 6), DstProtAddress: attachment.gateway.AsSlice(),
		},
	)
	if err := openClient(t).WritePacketData(frame); err != nil {
		t.Fatalf("sending ARP from %s: %v", mac, err)
	}
}

func portVLAN(t *testing.T, device *config.Device, port string) int {
	t.Helper()
	for _, iface := range device.Interfaces {
		if iface.Name == port && len(iface.VLANs) == 1 {
			return iface.VLANs[0]
		}
	}
	t.Fatalf("%s %s carries no single access VLAN", device.Name, port)
	return 0
}

func macOID(mac net.HardwareAddr) string {
	parts := make([]string, len(mac))
	for i, b := range mac {
		parts[i] = strconv.Itoa(int(b))
	}
	return strings.Join(parts, ".")
}

// awaitFDBPort polls dot1qTpFdbPort for mac and resolves the bridge port to
// the interface name a manager would render, through dot1dBasePortIfIndex and
// ifDescr. The frame that places a client races the poll, hence the retry.
func awaitFDBPort(t *testing.T, client *gosnmp.GoSNMP, vlan int, mac net.HardwareAddr) string {
	t.Helper()
	oid := oidDot1qTpFdbPort + "." + strconv.Itoa(vlan) + "." + macOID(mac)
	deadline := time.Now().Add(5 * time.Second)
	for {
		result, err := client.Get([]string{oid})
		if err != nil {
			t.Fatalf("GET %s on %s: %v", oid, client.Target, err)
		}
		if variable := result.Variables[0]; variable.Type == gosnmp.Integer {
			bridgePort := gosnmp.ToBigInt(variable.Value).String()
			return interfaceBehind(t, client, bridgePort)
		}
		if time.Now().After(deadline) {
			t.Fatalf("%s never learned %s on VLAN %d", client.Target, mac, vlan)
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func interfaceBehind(t *testing.T, client *gosnmp.GoSNMP, bridgePort string) string {
	t.Helper()
	result, err := client.Get([]string{oidDot1dBasePortIfIndex + "." + bridgePort})
	if err != nil || result.Variables[0].Type != gosnmp.Integer {
		t.Fatalf("bridge port %s on %s maps to no ifIndex: %v", bridgePort, client.Target, err)
	}
	ifIndex := gosnmp.ToBigInt(result.Variables[0].Value).String()
	descr, err := client.Get([]string{oidIfDescr + "." + ifIndex})
	if err != nil {
		t.Fatalf("GET ifDescr.%s on %s: %v", ifIndex, client.Target, err)
	}
	octets, ok := descr.Variables[0].Value.([]byte)
	if !ok {
		t.Fatalf("ifDescr.%s on %s is %T", ifIndex, client.Target, descr.Variables[0].Value)
	}
	return string(octets)
}

// fdbRow returns the OID of any row of column, under any VLAN, keyed by mac.
func fdbRow(t *testing.T, client *gosnmp.GoSNMP, column string, mac net.HardwareAddr) string {
	t.Helper()
	rows, err := client.BulkWalkAll(column)
	if err != nil {
		t.Fatalf("walking %s on %s: %v", column, client.Target, err)
	}
	suffix := "." + macOID(mac)
	for _, row := range rows {
		if strings.HasSuffix(row.Name, suffix) {
			return row.Name
		}
	}
	return ""
}
