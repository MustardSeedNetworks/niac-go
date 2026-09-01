package protocols_test

import (
	"bytes"
	"encoding/binary"
	"net"
	"testing"

	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/layers"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/logging"
	"github.com/MustardSeedNetworks/niac-go/internal/protocols"
)

// fdpFrame wraps a built FDP payload in the 802.3 header the handler sends it
// under, so an independent decoder sees what the wire sees. FDP is
// LLC/SNAP-encapsulated, so the length/type field carries the payload length
// rather than an EtherType.
func fdpFrame(t *testing.T, payload []byte, srcMAC net.HardwareAddr) []byte {
	t.Helper()

	dstMAC, err := net.ParseMAC(protocols.FDPMulticastMAC)
	if err != nil {
		t.Fatalf("parse FDP multicast MAC: %v", err)
	}

	frame := make([]byte, 0, 14+len(payload))
	frame = append(frame, dstMAC...)
	frame = append(frame, srcMAC...)
	frame = binary.BigEndian.AppendUint16(frame, uint16(len(payload)))
	return append(frame, payload...)
}

// fdpTLV is one decoded type/length/value triple.
type fdpTLV struct {
	typ   uint16
	value []byte
}

// walkFDPTLVs decodes the TLV chain the way a collector does — in order, taking
// each TLV's length as the step to the next one — rather than checking TLVs at
// known offsets. A TLV whose declared length disagrees with its real size
// derails every TLV after it, and that is only visible to a walk.
func walkFDPTLVs(t *testing.T, payload []byte) []fdpTLV {
	t.Helper()

	// Version(1) + TTL(1) + Checksum(2).
	const headerSize = 4
	if len(payload) < headerSize {
		t.Fatalf("payload is %d bytes, shorter than the %d-byte FDP header", len(payload), headerSize)
	}

	var tlvs []fdpTLV
	for offset := headerSize; offset < len(payload); {
		if remaining := len(payload) - offset; remaining < 4 {
			t.Fatalf("TLV %d at offset %d: %d trailing bytes, too few for a 4-byte TLV header",
				len(tlvs), offset, remaining)
		}

		typ := binary.BigEndian.Uint16(payload[offset : offset+2])
		length := int(binary.BigEndian.Uint16(payload[offset+2 : offset+4]))
		if length < 4 {
			t.Fatalf("TLV %d (type %#x) declares length %d, shorter than its own 4-byte header",
				len(tlvs), typ, length)
		}
		if offset+length > len(payload) {
			t.Fatalf("TLV %d (type %#x) declares length %d at offset %d, overrunning the %d-byte payload",
				len(tlvs), typ, length, offset, len(payload))
		}

		tlvs = append(tlvs, fdpTLV{typ: typ, value: payload[offset+4 : offset+length]})
		offset += length
	}

	return tlvs
}

// TestBuildFDPFrameDecodesEndToEnd decodes a built advertisement with an
// independent decoder and asserts the values carried after the first
// variable-length TLV.
//
// What this adds, measured rather than assumed. FDP already has a chain walk in
// TestBuildFDPFrame_TLVChainIsWellFormed, and injected defects confirm the
// existing tests catch bad lengths, wrong values, a truncated frame and a wrong
// OUI. So this is defence in depth rather than a gap being closed, and #1329's
// premise that FDP is as exposed as CDP was does not hold.
//
// Two things here are genuinely new. The existing chain walk asserts structure —
// length consistency, no overrun, an exact TLV count — but never reads a single
// TLV's value; this one asserts that Port really carries the interface name and
// that the address TLV decodes back to the address advertised. Asserting counts
// while every value is wrong is a failure mode this codebase has hit before.
//
// And nothing else wraps an FDP advertisement in an Ethernet header, so the
// framing was only ever checked component by component. That matters more for
// FDP than for its siblings: FDP and CDP share protocol type 0x2000, and only
// the SNAP OUI — Foundry's 00:E0:52 against Cisco's 00:00:0C — says which
// protocol the bytes are. gopacket dispatches on the type alone and decodes this
// frame as CDP, which is why the OUI is asserted directly rather than trusting
// the layer gopacket picked.
func TestBuildFDPFrameDecodesEndToEnd(t *testing.T) {
	cfg := &config.Config{}
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := protocols.NewFDPHandler(stack)

	srcMAC := net.HardwareAddr{0x00, 0x1A, 0x2B, 0x3C, 0x4D, 0x5E}
	device := &config.Device{
		Name:        "Switch-1",
		Type:        "switch",
		MACAddress:  srcMAC,
		IPAddresses: []net.IP{net.ParseIP("192.168.1.1")},
		Interfaces:  []config.Interface{{Name: "GigabitEthernet0/1"}},
	}

	payload := handler.BuildFDPFrame(device)
	packet := gopacket.NewPacket(fdpFrame(t, payload, srcMAC), layers.LinkTypeEthernet, gopacket.Default)

	// LLC and SNAP are what carry FDP to a collector. A wrong DSAP or OUI here
	// means the advertisement is never handed to an FDP dissector at all,
	// however correct its TLVs are.
	llcLayer, ok := packet.Layer(layers.LayerTypeLLC).(*layers.LLC)
	if !ok {
		t.Fatal("no LLC layer decoded")
	}
	if llcLayer.DSAP != 0xAA || llcLayer.SSAP != 0xAA || llcLayer.Control != 0x03 {
		t.Errorf("LLC DSAP/SSAP/Control = %#x/%#x/%#x, want 0xaa/0xaa/0x03",
			llcLayer.DSAP, llcLayer.SSAP, llcLayer.Control)
	}

	snapLayer, ok := packet.Layer(layers.LayerTypeSNAP).(*layers.SNAP)
	if !ok {
		t.Fatal("no SNAP layer decoded")
	}

	// The OUI is the half that distinguishes FDP from CDP: both use protocol
	// type 0x2000, and only Foundry's 00:E0:52 versus Cisco's 00:00:0C says
	// which protocol these bytes are. gopacket dispatches on the type alone and
	// so decodes this frame as CDP — which is exactly why the OUI is asserted
	// here rather than trusting the layer gopacket chose.
	wantOUI := []byte{0x00, 0xE0, 0x52}
	if got := snapLayer.OrganizationalCode; !bytes.Equal(got, wantOUI) {
		t.Errorf("SNAP OUI = %x, want the Foundry OUI %x", got, wantOUI)
	}
	if snapLayer.Type != 0x2000 {
		t.Errorf("SNAP protocol type = %#x, want 0x2000 (FDP)", snapLayer.Type)
	}

	// The bytes SNAP hands up must be the advertisement, byte for byte. This is
	// what catches a framing mistake that leaves the TLVs themselves intact —
	// an off-by-one header size, or an 802.3 length field that truncates the
	// payload before its last TLV.
	if got := snapLayer.LayerPayload(); !bytes.Equal(got, payload[8:]) {
		t.Errorf("SNAP payload is %d bytes, want the %d-byte FDP advertisement",
			len(got), len(payload)-8)
	}

	// The FDP payload begins after the 8-byte LLC/SNAP header.
	tlvs := walkFDPTLVs(t, payload[8:])

	// Device ID is the first variable-length TLV, so everything below is what a
	// derailed walk loses. Reaching them in order, at the offsets the chain
	// itself leads to, is the point of this test.
	byType := make(map[uint16][]byte, len(tlvs))
	order := make([]uint16, 0, len(tlvs))
	for _, tlv := range tlvs {
		if _, seen := byType[tlv.typ]; seen {
			t.Errorf("TLV type %#x appears more than once", tlv.typ)
		}
		byType[tlv.typ] = tlv.value
		order = append(order, tlv.typ)
	}

	if len(order) == 0 || order[0] != protocols.FDPTLVTypeDeviceID {
		t.Fatalf("TLV order = %#x, want Device ID (%#x) first", order, protocols.FDPTLVTypeDeviceID)
	}
	if got := string(byType[protocols.FDPTLVTypeDeviceID]); got != "Switch-1" {
		t.Errorf("Device ID TLV = %q, want %q", got, "Switch-1")
	}

	// Port carries the interface name. Reading it as anything else is the
	// documented past failure, so assert the value and not merely the presence.
	port, ok := byType[protocols.FDPTLVTypePort]
	if !ok {
		t.Fatalf("no Port TLV after the variable-length Device ID — the walk derailed; got %#x", order)
	}
	if got := string(port); got != "GigabitEthernet0/1" {
		t.Errorf("Port TLV = %q, want %q", got, "GigabitEthernet0/1")
	}

	// Platform is emitted last, so it is the TLV a derailed walk is likeliest to
	// lose entirely.
	platform, hasPlatform := byType[protocols.FDPTLVTypePlatform]
	if !hasPlatform {
		t.Errorf("no Platform TLV — the last TLV did not survive the walk; got %#x", order)
	} else if len(platform) == 0 {
		t.Error("Platform TLV is empty")
	}

	// The address TLV must decode back to the address we advertised, through the
	// emitter's own parser — the CDP-shaped encoding here is the one #1326 got
	// wrong on the sibling protocol.
	addr, ok := byType[protocols.FDPTLVTypeIPAddress]
	if !ok {
		t.Fatalf("no IP Address TLV; got %#x", order)
	}
	if got := protocols.ParseFDPAddressTLV(addr); !got.Equal(net.ParseIP("192.168.1.1")) {
		t.Errorf("IP Address TLV decodes to %v, want 192.168.1.1", got)
	}
}
