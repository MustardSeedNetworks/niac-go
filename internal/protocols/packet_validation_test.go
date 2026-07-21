package protocols

import (
	"encoding/binary"
	"errors"
	"testing"
)

func TestParsePacketRejectsTruncatedEthernetAndVLANHeaders(t *testing.T) {
	tests := []struct {
		name  string
		frame []byte
		want  error
	}{
		{name: "empty", frame: nil, want: ErrEthernetFrameTooShort},
		{name: "short Ethernet", frame: make([]byte, etherHeaderSize-1), want: ErrEthernetFrameTooShort},
		{name: "short VLAN", frame: vlanHeaderOnly(), want: ErrVLANHeaderTruncated},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParsePacket(tt.frame, 1)
			if !errors.Is(err, tt.want) {
				t.Fatalf("ParsePacket() error = %v, want %v", err, tt.want)
			}
		})
	}
}

func TestParsePacketMarksPriorityTaggedFrame(t *testing.T) {
	frame := rawVLANTag(makeEthernetFrame(EtherTypeIP), dot1qTPID, 0)
	pkt, err := ParsePacket(frame, 1)
	if err != nil {
		t.Fatalf("ParsePacket() error = %v", err)
	}
	if !pkt.VLANTagged || pkt.VLAN != 0 {
		t.Fatalf("parsed VLAN metadata = (tagged=%t, vlan=%d), want (true, 0)", pkt.VLANTagged, pkt.VLAN)
	}
	runtime := &fabricRuntime{}
	runtime.binding.Mode = "access"
	if runtime.acceptsFrame(pkt.VLAN, pkt.VLANTagged) {
		t.Fatal("routed access binding accepted an 802.1Q priority-tagged frame")
	}
}

func TestParsePacketRejectsStackedVLANTags(t *testing.T) {
	frame := rawVLANTag(rawVLANTag(makeEthernetFrame(EtherTypeIP), dot1qTPID, 200), dot1qTPID, 100)
	_, err := ParsePacket(frame, 1)
	if !errors.Is(err, ErrVLANStackUnsupported) {
		t.Fatalf("ParsePacket() error = %v, want %v", err, ErrVLANStackUnsupported)
	}
}

func vlanHeaderOnly() []byte {
	frame := make([]byte, etherHeaderSize)
	binary.BigEndian.PutUint16(frame[ethMACsLen:], dot1qTPID)
	return frame
}

func makeEthernetFrame(etherType uint16) []byte {
	frame := make([]byte, etherHeaderSize+8)
	binary.BigEndian.PutUint16(frame[ethMACsLen:], etherType)
	return frame
}

func rawVLANTag(frame []byte, tpid, vlan uint16) []byte {
	tagged := make([]byte, len(frame)+dot1qTagLen)
	copy(tagged, frame[:ethMACsLen])
	binary.BigEndian.PutUint16(tagged[ethMACsLen:], tpid)
	binary.BigEndian.PutUint16(tagged[ethMACsLen+2:], vlan&dot1qVLANIDMask)
	copy(tagged[ethMACsLen+dot1qTagLen:], frame[ethMACsLen:])
	return tagged
}
