package capture

import (
	"errors"
	"fmt"
)

// Structure-validation errors. These wrap the magic-number check so callers can
// errors.Is against a stable sentinel while still surfacing the offending value.
var (
	// ErrFileTooSmall reports a buffer too small to be a valid PCAP.
	ErrFileTooSmall = errors.New("file too small to be a valid PCAP (< 4 bytes)")
	// ErrInvalidMagicNumber reports an unrecognized PCAP/pcapng magic or header.
	ErrInvalidMagicNumber = errors.New("invalid PCAP magic number")
)

// minSize is the minimum byte length required to read a PCAP magic number.
const minSize = 4

// Byte-shift magnitudes for decoding big/little-endian header fields. Kept
// local so the leaf has no dependency on the api transport layer.
const (
	bitShift8  = 8
	bitShift16 = 16
	bitShift24 = 24
)

// PCAP magic number constants.
const (
	pcapMagicBigEndian        = 0xa1b2c3d4 // Standard pcap big-endian
	pcapMagicBigEndianNano    = 0xa1b23c4d // Nanosecond pcap big-endian
	pcapMagicLittleEndian     = 0xd4c3b2a1 // Standard pcap little-endian
	pcapMagicLittleEndianNano = 0x4d3cb2a1 // Nanosecond pcap little-endian
	pcapngMagic               = 0x0a0d0d0a // pcapng Section Header Block
)

// PCAP Data Link Type (DLT) constants from libpcap.
// See: https://www.tcpdump.org/linktypes.html
const (
	dltNull              uint32 = 0   // BSD loopback encapsulation
	dltEN10MB            uint32 = 1   // Ethernet (10Mb, 100Mb, 1000Mb, and up)
	dltIEEE802           uint32 = 6   // 802.5 Token Ring
	dltSLIP              uint32 = 8   // SLIP
	dltPPP               uint32 = 9   // PPP
	dltFDDI              uint32 = 12  // FDDI
	dltRaw               uint32 = 101 // Raw IP
	dltIEEE80211         uint32 = 105 // IEEE 802.11 wireless
	dltLinuxSLL          uint32 = 113 // Linux cooked capture
	dltIEEE80211Radiotap uint32 = 127 // Radiotap link-layer information
	dltMTP2Pseudoheader  uint32 = 140 // MTP2 with pseudo-header
	dltIEEE80211AVS      uint32 = 147 // AVS header + 802.11 frame
	dltRawOpenBSD        uint32 = 162 // Raw IP (OpenBSD)
	dltNFLOG             uint32 = 163 // NFLOG
	dltBluetoothHCI      uint32 = 187 // Bluetooth HCI UART transport
	dltUSB               uint32 = 189 // USB packets
	dltIEEE802154        uint32 = 195 // IEEE 802.15.4
	dltPPP2              uint32 = 204 // PPP (alternative)
	dltIPv4              uint32 = 228 // Raw IPv4
	dltIPv6              uint32 = 229 // Raw IPv6
	dltNFQUEUE           uint32 = 253 // NFQUEUE
)

var validLinkTypes = map[uint32]bool{ //nolint:gochecknoglobals // Static DLT lookup table.
	dltNull:              true,
	dltEN10MB:            true,
	dltIEEE802:           true,
	dltSLIP:              true,
	dltPPP:               true,
	dltFDDI:              true,
	dltRaw:               true,
	dltIEEE80211:         true,
	dltLinuxSLL:          true,
	dltIEEE80211Radiotap: true,
	dltMTP2Pseudoheader:  true,
	dltIEEE80211AVS:      true,
	dltRawOpenBSD:        true,
	dltNFLOG:             true,
	dltBluetoothHCI:      true,
	dltUSB:               true,
	dltIEEE802154:        true,
	dltPPP2:              true,
	dltIPv4:              true,
	dltIPv6:              true,
	dltNFQUEUE:           true,
}

// isValidLinkType checks if a link-layer type is recognized.
// SECURITY FIX #170: Only accept recognized link-layer types.
func isValidLinkType(linkType uint32) bool {
	return validLinkTypes[linkType]
}

// ValidateStructure validates that the file has a valid PCAP/pcapng structure.
// SECURITY FIX #170: Enhanced beyond magic-only validation to check header structure.
func ValidateStructure(data []byte) error {
	if len(data) < minSize {
		return ErrFileTooSmall
	}

	// Check for valid PCAP magic numbers
	magic := uint32(data[0])<<24 | uint32(data[1])<<16 | uint32(data[2])<<8 | uint32(data[3])

	switch magic {
	case pcapMagicBigEndian, pcapMagicBigEndianNano:
		// Big-endian pcap format
		return validateGlobalHeader(data, true)
	case pcapMagicLittleEndian, pcapMagicLittleEndianNano:
		// Little-endian pcap format
		return validateGlobalHeader(data, false)
	case pcapngMagic:
		// pcapng format: validate Section Header Block
		return validateNgSHB(data)
	default:
		return fmt.Errorf(
			"%w: 0x%08x (expected pcap or pcapng format)",
			ErrInvalidMagicNumber,
			magic,
		)
	}
}

// validateGlobalHeader validates the pcap global header structure (24 bytes).
func validateGlobalHeader(data []byte, bigEndian bool) error {
	const pcapGlobalHeaderLen = 24

	if len(data) < pcapGlobalHeaderLen {
		return fmt.Errorf("%w: file too small for pcap global header (need %d bytes, got %d)",
			ErrInvalidMagicNumber, pcapGlobalHeaderLen, len(data))
	}

	var versionMajor, versionMinor, linkType uint32
	if bigEndian {
		versionMajor = uint32(data[4])<<bitShift8 | uint32(data[5])
		versionMinor = uint32(data[6])<<bitShift8 | uint32(data[7])
		linkType = uint32(
			data[20],
		)<<bitShift24 | uint32(
			data[21],
		)<<bitShift16 | uint32(
			data[22],
		)<<bitShift8 | uint32(
			data[23],
		)
	} else {
		versionMajor = uint32(data[5])<<bitShift8 | uint32(data[4])
		versionMinor = uint32(data[7])<<bitShift8 | uint32(data[6])
		linkType = uint32(
			data[23],
		)<<bitShift24 | uint32(
			data[22],
		)<<bitShift16 | uint32(
			data[21],
		)<<bitShift8 | uint32(
			data[20],
		)
	}

	// Validate version (must be 2.4)
	if versionMajor != 2 || versionMinor != 4 {
		return fmt.Errorf("%w: unsupported pcap version %d.%d (expected 2.4)",
			ErrInvalidMagicNumber, versionMajor, versionMinor)
	}

	// Validate link-layer type
	if !isValidLinkType(linkType) {
		return fmt.Errorf("%w: unknown link-layer type %d",
			ErrInvalidMagicNumber, linkType)
	}

	return nil
}

// validateNgSHB validates the pcapng Section Header Block.
func validateNgSHB(data []byte) error {
	const minSHBLen = 28

	if len(data) < minSHBLen {
		return fmt.Errorf(
			"%w: file too small for pcapng Section Header Block (need %d bytes, got %d)",
			ErrInvalidMagicNumber,
			minSHBLen,
			len(data),
		)
	}

	// Bytes 8-11 contain Byte-Order Magic to determine endianness
	bom := uint32(data[8])<<24 | uint32(data[9])<<16 | uint32(data[10])<<8 | uint32(data[11])
	if bom != 0x1a2b3c4d && bom != 0x4d3c2b1a {
		return fmt.Errorf("%w: invalid pcapng byte-order magic 0x%08x",
			ErrInvalidMagicNumber, bom)
	}

	return nil
}
