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

// edpFrame wraps a built EDP payload in the 802.3 header the handler sends it
// under, so an independent decoder sees what the wire sees. EDP is
// LLC/SNAP-encapsulated, so the length/type field carries the payload length
// rather than an EtherType.
func edpFrame(t *testing.T, payload []byte, srcMAC net.HardwareAddr) []byte {
	t.Helper()

	dstMAC, err := net.ParseMAC(protocols.EDPMulticastMAC)
	if err != nil {
		t.Fatalf("parse EDP multicast MAC: %v", err)
	}

	frame := make([]byte, 0, 14+len(payload))
	frame = append(frame, dstMAC...)
	frame = append(frame, srcMAC...)
	frame = binary.BigEndian.AppendUint16(frame, uint16(len(payload)))
	return append(frame, payload...)
}

// edpTLV is one decoded marker-prefixed TLV.
type edpTLV struct {
	typ   byte
	value []byte
}

// walkEDPTLVs decodes the TLV chain the way a collector does — in order, taking
// each TLV's length as the step to the next one — rather than checking TLVs at
// known offsets. A TLV whose declared length disagrees with its real size
// derails every TLV after it, and that is only visible to a walk.
func walkEDPTLVs(t *testing.T, payload []byte) []edpTLV {
	t.Helper()

	// Version(1) + Reserved(1) + Length(2) + Checksum(2) + Sequence(2) +
	// MachineIDType(2) + MAC(6), zero-padded to the 16-byte EDP header.
	const headerSize = 16
	if len(payload) < headerSize {
		t.Fatalf("payload is %d bytes, shorter than the %d-byte EDP header", len(payload), headerSize)
	}

	// The header's length field must account for the whole advertisement, or a
	// collector truncates the chain before reaching the last TLV.
	if declared := binary.BigEndian.Uint16(payload[2:4]); int(declared) != len(payload) {
		t.Errorf("header length field = %d, want %d (the full header+TLV length)",
			declared, len(payload))
	}

	var tlvs []edpTLV
	for offset := headerSize; offset < len(payload); {
		if remaining := len(payload) - offset; remaining < 4 {
			t.Fatalf("TLV %d at offset %d: %d trailing bytes, too few for a 4-byte TLV header",
				len(tlvs), offset, remaining)
		}

		if marker := payload[offset]; marker != 0x99 {
			t.Fatalf("TLV %d at offset %d: marker = %#x, want 0x99 — the walk has derailed",
				len(tlvs), offset, marker)
		}

		typ := payload[offset+1]
		length := int(binary.BigEndian.Uint16(payload[offset+2 : offset+4]))
		if length < 4 {
			t.Fatalf("TLV %d (type %#x) declares length %d, shorter than its own 4-byte header",
				len(tlvs), typ, length)
		}
		if offset+length > len(payload) {
			t.Fatalf("TLV %d (type %#x) declares length %d at offset %d, overrunning the %d-byte payload",
				len(tlvs), typ, length, offset, len(payload))
		}

		tlvs = append(tlvs, edpTLV{typ: typ, value: payload[offset+4 : offset+length]})

		if typ == protocols.EDPTLVTypeNull {
			break
		}
		offset += length
	}

	return tlvs
}

// TestBuildEDPFrameDecodesEndToEnd decodes a built advertisement with an
// independent decoder and walks its TLV chain in order.
//
// What this adds, measured rather than assumed. Five defects were injected into
// the emitter to find out what the existing tests already cover, and they are
// stronger than #1329 assumed — a bad TLV length, a wrong header length, a
// truncated frame and a wrong OUI are all caught today by the per-TLV and
// frame-level tests. This test is therefore defence in depth, not the closing
// of a demonstrated hole.
//
// The one real asymmetry it does close: fdp_test.go has a TLV chain walk
// (TestBuildFDPFrame_TLVChainIsWellFormed) and edp_test.go has none, so nothing
// here read EDP's TLVs the way a collector does — in order, stepping by each
// declared length. That is the shape that let a malformed CDP Address TLV ship
// (#1326): a TLV can read as well-formed on its own terms and still derail a
// decoder walking the chain, and everything after it is silently lost.
//
// It also puts the frame through gopacket, which nothing else here does — the
// per-TLV tests never wrap the advertisement in an Ethernet header at all, so
// the 802.3/LLC/SNAP framing a collector must traverse was previously only
// checked component by component.
func TestBuildEDPFrameDecodesEndToEnd(t *testing.T) {
	cfg := &config.Config{}
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := protocols.NewEDPHandler(stack)

	srcMAC := net.HardwareAddr{0x00, 0x1A, 0x2B, 0x3C, 0x4D, 0x5E}
	device := &config.Device{
		Name:        "Switch-1",
		Type:        "switch",
		MACAddress:  srcMAC,
		IPAddresses: []net.IP{net.ParseIP("192.168.1.1")},
		Interfaces:  []config.Interface{{Name: "GigabitEthernet0/1"}},
	}

	payload := handler.BuildEDPFrame(device)
	packet := gopacket.NewPacket(edpFrame(t, payload, srcMAC), layers.LinkTypeEthernet, gopacket.Default)

	// Note on what is NOT asserted here: gopacket dispatches a SNAP payload on
	// the protocol type alone, ignoring the OUI that gives it meaning, and it
	// knows no EDP dissector — so it reports "Unable to decode EthernetType 187"
	// (0x00BB) and leaves an error layer behind on a perfectly well-formed
	// frame. Wireshark, which keys on the OUI (llc.extreme_pid), dissects the
	// same bytes as EDP. Requiring ErrorLayer() == nil would therefore assert
	// gopacket's dissector coverage rather than our wire format.
	//
	// What is asserted instead is stronger for our purposes: that every layer a
	// collector must traverse to reach EDP decoded correctly, and that the bytes
	// handed up are exactly the advertisement we built — nothing truncated,
	// padded or shifted.

	// LLC and SNAP are what carry EDP to a collector. A wrong DSAP or OUI here
	// means the advertisement is never handed to an EDP dissector at all, however
	// correct its TLVs are.
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
	if got := uint32(snapLayer.OrganizationalCode[0])<<16 |
		uint32(snapLayer.OrganizationalCode[1])<<8 |
		uint32(snapLayer.OrganizationalCode[2]); got != protocols.EDPOrgCode {
		t.Errorf("SNAP OUI = %#06x, want the Extreme OUI %#06x", got, protocols.EDPOrgCode)
	}
	if snapLayer.Type != 0x00BB {
		t.Errorf("SNAP protocol type = %#x, want 0x00bb (EDP)", snapLayer.Type)
	}

	// The bytes SNAP hands up must be the advertisement, byte for byte. This is
	// what catches a framing mistake that leaves the TLVs themselves intact —
	// an off-by-one header size, or an 802.3 length field that truncates the
	// payload before its last TLV.
	if got := snapLayer.LayerPayload(); !bytes.Equal(got, payload[8:]) {
		t.Errorf("SNAP payload is %d bytes, want the %d-byte EDP advertisement",
			len(got), len(payload)-8)
	}

	// The EDP payload begins after the 8-byte LLC/SNAP header.
	tlvs := walkEDPTLVs(t, payload[8:])

	// Display is the first variable-length TLV, so Info and the Null terminator
	// are the ones a derailed walk loses. Asserting they are still reachable —
	// in order, at the offsets the chain itself leads to — is the point of this
	// test.
	if len(tlvs) < 3 {
		t.Fatalf("walked %d TLVs, want at least Display, Info and Null", len(tlvs))
	}
	if tlvs[0].typ != protocols.EDPTLVTypeDisplay {
		t.Errorf("first TLV type = %#x, want Display (%#x)", tlvs[0].typ, protocols.EDPTLVTypeDisplay)
	}
	if got := string(tlvs[0].value); got != "Switch-1 (switch)" {
		t.Errorf("Display TLV = %q, want %q", got, "Switch-1 (switch)")
	}
	if tlvs[1].typ != protocols.EDPTLVTypeInfo {
		t.Errorf("TLV after the variable-length Display = %#x, want Info (%#x) — the walk derailed",
			tlvs[1].typ, protocols.EDPTLVTypeInfo)
	}
	if last := tlvs[len(tlvs)-1]; last.typ != protocols.EDPTLVTypeNull {
		t.Errorf("last TLV type = %#x, want the Null end marker (%#x)", last.typ, protocols.EDPTLVTypeNull)
	}
}
