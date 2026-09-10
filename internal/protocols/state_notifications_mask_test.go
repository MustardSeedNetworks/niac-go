package protocols

import (
	"net"
	"net/netip"
	"testing"

	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/layers"

	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
)

func TestNotificationHostMaskFirstHop(t *testing.T) {
	for _, test := range []struct {
		name, destination, target string
		bits                      int
	}{
		{"healthy", "192.0.2.20", "192.0.2.20", -1},
		{"narrow", "192.0.2.20", "192.0.2.1", 30},
		{"broad", "192.0.3.20", "192.0.3.20", 16},
	} {
		t.Run(test.name, func(t *testing.T) {
			stack, packet := hostEgressFixture(t, test.destination)
			if test.bits >= 0 {
				setNotificationMask(t, stack, packet, test.bits)
			}
			if err := stack.notifications.sender.Send(
				packet.generatedHost, 200, net.JoinHostPort(test.destination, "514"), 514, []byte("event"),
			); err != nil {
				t.Fatal(err)
			}
			assertNotificationProbe(t, receiveRoutedReply(t, stack), "192.0.2.10", test.target)
		})
	}
}

func setNotificationMask(t *testing.T, stack *Stack, packet *Packet, bits int) {
	t.Helper()
	if err := stack.deviceStates[packet.generatedHost].SetInterfacePrefixFault(
		devicestate.InterfacePrefixFault{Interface: "eth0", Type: devicestate.FaultBadMask, PrefixBits: bits},
	); err != nil {
		t.Fatal(err)
	}
}

func assertNotificationProbe(t *testing.T, packet *Packet, source, target string) {
	t.Helper()
	decoded := gopacket.NewPacket(packet.Buffer, layers.LayerTypeEthernet, gopacket.Default)
	arp, _ := decoded.Layer(layers.LayerTypeARP).(*layers.ARP)
	if arp == nil || !net.IP(arp.SourceProtAddress).Equal(net.ParseIP(source)) ||
		!net.IP(arp.DstProtAddress).Equal(net.ParseIP(target)) || packet.VLAN != 200 {
		t.Fatalf("notification probe = %+v VLAN=%d, want %s -> %s VLAN 200", arp, packet.VLAN, source, target)
	}
}

func TestNotificationHostMaskNoDirectFallback(t *testing.T) {
	stack, packet := hostEgressFixture(t, "192.0.2.20")
	setNotificationMask(t, stack, packet, 32)
	store := stack.deviceStates[packet.generatedHost]
	network := store.Snapshot().Network
	network.Routes = nil
	store.ReplaceNetwork(network)
	if err := stack.notifications.sender.Send(
		packet.generatedHost, 200, "192.0.2.20:514", 514, []byte("event"),
	); err == nil {
		t.Fatal("faulted /32 notification accepted without route")
	}
	select {
	case <-stack.sendQueue:
		t.Fatal("notification escaped missing route")
	default:
	}
}

func TestNotificationPendingMaskUsesCurrentIntent(t *testing.T) {
	for _, change := range []string{"clear", "address", "activate"} {
		t.Run(change, func(t *testing.T) {
			stack, packet := hostEgressFixture(t, "192.0.2.20")
			store := stack.deviceStates[packet.generatedHost]
			if change != "activate" {
				setNotificationMask(t, stack, packet, 30)
			}
			sender := stack.notifications.sender.(*stackDatagramSender)
			if err := sender.Send(packet.generatedHost, 200, "192.0.2.20:514", 514, []byte("event")); err != nil {
				t.Fatal(err)
			}
			receiveRoutedReply(t, stack)
			source, previous, target := "192.0.2.10", "192.0.2.1", "192.0.2.20"
			switch change {
			case "clear":
				store.ClearAllFaults()
			case "address":
				source, target = "192.0.2.21", "192.0.2.20"
				if err := store.UpdateInterface(
					"eth0",
					func(iface devicestate.Interface) (devicestate.Interface, error) {
						iface.Address = netip.MustParsePrefix("192.0.2.21/24")
						return iface, nil
					},
				); err != nil {
					t.Fatal(err)
				}
			case "activate":
				previous, target = "192.0.2.20", "192.0.2.1"
				setNotificationMask(t, stack, packet, 30)
			}
			sender.observeNeighbor(201, netip.MustParseAddr(previous), mustParseMAC(t, "02:00:00:00:00:99"))
			select {
			case <-stack.sendQueue:
				t.Fatal("notification resolved on unrelated VLAN")
			default:
			}
			sender.observeNeighbor(200, netip.MustParseAddr(previous), mustParseMAC(t, "02:00:00:00:00:99"))
			assertNotificationProbe(t, receiveRoutedReply(t, stack), source, target)
			sender.observeNeighbor(200, netip.MustParseAddr(target), mustParseMAC(t, "02:00:00:00:00:98"))
			assertNotificationDatagram(t, receiveRoutedReply(t, stack), source)
		})
	}
}

func assertNotificationDatagram(t *testing.T, packet *Packet, source string) {
	t.Helper()
	decoded := gopacket.NewPacket(packet.Buffer, layers.LayerTypeEthernet, gopacket.Default)
	ip, _ := decoded.Layer(layers.LayerTypeIPv4).(*layers.IPv4)
	udp, _ := decoded.Layer(layers.LayerTypeUDP).(*layers.UDP)
	if ip == nil || udp == nil || !ip.SrcIP.Equal(net.ParseIP(source)) || string(udp.Payload) != "event" ||
		udp.SrcPort != 514 || udp.DstPort != 514 || packet.VLAN != 200 {
		t.Fatalf("notification IP=%+v UDP=%+v VLAN=%d", ip, udp, packet.VLAN)
	}
}
