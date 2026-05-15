package api

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"
)

// TestSnoopMagicDetection covers the cheap probe used in the upload path.
func TestSnoopMagicDetection(t *testing.T) {
	tests := []struct {
		name string
		in   []byte
		want bool
	}{
		{"valid magic", append([]byte("snoop\x00\x00\x00"), 0, 0), true},
		{"empty", nil, false},
		{"too short", []byte("sno"), false},
		{"pcap magic", []byte{0xa1, 0xb2, 0xc3, 0xd4, 0, 0, 0, 0, 0, 0}, false},
		{"snoop ascii but wrong padding", []byte("snoop\x01\x02\x03\x00"), false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := hasSnoopMagic(tc.in); got != tc.want {
				t.Errorf("hasSnoopMagic(%v) = %v, want %v", tc.in, got, tc.want)
			}
		})
	}
}

// TestSnoopToPCAPHeader verifies the converted file is recognisable as
// classic libpcap and carries the correct linktype.
func TestSnoopToPCAPHeader(t *testing.T) {
	snoop := makeMinimalSnoop(t, snoopDLTEthernet)

	out, err := snoopToPCAP(snoop)
	if err != nil {
		t.Fatalf("conversion failed: %v", err)
	}
	if len(out) < 24 {
		t.Fatalf("converted file too short: %d bytes", len(out))
	}

	// On-disk LE-pcap bytes are [d4 c3 b2 a1]; reading them as BE uint32
	// yields pcapMagicLittleEndian (0xd4c3b2a1), which is what the
	// shared validator switches on. Reading as LE yields the canonical
	// 0xa1b2c3d4 — both are equivalent representations.
	magic := binary.BigEndian.Uint32(out[0:4])
	if magic != pcapMagicLittleEndian {
		t.Errorf("magic = 0x%08x, want 0x%08x (pcapMagicLittleEndian)",
			magic, pcapMagicLittleEndian)
	}
	linkType := binary.LittleEndian.Uint32(out[20:24])
	if linkType != dltEN10MB {
		t.Errorf("linkType = %d, want %d (dltEN10MB)", linkType, dltEN10MB)
	}

	// And it should pass the same validator the upload path uses.
	if vErr := validatePCAPStructure(out); vErr != nil {
		t.Errorf("converted output failed validatePCAPStructure: %v", vErr)
	}
}

// TestSnoopToPCAPRoundTripPacket builds a snoop file with one synthetic
// packet, converts it, then walks the pcap record and confirms the
// payload bytes / lengths / timestamp survived intact.
func TestSnoopToPCAPRoundTripPacket(t *testing.T) {
	payload := []byte{0xaa, 0xbb, 0xcc, 0xdd, 0xee}
	snoop := makeMinimalSnoop(t, snoopDLTEthernet)
	snoop = append(snoop, makeSnoopRecord(t, payload, 12345, 678901)...)

	out, err := snoopToPCAP(snoop)
	if err != nil {
		t.Fatalf("conversion failed: %v", err)
	}

	// pcap = 24-byte file header + 16-byte record header + payload
	if got, want := len(out), 24+16+len(payload); got != want {
		t.Fatalf("converted file length = %d, want %d", got, want)
	}

	rec := out[24:]
	tsSec := binary.LittleEndian.Uint32(rec[0:4])
	tsUsec := binary.LittleEndian.Uint32(rec[4:8])
	inclLen := binary.LittleEndian.Uint32(rec[8:12])
	origLen := binary.LittleEndian.Uint32(rec[12:16])

	if tsSec != 12345 || tsUsec != 678901 {
		t.Errorf("timestamp = (%d, %d), want (12345, 678901)", tsSec, tsUsec)
	}
	if inclLen != uint32(len(payload)) || origLen != uint32(len(payload)) {
		t.Errorf("lengths = (%d, %d), want (%d, %d)",
			inclLen, origLen, len(payload), len(payload))
	}

	gotPayload := rec[16:]
	for i, b := range payload {
		if gotPayload[i] != b {
			t.Errorf("payload[%d] = 0x%02x, want 0x%02x", i, gotPayload[i], b)
		}
	}
}

// TestSnoopToPCAPRejects covers the bail-out paths.
func TestSnoopToPCAPRejects(t *testing.T) {
	t.Run("too short", func(t *testing.T) {
		if _, err := snoopToPCAP([]byte("snoo")); err == nil {
			t.Error("expected error for short input, got nil")
		}
	})

	t.Run("missing magic", func(t *testing.T) {
		junk := make([]byte, snoopHeaderLen)
		if _, err := snoopToPCAP(junk); err == nil {
			t.Error("expected error for missing magic, got nil")
		}
	})

	t.Run("unsupported version", func(t *testing.T) {
		bad := makeMinimalSnoop(t, snoopDLTEthernet)
		binary.BigEndian.PutUint32(bad[8:12], 99) // way above supported range
		if _, err := snoopToPCAP(bad); err == nil {
			t.Error("expected error for unsupported version, got nil")
		}
	})

	// Solaris/illumos extension types (10+) used to be rejected. Now
	// they fall through to Ethernet so the shipped example captures
	// import cleanly. The trade-off is documented in snoopDLTToPCAP.
	t.Run("unknown datalink falls back to ethernet", func(t *testing.T) {
		base := makeMinimalSnoop(t, 10)
		out, err := snoopToPCAP(base)
		if err != nil {
			t.Fatalf("unexpected error for fallback datalink: %v", err)
		}
		linkType := binary.LittleEndian.Uint32(out[20:24])
		if linkType != dltEN10MB {
			t.Errorf("fallback linkType = %d, want %d (dltEN10MB)", linkType, dltEN10MB)
		}
	})
}

// TestSnoopExampleCapture exercises the converter against the actual
// mislabelled-snoop files shipped under examples/captures. If those
// files ever get re-saved as real pcap, this test will start failing
// the hasSnoopMagic precondition — which is the right place to learn
// the layout has changed.
func TestSnoopExampleCapture(t *testing.T) {
	candidates, _ := filepath.Glob("../../examples/captures/*.pcap")
	if len(candidates) == 0 {
		t.Skip("no example captures present")
	}

	for _, path := range candidates {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		if !hasSnoopMagic(data) {
			continue // genuinely a pcap; nothing for us to do
		}

		converted, convErr := snoopToPCAP(data)
		if convErr != nil {
			t.Errorf("%s: snoopToPCAP failed: %v", path, convErr)
			continue
		}
		if vErr := validatePCAPStructure(converted); vErr != nil {
			t.Errorf("%s: converted output fails validatePCAPStructure: %v", path, vErr)
		}
	}
}

// makeMinimalSnoop returns a 16-byte snoop header with the given DLT
// at the minimum supported snoop version. Tests that need a non-default
// version mutate the returned bytes directly (see "unsupported version"
// subtest of TestSnoopToPCAPRejects).
func makeMinimalSnoop(t *testing.T, datalink uint32) []byte {
	t.Helper()
	hdr := make([]byte, snoopHeaderLen)
	copy(hdr[0:8], snoopMagic)
	binary.BigEndian.PutUint32(hdr[8:12], snoopVersionMin)
	binary.BigEndian.PutUint32(hdr[12:16], datalink)
	return hdr
}

// makeSnoopRecord builds a single snoop packet record (24-byte header +
// payload + padding to 4-byte boundary). Caller passes payload + ts.
func makeSnoopRecord(t *testing.T, payload []byte, tsSec, tsUsec uint32) []byte {
	t.Helper()
	inclLen := uint32(len(payload))
	pad := (4 - (snoopRecordHeader+inclLen)%4) % 4
	recordLen := snoopRecordHeader + inclLen + pad

	rec := make([]byte, recordLen)
	binary.BigEndian.PutUint32(rec[0:4], inclLen)    // original_length
	binary.BigEndian.PutUint32(rec[4:8], inclLen)    // included_length
	binary.BigEndian.PutUint32(rec[8:12], recordLen) // packet_record_length
	binary.BigEndian.PutUint32(rec[12:16], 0)        // cumulative_drops
	binary.BigEndian.PutUint32(rec[16:20], tsSec)
	binary.BigEndian.PutUint32(rec[20:24], tsUsec)
	copy(rec[24:], payload)
	return rec
}
