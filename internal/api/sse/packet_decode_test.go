package sse

import "testing"

// A real 802.1Q-tagged RSTP BPDU captured from NIAC's own trunk on CT304.
// This is the exact frame shape the Packets page reported as
// "Unknown  Unknown->Unknown" while rendering its bytes correctly (D16).
//
//	01 80 c2 00 00 00  dst — IEEE 802.1D bridge group
//	d4 c1 9e 23 58 5e  src
//	81 00 01 2b        802.1Q tag, VLAN 0x12b = 299
//	00 27              LLC length 39 (not an EtherType)
//	42 42 03           LLC DSAP/SSAP 0x42, control 0x03 → STP
func taggedBPDU() []byte {
	return []byte{
		0x01, 0x80, 0xc2, 0x00, 0x00, 0x00,
		0xd4, 0xc1, 0x9e, 0x23, 0x58, 0x5e,
		0x81, 0x00, 0x01, 0x2b,
		0x00, 0x27,
		0x42, 0x42, 0x03,
		0x00, 0x00, 0x02, 0x02, 0x7e,
		0x00, 0x00, 0xd4, 0xc1, 0x9e, 0x23, 0x58, 0x43,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0xd4, 0xc1, 0x9e, 0x23, 0x58, 0x43,
		0x80, 0x81, 0x00, 0x00, 0x14, 0x00, 0x02, 0x00, 0x0f, 0x00,
	}
}

func TestEnrichWithLayersDecodesTaggedBPDU(t *testing.T) {
	out := map[string]any{"protocol": "Unknown", "summary": ""}
	enrichWithLayers(out, taggedBPDU())

	if got := out["protocol"]; got != "STP" {
		t.Errorf("protocol = %v, want STP — a tagged BPDU is not Unknown", got)
	}
	if got := out["vlan_tag"]; got != uint16(299) {
		t.Errorf("vlan_tag = %v, want 299", got)
	}
	if got := out["src_mac"]; got != "d4:c1:9e:23:58:5e" {
		t.Errorf("src_mac = %v, want d4:c1:9e:23:58:5e", got)
	}
	if got := out["dst_mac"]; got != "01:80:c2:00:00:00" {
		t.Errorf("dst_mac = %v, want 01:80:c2:00:00:00", got)
	}
}

// The UI builds its layer tree from a nested `headers` map. Emitting only flat
// keys meant it never found an ethernet layer and printed "(not parsed)" for
// the MACs of every packet.
func TestEnrichWithLayersEmitsNestedHeaders(t *testing.T) {
	out := map[string]any{"protocol": "Unknown", "summary": ""}
	enrichWithLayers(out, taggedBPDU())

	headers, ok := out["headers"].(map[string]any)
	if !ok {
		t.Fatal("no headers map emitted; the UI reads headers.ethernet")
	}
	eth, ok := headers["ethernet"].(map[string]any)
	if !ok {
		t.Fatal("headers.ethernet missing")
	}
	if eth["srcMac"] != "d4:c1:9e:23:58:5e" {
		t.Errorf("headers.ethernet.srcMac = %v", eth["srcMac"])
	}
	if _, hasIP := headers["ipv4"]; hasIP {
		t.Error("an STP BPDU must not report an IPv4 layer")
	}
}

// An IP packet must still classify as before — the L2 pass runs last and only
// fills a protocol still marked Unknown.
func TestEnrichWithLayersStillClassifiesIPv4(t *testing.T) {
	frame := []byte{
		0x00, 0x11, 0x22, 0x33, 0x44, 0x55,
		0x66, 0x77, 0x88, 0x99, 0xaa, 0xbb,
		0x08, 0x00,
		0x45, 0x00, 0x00, 0x1c, 0x00, 0x01, 0x00, 0x00, 0x40, 0x01, 0x00, 0x00,
		0x0a, 0x00, 0x00, 0x01,
		0x0a, 0x00, 0x00, 0x02,
		0x08, 0x00, 0xf7, 0xff, 0x00, 0x01, 0x00, 0x01,
	}
	out := map[string]any{"protocol": "Unknown", "summary": ""}
	enrichWithLayers(out, frame)

	if got := out["protocol"]; got != "ICMP" {
		t.Errorf("protocol = %v, want ICMP", got)
	}
	if got := out["source_ip"]; got != "10.0.0.1" {
		t.Errorf("source_ip = %v, want 10.0.0.1", got)
	}
	headers, _ := out["headers"].(map[string]any)
	if _, hasIP := headers["ipv4"]; !hasIP {
		t.Error("headers.ipv4 missing for an IPv4 packet")
	}
}
