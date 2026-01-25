package protocols

import (
	"fmt"
	"os"
)

// decodePacket decodes a packet and routes to appropriate handler.
func (s *Stack) decodePacket(pkt *Packet) {
	// Try MAC-based routing first (multicast protocols)
	if s.routeByMAC(pkt) {
		return
	}

	// Route by EtherType
	s.routeByEtherType(pkt)
}

// routeByMAC routes packets based on multicast MAC addresses.
// Returns true if packet was handled.
func (s *Stack) routeByMAC(pkt *Packet) bool {
	dstMAC := pkt.GetDestMAC()
	if len(dstMAC) != macAddrLen {
		return false
	}

	// Check for STP (multicast MAC 01:80:C2:00:00:00)
	if matchMAC(
		dstMAC,
		macMulticastIEEE,
		macSecondByteSTP,
		macThirdByteSTP,
		macUnicastZero,
		macUnicastZero,
		macUnicastZero,
	) {
		s.stpHandler.HandlePacket(pkt)
		return true
	}

	// Check for LLDP (multicast MAC 01:80:C2:00:00:0E)
	if matchMAC(
		dstMAC,
		macMulticastIEEE,
		macSecondByteSTP,
		macThirdByteSTP,
		macUnicastZero,
		macUnicastZero,
		macByteLLDP,
	) {
		s.lldpHandler.HandlePacket(pkt)
		return true
	}

	// Check for CDP (multicast MAC 01:00:0C:CC:CC:CC)
	if matchMAC(
		dstMAC,
		macMulticastIEEE,
		macUnicastZero,
		macThirdByteCDP,
		macByteCC,
		macByteCC,
		macByteCC,
	) {
		s.cdpHandler.HandlePacket(pkt)
		return true
	}

	// Check for EDP (multicast MAC 00:E0:2B:00:00:00)
	if matchMAC(
		dstMAC,
		macUnicastZero,
		macSecondByteEDP,
		macThirdByteEDP,
		macUnicastZero,
		macUnicastZero,
		macUnicastZero,
	) {
		s.edpHandler.HandlePacket(pkt)
		return true
	}

	// Check for FDP (multicast MAC 01:E0:52:CC:CC:CC)
	if matchMAC(
		dstMAC,
		macMulticastIEEE,
		macSecondByteEDP,
		macThirdByteFDP,
		macByteCC,
		macByteCC,
		macByteCC,
	) {
		s.fdpHandler.HandlePacket(pkt)
		return true
	}

	return false
}

// matchMAC checks if a MAC address matches the given bytes.
func matchMAC(mac []byte, b0, b1, b2, b3, b4, b5 byte) bool {
	return mac[0] == b0 && mac[1] == b1 && mac[2] == b2 &&
		mac[3] == b3 && mac[4] == b4 && mac[5] == b5
}

// routeByEtherType routes packets based on EtherType.
func (s *Stack) routeByEtherType(pkt *Packet) {
	etherType := pkt.GetEtherType()

	// Check for VLAN
	offset := SizeOfMac * ethMACCount
	if etherType == EtherTypeVLAN {
		// VLAN present, get actual EtherType
		offset += 4
		etherType = pkt.Get16(offset)
	}

	if s.debugConfig.GetGlobal() >= DebugLevelVerbose {
		_, _ = fmt.Fprintf(
			os.Stdout,
			"Decoding packet sn=%d etherType=0x%04x\n",
			pkt.SerialNumber,
			etherType,
		)
	}

	// Route to protocol handler
	switch etherType {
	case EtherTypeARP:
		s.arpHandler.HandlePacket(pkt)
	case EtherTypeIP:
		s.ipHandler.HandlePacket(pkt)
	case EtherTypeIPv6:
		s.ipv6Handler.HandlePacket(pkt)
	case EtherTypeLLDP:
		s.lldpHandler.HandlePacket(pkt)
	case EtherTypeEDP:
		s.edpHandler.HandlePacket(pkt)
	case EtherTypeFDP:
		s.fdpHandler.HandlePacket(pkt)
	default:
		if s.debugConfig.GetGlobal() >= DebugLevelInfo {
			_, _ = fmt.Fprintf(
				os.Stdout,
				"Unknown EtherType 0x%04x sn=%d\n",
				etherType,
				pkt.SerialNumber,
			)
		}
	}
}
