package protocols

import (
	"errors"
	"net"
	"testing"
	"time"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/logging"
)

// The observed-client table answers "who is attached to this scenario right
// now". Learning used to happen only on a DHCP ACK or an inbound discovery
// frame, so a tester with a static address that never advertises LLDP/CDP was
// invisible (AP-1b, niac attachment-pool plan D4). These tests drive real
// frames through decodePacket, which is the one funnel every received frame
// passes, rather than poking the table directly.

const (
	testClientIP = "192.0.2.50"
	testGatewayI = "192.0.2.1"
)

func clientMAC() net.HardwareAddr { return net.HardwareAddr{0x02, 0x00, 0x00, 0x00, 0x00, 0x11} }
func deviceMAC() net.HardwareAddr { return net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, 0x55} }

func observedClientStack(t *testing.T) *Stack {
	t.Helper()
	cfg := &config.Config{
		Devices: []config.Device{{
			Name:        "sw1",
			Type:        "switch",
			MACAddress:  deviceMAC(),
			IPAddresses: []net.IP{net.ParseIP(testGatewayI)},
		}},
	}
	return NewStack(nil, cfg, logging.NewDebugConfig(0))
}

func arpRequestFrame(src net.HardwareAddr) *Packet {
	frame := make([]byte, etherHeaderSize+28)
	copy(frame[0:6], net.HardwareAddr{0xff, 0xff, 0xff, 0xff, 0xff, 0xff})
	copy(frame[6:12], src)
	frame[12], frame[13] = 0x08, 0x06 // ARP
	arp := frame[etherHeaderSize:]
	arp[0], arp[1] = 0x00, 0x01 // Ethernet
	arp[2], arp[3] = 0x08, 0x00 // IPv4
	arp[4], arp[5] = arpHWAddressSize, arpProtAddressSize
	arp[6], arp[7] = 0x00, 0x01 // request
	copy(arp[8:14], src)
	copy(arp[14:18], net.ParseIP(testClientIP).To4())
	copy(arp[24:28], net.ParseIP(testGatewayI).To4())

	pkt, _ := ParsePacket(frame, 1)
	return pkt
}

func ipv4Frame(src, dst net.HardwareAddr, srcIP, dstIP string) *Packet {
	frame := make([]byte, etherHeaderSize+20)
	copy(frame[0:6], dst)
	copy(frame[6:12], src)
	frame[12], frame[13] = 0x08, 0x00 // IPv4
	ip := frame[etherHeaderSize:]
	ip[0] = 0x45
	ip[9] = 1 // ICMP
	copy(ip[12:16], net.ParseIP(srcIP).To4())
	copy(ip[16:20], net.ParseIP(dstIP).To4())

	pkt, _ := ParsePacket(frame, 2)
	return pkt
}

func TestObservedClientsRecordAnARPOnlyClient(t *testing.T) {
	stack := observedClientStack(t)
	stack.decodePacket(arpRequestFrame(clientMAC()))

	clients := stack.GetObservedClients()
	if len(clients) != 1 {
		t.Fatalf("GetObservedClients() = %d entries, want 1", len(clients))
	}
	if clients[0].MAC != clientMAC().String() {
		t.Errorf("MAC = %q, want %q", clients[0].MAC, clientMAC().String())
	}
	if clients[0].IP != testClientIP {
		t.Errorf("IP = %q, want %q", clients[0].IP, testClientIP)
	}
	if clients[0].Frames != 1 {
		t.Errorf("Frames = %d, want 1", clients[0].Frames)
	}
}

func TestObservedClientsRecordAStaticIPClientThatNeverDHCPs(t *testing.T) {
	stack := observedClientStack(t)
	stack.decodePacket(ipv4Frame(clientMAC(), deviceMAC(), testClientIP, testGatewayI))

	clients := stack.GetObservedClients()
	if len(clients) != 1 {
		t.Fatalf("GetObservedClients() = %d entries, want 1", len(clients))
	}
	if clients[0].IP != testClientIP {
		t.Errorf("IP = %q, want %q", clients[0].IP, testClientIP)
	}
}

func TestObservedClientsIgnoreTheSimulationsOwnDevices(t *testing.T) {
	stack := observedClientStack(t)
	// libpcap on Linux reports the sim's own outbound frames on the same
	// handle, so without this filter the table fills with the scenario itself.
	stack.decodePacket(ipv4Frame(deviceMAC(), clientMAC(), testGatewayI, testClientIP))

	if clients := stack.GetObservedClients(); len(clients) != 0 {
		t.Fatalf("GetObservedClients() = %+v, want no entries", clients)
	}
}

func TestObservedClientsIgnoreGroupAndZeroSourceMACs(t *testing.T) {
	stack := observedClientStack(t)
	multicast := net.HardwareAddr{0x01, 0x00, 0x5e, 0x00, 0x00, 0x01}
	stack.decodePacket(ipv4Frame(multicast, deviceMAC(), testClientIP, testGatewayI))
	stack.decodePacket(ipv4Frame(net.HardwareAddr{0, 0, 0, 0, 0, 0}, deviceMAC(), testClientIP, testGatewayI))

	if clients := stack.GetObservedClients(); len(clients) != 0 {
		t.Fatalf("GetObservedClients() = %+v, want no entries", clients)
	}
}

func TestObservedClientsLeaveTheIPEmptyUntilTheClientClaimsOne(t *testing.T) {
	stack := observedClientStack(t)
	// A DHCP DISCOVER sources 0.0.0.0: the client is attached but has no
	// address yet, and recording 0.0.0.0 would read as one.
	stack.decodePacket(ipv4Frame(clientMAC(), deviceMAC(), "0.0.0.0", "255.255.255.255"))

	clients := stack.GetObservedClients()
	if len(clients) != 1 {
		t.Fatalf("GetObservedClients() = %d entries, want 1", len(clients))
	}
	if clients[0].IP != "" {
		t.Errorf("IP = %q, want empty", clients[0].IP)
	}

	stack.decodePacket(arpRequestFrame(clientMAC()))
	clients = stack.GetObservedClients()
	if len(clients) != 1 {
		t.Fatalf("after the lease, GetObservedClients() = %d entries, want 1", len(clients))
	}
	if clients[0].IP != testClientIP {
		t.Errorf("IP = %q, want %q", clients[0].IP, testClientIP)
	}
	if clients[0].Frames != 2 {
		t.Errorf("Frames = %d, want 2", clients[0].Frames)
	}
}

func TestObservedClientsAgeOut(t *testing.T) {
	stack := observedClientStack(t)
	base := time.Date(2026, 9, 17, 12, 0, 0, 0, time.UTC)
	now := base
	stack.observedClients.now = func() time.Time { return now }

	stack.decodePacket(arpRequestFrame(clientMAC()))

	now = base.Add(observedClientTTL - time.Second)
	stack.observedClients.cleanupExpired()
	if len(stack.GetObservedClients()) != 1 {
		t.Fatal("entry aged out before its TTL elapsed")
	}

	now = base.Add(observedClientTTL + time.Second)
	stack.observedClients.cleanupExpired()
	if clients := stack.GetObservedClients(); len(clients) != 0 {
		t.Fatalf("GetObservedClients() = %+v after the TTL elapsed, want no entries", clients)
	}
}

// idleTransport stands in for the capture handle a running session owns. It
// reads nothing, so the receive thread makes no progress and Stop is the only
// thing that ends it.
var errIdleTransport = errors.New("idle transport: nothing to read")

type idleTransport struct{}

func (idleTransport) ReadPacket([]byte) ([]byte, error) {
	time.Sleep(time.Millisecond)
	return nil, errIdleTransport
}
func (idleTransport) SendPacket([]byte) error { return nil }
func (idleTransport) SetFilter(string) error  { return nil }
func (idleTransport) Filter() string          { return "" }

func TestObservedClientsAreClearedWhenTheSessionStops(t *testing.T) {
	cfg := &config.Config{Devices: []config.Device{{
		Name: "sw1", Type: "switch", MACAddress: deviceMAC(),
		IPAddresses: []net.IP{net.ParseIP(testGatewayI)},
	}}}
	stack := NewStackWithTransport(idleTransport{}, cfg, logging.NewDebugConfig(0))
	if err := stack.Start(); err != nil {
		t.Fatalf("Start() = %v", err)
	}
	stack.decodePacket(arpRequestFrame(clientMAC()))
	if len(stack.GetObservedClients()) != 1 {
		t.Fatal("client was not recorded while the session ran")
	}

	stack.Stop()

	if clients := stack.GetObservedClients(); len(clients) != 0 {
		t.Fatalf("GetObservedClients() = %+v after Stop, want no entries", clients)
	}
}
