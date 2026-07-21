package protocols

import (
	"encoding/binary"
	"errors"
)

// 802.1Q VLAN tag constants.
const (
	ethMACsLen      = 12     // dst MAC (6) + src MAC (6)
	ethHeaderLen    = 14     // MACs + EtherType
	dot1qTPID       = 0x8100 // 802.1Q Tag Protocol Identifier
	dot1adTPID      = 0x88A8 // provider bridging Tag Protocol Identifier
	legacyQinQTPID  = 0x9100 // common legacy stacked-VLAN Tag Protocol Identifier
	dot1qTagLen     = 4      // TPID (2) + TCI (2)
	vlanIDMax       = 4094   // valid VLAN IDs are 1..4094
	dot1qVLANIDMask = 0x0FFF // low 12 bits of the TCI carry the VLAN ID
)

// insertDot1Q returns frame with an 802.1Q VLAN tag inserted after the source
// MAC (priority 0, DEI 0). It is a no-op when the frame is too short, the VLAN
// ID is out of range (valid 1..4094), or the frame already carries a tag — so it
// is safe to call unconditionally on any reply frame. The input slice is never
// mutated.
func insertDot1Q(frame []byte, vlanID int) []byte {
	if len(frame) < ethHeaderLen || vlanID <= 0 || vlanID > vlanIDMax {
		return frame
	}
	if isVLANTPID(binary.BigEndian.Uint16(frame[ethMACsLen:])) {
		return frame // already tagged; do not double-tag
	}

	tci := uint16(vlanID) & dot1qVLANIDMask

	tagged := make([]byte, len(frame)+dot1qTagLen)
	copy(tagged, frame[:ethMACsLen])
	binary.BigEndian.PutUint16(tagged[ethMACsLen:], dot1qTPID)
	binary.BigEndian.PutUint16(tagged[ethMACsLen+2:], tci)
	copy(tagged[ethMACsLen+dot1qTagLen:], frame[ethMACsLen:])

	return tagged
}

func stripDot1Q(frame []byte) ([]byte, error) {
	for len(frame) >= ethHeaderLen && isVLANTPID(binary.BigEndian.Uint16(frame[ethMACsLen:])) {
		if len(frame) < ethHeaderLen+dot1qTagLen {
			return nil, errors.New("truncated 802.1Q Ethernet frame")
		}
		untagged := make([]byte, len(frame)-dot1qTagLen)
		copy(untagged, frame[:ethMACsLen])
		copy(untagged[ethMACsLen:], frame[ethMACsLen+dot1qTagLen:])
		frame = untagged
	}
	return frame, nil
}

func isVLANTPID(etherType uint16) bool {
	return etherType == dot1qTPID || etherType == dot1adTPID || etherType == legacyQinQTPID
}
