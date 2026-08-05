package daemon

import (
	"encoding/binary"
	"errors"
)

type trunkReplaySender struct {
	transport *trunkSessionTransport
	vlan      uint16
}

func (s *trunkReplaySender) SendPacket(frame []byte) error {
	tagged, err := retagPhysicalVLAN(frame, s.vlan)
	if err != nil {
		return err
	}
	return s.transport.SendPacket(tagged)
}

func retagPhysicalVLAN(frame []byte, vlan uint16) ([]byte, error) {
	if len(frame) < ethernetHeaderLength {
		return nil, errors.New("replay frame is shorter than an Ethernet header")
	}
	if binary.BigEndian.Uint16(frame[12:14]) == dot1QEtherType {
		return replacePhysicalVLAN(frame, vlan)
	}
	return insertPhysicalVLAN(frame, vlan), nil
}

func replacePhysicalVLAN(frame []byte, vlan uint16) ([]byte, error) {
	if len(frame) < ethernetHeaderLength+dot1QHeaderLength {
		return nil, errors.New("replay frame has a truncated VLAN header")
	}
	tagged := append([]byte(nil), frame...)
	tci := binary.BigEndian.Uint16(tagged[14:16]) & ^uint16(dot1QVLANMask)
	binary.BigEndian.PutUint16(tagged[14:16], tci|vlan)
	return tagged, nil
}

func insertPhysicalVLAN(frame []byte, vlan uint16) []byte {
	tagged := make([]byte, len(frame)+dot1QHeaderLength)
	copy(tagged[:12], frame[:12])
	binary.BigEndian.PutUint16(tagged[12:14], dot1QEtherType)
	binary.BigEndian.PutUint16(tagged[14:16], vlan)
	copy(tagged[16:], frame[12:])
	return tagged
}
