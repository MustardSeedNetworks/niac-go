package protocols_test

import (
	"encoding/binary"
	"net"
	"testing"

	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/layers"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/logging"
	"github.com/MustardSeedNetworks/niac-go/internal/protocols"
)

// lldpFrame wraps a built LLDP payload in the Ethernet header the handler sends
// it under, so an independent decoder sees what the wire sees.
func lldpFrame(t *testing.T, payload []byte, srcMAC net.HardwareAddr) []byte {
	t.Helper()

	dstMAC, err := net.ParseMAC(protocols.LLDPMulticastMAC)
	if err != nil {
		t.Fatalf("parse LLDP multicast MAC: %v", err)
	}

	frame := make([]byte, 0, 14+len(payload))
	frame = append(frame, dstMAC...)
	frame = append(frame, srcMAC...)
	frame = binary.BigEndian.AppendUint16(frame, uint16(layers.EthernetTypeLinkLayerDiscovery))
	return append(frame, payload...)
}

// TestBuildLLDPFrameDecodesEndToEnd feeds a built advertisement to an
// independent decoder rather than checking each TLV against expected bytes.
//
// The per-TLV tests around it are the same shape that let a malformed CDP
// Address TLV ship (#1326): a TLV can be well-formed read on its own and still
// derail a decoder walking the chain in order, and only a decode of the whole
// frame notices. Everything after the derailing TLV is silently lost on the wire.
func TestBuildLLDPFrameDecodesEndToEnd(t *testing.T) {
	cfg := &config.Config{}
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := protocols.NewLLDPHandler(stack)

	srcMAC := net.HardwareAddr{0x00, 0x1A, 0x2B, 0x3C, 0x4D, 0x5E}
	device := &config.Device{
		Name:        "Switch-1",
		Type:        "switch",
		MACAddress:  srcMAC,
		IPAddresses: []net.IP{net.ParseIP("192.168.1.1")},
		Interfaces:  []config.Interface{{Name: "GigabitEthernet0/1"}},
	}

	packet := gopacket.NewPacket(
		lldpFrame(t, handler.BuildLLDPFrame(device), srcMAC),
		layers.LinkTypeEthernet,
		gopacket.Default,
	)
	if errLayer := packet.ErrorLayer(); errLayer != nil {
		t.Fatalf("decode failed: %v", errLayer.Error())
	}

	discovery, ok := packet.Layer(layers.LayerTypeLinkLayerDiscovery).(*layers.LinkLayerDiscovery)
	if !ok {
		t.Fatal("no LLDP layer decoded")
	}
	if got := string(discovery.PortID.ID); got != "GigabitEthernet0/1" {
		t.Errorf("PortID = %q, want %q", got, "GigabitEthernet0/1")
	}

	infoLayer := packet.Layer(layers.LayerTypeLinkLayerDiscoveryInfo)
	if infoLayer == nil {
		t.Fatal("no LLDP info layer decoded")
	}
	info, ok := infoLayer.(*layers.LinkLayerDiscoveryInfo)
	if !ok {
		t.Fatalf("unexpected layer type %T", infoLayer)
	}

	if info.SysName != "Switch-1" {
		t.Errorf("SysName = %q, want %q", info.SysName, "Switch-1")
	}

	// The management address is the TLV most like CDP's: a length-prefixed
	// address whose family is chosen by the emitter. Advertising an IPv4 address
	// under the IPv6 subtype is invisible to a per-field test.
	if got := info.MgmtAddress.Subtype; got != layers.IANAAddressFamilyIPV4 {
		t.Errorf("management address subtype = %d, want IPv4 (%d)",
			got, layers.IANAAddressFamilyIPV4)
	}
	if got := net.IP(info.MgmtAddress.Address); !got.Equal(net.ParseIP("192.168.1.1")) {
		t.Errorf("management address = %v, want 192.168.1.1", got)
	}
}
