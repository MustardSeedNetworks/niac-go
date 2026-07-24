package sse

import (
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
