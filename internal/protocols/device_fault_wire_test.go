package protocols

import (
	"net"
	"testing"

	"github.com/gopacket/gopacket/layers"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
	"github.com/MustardSeedNetworks/niac-go/internal/logging"
)

// The positive case comes first in every test below: a suppressed answer is
// only evidence of the fault if the same request is answered without it.
func TestDHCPNoOfferFaultSuppressesTheOffer(t *testing.T) {
	stack, handler, device := newFaultedDHCPHandler(t)
	discover := dhcpDiscover(net.HardwareAddr{0x02, 0, 0, 0, 0, 0x11})

	handler.handleDHCPDiscover(discover, device, 1, 0)
	offer := drainDHCP(t, stack)
	if got := dhcpMsgType(dhcpOf(t, offer)); got != DHCPOffer {
		t.Fatalf("healthy server message type = %d, want DHCPOFFER", got)
	}

	if err := stack.SetDeviceFault(
		device.Name, devicestate.FaultDHCPNoOffer, 1,
	); err != nil {
		t.Fatalf("SetDeviceFault() error = %v", err)
	}

	handler.handleDHCPDiscover(
		dhcpDiscover(net.HardwareAddr{0x02, 0, 0, 0, 0, 0x12}), device, 2, 0)
	select {
	case pkt := <-stack.sendQueue:
		t.Fatalf("faulted server still answered: %#v", dhcpOf(t, pkt))
	default:
	}

	if err := stack.SetDeviceFault(
		device.Name, devicestate.FaultDHCPNoOffer, 0,
	); err != nil {
		t.Fatalf("SetDeviceFault(clear) error = %v", err)
	}
	handler.handleDHCPDiscover(
		dhcpDiscover(net.HardwareAddr{0x02, 0, 0, 0, 0, 0x13}), device, 3, 0)
	if got := dhcpMsgType(dhcpOf(t, drainDHCP(t, stack))); got != DHCPOffer {
		t.Fatalf("cleared fault message type = %d, want DHCPOFFER", got)
	}
}

func TestDNSFaultsChangeTheAnswer(t *testing.T) {
	stack, handler, device := newFaultedDNSHandler(t)
	query := &layers.DNS{
		ID: 0x4242, RD: true,
		Questions: []layers.DNSQuestion{{
			Name: []byte("host.example."), Type: layers.DNSTypeA, Class: layers.DNSClassIN,
		}},
	}

	healthy := handler.buildDNSResponse(query, device, 0, 1)
	if healthy.ResponseCode != layers.DNSResponseCodeNoErr || len(healthy.Answers) != 1 {
		t.Fatalf("healthy response = rcode %v, %d answers",
			healthy.ResponseCode, len(healthy.Answers))
	}

	if err := stack.SetDeviceFault(
		device.Name, devicestate.FaultDNSNXDomain, 1,
	); err != nil {
		t.Fatalf("SetDeviceFault(nxdomain) error = %v", err)
	}

	faulted := handler.buildDNSResponse(query, device, 0, 2)
	if faulted.ResponseCode != layers.DNSResponseCodeNXDomain {
		t.Fatalf("faulted rcode = %v, want NXDomain", faulted.ResponseCode)
	}
	if len(faulted.Answers) != 0 {
		t.Fatalf("faulted answers = %#v, want none", faulted.Answers)
	}
	if !faulted.AA || faulted.ID != query.ID || len(faulted.Questions) != 1 {
		t.Fatalf("faulted response stopped looking like this server: %#v", faulted)
	}

	if handler.dnsQuerySilenced(device) {
		t.Fatal("an NXDOMAIN fault also silenced the server")
	}
	if err := stack.SetDeviceFault(
		device.Name, devicestate.FaultDNSTimeout, 1,
	); err != nil {
		t.Fatalf("SetDeviceFault(timeout) error = %v", err)
	}
	if !handler.dnsQuerySilenced(device) {
		t.Fatal("a timeout fault did not silence the server")
	}
}

func newFaultedDHCPHandler(t *testing.T) (*Stack, *DHCPHandler, *config.Device) {
	t.Helper()

	cfg := &config.Config{Devices: []config.Device{{
		Name:        "dhcp-server",
		MACAddress:  net.HardwareAddr{0x02, 0x00, 0x14, 0x03, 0x00, 0x01},
		IPAddresses: []net.IP{net.ParseIP("10.20.200.2")},
		Interfaces:  []config.Interface{{Name: "Gi0/1", Address: "10.20.200.2/24", Speed: 100}},
		DHCPConfig: &config.DHCPConfig{
			PoolStart: net.ParseIP("10.20.200.100"), PoolEnd: net.ParseIP("10.20.200.199"),
		},
		SNMPConfig: config.SNMPConfig{Community: "public"},
	}}}
	stack := NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := NewDHCPHandler(stack)
	handler.SetPool(net.ParseIP("10.20.200.100"), net.ParseIP("10.20.200.199"))
	handler.SetServerConfig(
		net.ParseIP("10.20.200.2"), net.ParseIP("10.20.200.1"), nil, "demo.lab")
	handler.subnetMask = net.ParseIP("255.255.255.0")

	return stack, handler, &cfg.Devices[0]
}

func newFaultedDNSHandler(t *testing.T) (*Stack, *DNSHandler, *config.Device) {
	t.Helper()

	cfg := &config.Config{Devices: []config.Device{{
		Name:        "dns-server",
		MACAddress:  net.HardwareAddr{0x02, 0x00, 0x14, 0x03, 0x00, 0x02},
		IPAddresses: []net.IP{net.ParseIP("10.20.200.3")},
		Interfaces:  []config.Interface{{Name: "Gi0/1", Address: "10.20.200.3/24", Speed: 100}},
		DNSConfig: &config.DNSConfig{ForwardRecords: []config.DNSRecord{
			{Name: "host.example.", IP: net.ParseIP("10.20.200.50"), TTL: 300},
		}},
		SNMPConfig: config.SNMPConfig{Community: "public"},
	}}}
	stack := NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := NewDNSHandler(stack)
	device := &cfg.Devices[0]
	handler.LoadDeviceDNSConfig(device)

	return stack, handler, device
}

func dhcpDiscover(clientMAC net.HardwareAddr) *dhcpPacketInfo {
	return &dhcpPacketInfo{
		dhcp: &layers.DHCPv4{
			Operation:    layers.DHCPOpRequest,
			Xid:          0x12345678,
			ClientHWAddr: clientMAC,
			Options: []layers.DHCPOption{
				{Type: layers.DHCPOptMessageType, Length: 1, Data: []byte{DHCPDiscover}},
			},
		},
		messageType: DHCPDiscover,
	}
}
