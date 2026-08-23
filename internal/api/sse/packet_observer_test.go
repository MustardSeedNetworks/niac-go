package sse

import (
	"strings"
	"testing"
	"time"

	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/layers"

	"github.com/MustardSeedNetworks/niac-go/internal/protocols"
)

func TestPacketToWirePreservesSimulationDirection(t *testing.T) {
	packet := &protocols.Packet{
		Buffer:       []byte{0x00, 0x11, 0x22, 0x33},
		Length:       4,
		SerialNumber: 7,
		Timestamp:    time.Date(2026, 7, 24, 12, 0, 0, 0, time.UTC),
		VLAN:         -1,
	}

	for _, direction := range []string{"rx", "tx"} {
		t.Run(direction, func(t *testing.T) {
			wire := packetToWire(direction, packet)
			if wire["direction"] != direction {
				t.Errorf("direction = %v, want %q", wire["direction"], direction)
			}
			if wire["raw_data"] != "00112233" {
				t.Errorf("raw_data = %v", wire["raw_data"])
			}
			if wire["serial"] != 7 {
				t.Errorf("serial = %v", wire["serial"])
			}
		})
	}
}

func TestGopacketToWireUsesStandaloneCaptureShape(t *testing.T) {
	raw := []byte{
		0x00, 0x11, 0x22, 0x33, 0x44, 0x55,
		0x66, 0x77, 0x88, 0x99, 0xaa, 0xbb,
		0x08, 0x00,
	}
	packet := gopacket.NewPacket(raw, layers.LayerTypeEthernet, gopacket.Default)

	wire := GopacketToWire(packet)
	if wire["direction"] != "rx" {
		t.Errorf("direction = %v, want rx", wire["direction"])
	}
	if wire["size"] != len(raw) {
		t.Errorf("size = %v, want %d", wire["size"], len(raw))
	}
	if wire["raw_data"] != "00112233445566778899aabb0800" {
		t.Errorf("raw_data = %v", wire["raw_data"])
	}
}

// TestPacketToWireNeverEmitsTheZeroTimestamp guards #1457.
//
// protocols.Packet carries a Timestamp, but only 3 of the 27 &Packet{...}
// literals outside tests set it: NewPacket and ParsePacket stamp time.Now(),
// while the protocol emitters build the struct directly and leave it zero.
// packetToWire formatted it unconditionally, so the inspector showed every
// frame arriving at "0001-01-01T00:00:00.000Z".
//
// The guard lives at the encoder because the encoder owns the wire contract —
// the 24 emitter sites are not a boundary, and a 25th would regress silently.
// Its documented cousin GopacketToWire already guards exactly this way.
func TestPacketToWireNeverEmitsTheZeroTimestamp(t *testing.T) {
	pkt := &protocols.Packet{Buffer: []byte{0xde, 0xad, 0xbe, 0xef}, Length: 4}
	if !pkt.Timestamp.IsZero() {
		t.Fatal("precondition: a bare Packet literal should carry the zero time")
	}

	got, _ := packetToWire("tx", pkt)["timestamp"].(string)
	if strings.HasPrefix(got, "0001-01-01") {
		t.Errorf("timestamp = %q, want an observation time, not the zero value", got)
	}
}

// A packet that does carry a real timestamp must keep it — the guard fills a
// gap, it does not overwrite recorded arrival times.
func TestPacketToWirePreservesARealTimestamp(t *testing.T) {
	when := time.Date(2026, 8, 23, 14, 30, 15, 0, time.UTC)
	pkt := &protocols.Packet{Buffer: []byte{0x01}, Length: 1, Timestamp: when}

	got, _ := packetToWire("rx", pkt)["timestamp"].(string)
	if want := "2026-08-23T14:30:15.000Z"; got != want {
		t.Errorf("timestamp = %q, want %q", got, want)
	}
}
