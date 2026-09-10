package protocols

import (
	"bytes"
	"net"
	"testing"

	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/layers"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
	"github.com/MustardSeedNetworks/niac-go/internal/logging"
)

func hostEgressFixture(t *testing.T, destination string) (*Stack, *Packet) {
	t.Helper()
	device := config.Device{
		Name: "host", Type: "server", VLAN: 200,
		MACAddress:  mustParseMAC(t, "02:00:00:00:00:10"),
		IPAddresses: []net.IP{net.ParseIP("192.0.2.10")},
		Interfaces:  []config.Interface{{Name: "eth0", Address: "192.0.2.10/24"}},
		Routes:      []config.Route{{Destination: "0.0.0.0/0", Via: "eth0", NextHop: "192.0.2.1"}},
	}
	stack := NewStack(nil, &config.Config{Devices: []config.Device{device}}, logging.NewDebugConfig(0))
	stack.running.Store(true)
	t.Cleanup(stack.Stop)
	buffer := gopacket.NewSerializeBuffer()
	err := gopacket.SerializeLayers(
		buffer,
		gopacket.SerializeOptions{FixLengths: true, ComputeChecksums: true},
		&layers.Ethernet{
			SrcMAC:       device.MACAddress,
			DstMAC:       mustParseMAC(t, "02:00:00:00:00:99"),
			EthernetType: layers.EthernetTypeIPv4,
		},
		&layers.IPv4{
			Version:  4,
			TTL:      64,
			Protocol: layers.IPProtocolICMPv4,
			SrcIP:    device.IPAddresses[0],
			DstIP:    net.ParseIP(destination),
		},
		&layers.ICMPv4{TypeCode: layers.CreateICMPv4TypeCode(layers.ICMPv4TypeEchoReply, 0)},
		gopacket.Payload("test"),
	)
	if err != nil {
		t.Fatal(err)
	}
	return stack, &Packet{
		Buffer:        buffer.Bytes(),
		Length:        len(buffer.Bytes()),
		VLAN:          200,
		generatedHost: &stack.config.Devices[0],
	}
}

func TestHostEgressMaskRoute(t *testing.T) {
	for _, test := range []struct {
		name, destination, target string
		bits                      int
	}{
		{"broad-direct", "192.0.3.20", "192.0.3.20", 16},
		{"narrow-gateway", "192.0.2.20", "192.0.2.1", 30},
		{"host-only-gateway", "192.0.2.20", "192.0.2.1", 32},
		{"all-local", "203.0.113.10", "203.0.113.10", 0},
	} {
		t.Run(test.name, func(t *testing.T) {
			stack, packet := hostEgressFixture(t, test.destination)
			if err := stack.deviceStates[packet.generatedHost].SetInterfacePrefixFault(
				devicestate.InterfacePrefixFault{
					Interface:  "eth0",
					Type:       devicestate.FaultBadMask,
					PrefixBits: test.bits,
				},
			); err != nil {
				t.Fatal(err)
			}
			route, handled, err := stack.hostPacketRoute(packet)
			if err != nil || !handled || route.target.String() != test.target {
				t.Fatalf("route=%+v handled=%v err=%v", route, handled, err)
			}
		})
	}
}

func TestHostEgressUnmarkedAndHealthyPreserveBytes(t *testing.T) {
	stack, packet := hostEgressFixture(t, "192.0.3.20")
	before := bytes.Clone(packet.Buffer)
	if _, handled, err := stack.hostPacketRoute(packet); err != nil || handled {
		t.Fatalf("healthy handled=%v error=%v", handled, err)
	}
	if err := stack.deviceStates[packet.generatedHost].SetInterfacePrefixFault(
		devicestate.InterfacePrefixFault{Interface: "eth0", Type: devicestate.FaultBadMask, PrefixBits: 16},
	); err != nil {
		t.Fatal(err)
	}
	packet.Device = packet.generatedHost
	packet.generatedHost = nil
	if _, handled, err := stack.hostPacketRoute(packet); err != nil || handled {
		t.Fatalf("raw replay handled=%v error=%v", handled, err)
	}
	if !bytes.Equal(before, packet.Buffer) {
		t.Fatal("packet mutated")
	}
}

func TestHostEgressNoRouteOrInactiveInterface(t *testing.T) {
	for _, down := range []bool{false, true} {
		stack, packet := hostEgressFixture(t, "192.0.2.20")
		store := stack.deviceStates[packet.generatedHost]
		if err := store.SetInterfacePrefixFault(
			devicestate.InterfacePrefixFault{Interface: "eth0", Type: devicestate.FaultBadMask, PrefixBits: 32},
		); err != nil {
			t.Fatal(err)
		}
		if down {
			if err := store.SetInterfaceFault("eth0", devicestate.FaultLinkDown, 100); err != nil {
				t.Fatal(err)
			}
		} else {
			network := store.Snapshot().Network
			network.Routes = nil
			store.ReplaceNetwork(network)
		}
		if _, handled, err := stack.hostPacketRoute(packet); !handled || err == nil {
			t.Fatalf("invalid egress handled=%v err=%v", handled, err)
		}
	}
}
