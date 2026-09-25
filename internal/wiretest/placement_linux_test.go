//go:build linux && integration

package wiretest_test

import (
	"net"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gopacket/gopacket/layers"
	"github.com/gosnmp/gosnmp"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
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

func TestTwoClientsAreLearnedOnTheirOwnPoolPorts(t *testing.T) {
	authored, attachment := startPack(t, "hospital")
	pool := authored.Attachments[0].At
	access := dialDevice(t, authored, pool.Device)

	// The test end's first frames place it, so it takes the pool's first port.
	if _, err := access.Get([]string{oidSysName}); err != nil {
		t.Fatalf("GET sysName on %s: %v", pool.Device, err)
	}
	testEnd := clientMAC(t)
	second := net.HardwareAddr{0x02, 0x00, 0x00, 0x00, 0xa2, 0x02}
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
