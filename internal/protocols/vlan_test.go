package protocols

import (
	"bytes"
	"testing"

	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/layers"
)

// untaggedFrame builds a minimal Ethernet+ARP frame (EtherType 0x0806).
func untaggedFrame(t *testing.T) []byte {
	t.Helper()
	eth := &layers.Ethernet{
		SrcMAC:       []byte{0x02, 0, 0, 0, 0, 0x01},
		DstMAC:       []byte{0x02, 0, 0, 0, 0, 0x02},
		EthernetType: layers.EthernetTypeARP,
	}
	arp := &layers.ARP{
		AddrType: layers.LinkTypeEthernet, Protocol: layers.EthernetTypeIPv4,
		HwAddressSize: 6, ProtAddressSize: 4, Operation: layers.ARPReply,
		SourceHwAddress: []byte{0x02, 0, 0, 0, 0, 0x01}, SourceProtAddress: []byte{10, 0, 0, 1},
		DstHwAddress: []byte{0x02, 0, 0, 0, 0, 0x02}, DstProtAddress: []byte{10, 0, 0, 2},
	}
	buf := gopacket.NewSerializeBuffer()
	if err := gopacket.SerializeLayers(buf, gopacket.SerializeOptions{FixLengths: true}, eth, arp); err != nil {
		t.Fatalf("serialize: %v", err)
	}
	return buf.Bytes()
}

func TestInsertDot1QTagsFrame(t *testing.T) {
	frame := untaggedFrame(t)
	tagged := insertDot1Q(frame, 210)

	if len(tagged) != len(frame)+4 {
		t.Fatalf("tagged len = %d, want %d", len(tagged), len(frame)+4)
	}
	// MACs preserved.
	if !bytes.Equal(tagged[:12], frame[:12]) {
		t.Error("MAC header not preserved")
	}
	// Decode and confirm the tag round-trips with the right VLAN + inner type.
	pkt := gopacket.NewPacket(tagged, layers.LayerTypeEthernet, gopacket.Default)
	dot1q, ok := pkt.Layer(layers.LayerTypeDot1Q).(*layers.Dot1Q)
	if !ok {
		t.Fatal("no Dot1Q layer after insert")
	}
	if dot1q.VLANIdentifier != 210 {
		t.Errorf("VLAN = %d, want 210", dot1q.VLANIdentifier)
	}
	if dot1q.Type != layers.EthernetTypeARP {
		t.Errorf("inner type = %v, want ARP", dot1q.Type)
	}
	if pkt.Layer(layers.LayerTypeARP) == nil {
		t.Error("ARP payload lost after tagging")
	}
}

func TestInsertDot1QNoOps(t *testing.T) {
	frame := untaggedFrame(t)
	cases := []struct {
		name string
		vlan int
		in   []byte
	}{
		{"vlan zero", 0, frame},
		{"vlan too big", 5000, frame},
		{"frame too short", 210, frame[:8]},
		{"already tagged", 210, insertDot1Q(frame, 100)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out := insertDot1Q(tc.in, tc.vlan)
			if len(out) != len(tc.in) {
				t.Errorf("expected no-op, len changed %d -> %d", len(tc.in), len(out))
			}
		})
	}
}
