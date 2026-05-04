package api

import (
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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

// validatePCAPStructure validates that the file has a valid PCAP/pcapng structure.
// SECURITY FIX #170: Enhanced beyond magic-only validation to check header structure.
func validatePCAPStructure(data []byte) error {
	if len(data) < minPCAPSize {
		return ErrFileTooSmallForPCAP
	}

	// Check for valid PCAP magic numbers
	magic := uint32(data[0])<<24 | uint32(data[1])<<16 | uint32(data[2])<<8 | uint32(data[3])

	switch magic {
	case pcapMagicBigEndian, pcapMagicBigEndianNano:
		// Big-endian pcap format
		return validatePCAPGlobalHeader(data, true)
	case pcapMagicLittleEndian, pcapMagicLittleEndianNano:
		// Little-endian pcap format
		return validatePCAPGlobalHeader(data, false)
	case pcapngMagic:
		// pcapng format: validate Section Header Block
		return validatePCAPngSHB(data)
	default:
		return fmt.Errorf(
			"%w: 0x%08x (expected pcap or pcapng format)",
			ErrInvalidPCAPMagicNumber,
			magic,
		)
	}
}

// validatePCAPGlobalHeader validates the pcap global header structure (24 bytes).
func validatePCAPGlobalHeader(data []byte, bigEndian bool) error {
	const pcapGlobalHeaderLen = 24

	if len(data) < pcapGlobalHeaderLen {
		return fmt.Errorf("%w: file too small for pcap global header (need %d bytes, got %d)",
			ErrInvalidPCAPMagicNumber, pcapGlobalHeaderLen, len(data))
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
			ErrInvalidPCAPMagicNumber, versionMajor, versionMinor)
	}

	// Validate link-layer type
	if !isValidLinkType(linkType) {
		return fmt.Errorf("%w: unknown link-layer type %d",
			ErrInvalidPCAPMagicNumber, linkType)
	}

	return nil
}

// validatePCAPngSHB validates the pcapng Section Header Block.
func validatePCAPngSHB(data []byte) error {
	const minSHBLen = 28

	if len(data) < minSHBLen {
		return fmt.Errorf(
			"%w: file too small for pcapng Section Header Block (need %d bytes, got %d)",
			ErrInvalidPCAPMagicNumber,
			minSHBLen,
			len(data),
		)
	}

	// Bytes 8-11 contain Byte-Order Magic to determine endianness
	bom := uint32(data[8])<<24 | uint32(data[9])<<16 | uint32(data[10])<<8 | uint32(data[11])
	if bom != 0x1a2b3c4d && bom != 0x4d3c2b1a {
		return fmt.Errorf("%w: invalid pcapng byte-order magic 0x%08x",
			ErrInvalidPCAPMagicNumber, bom)
	}

	return nil
}

// processInlineData decodes and validates inline PCAP data, writes it to a temp file.
func (s *Server) processInlineData(inlineData string) (string, error) {
	// SECURITY FIX #97: Additional check on base64 encoded data size
	if len(inlineData) > MaxPCAPUploadSize*4/base64Ratio {
		return "", ErrPCAPDataExceedsSizeLimit
	}

	data, decodeErr := base64.StdEncoding.DecodeString(inlineData)
	if decodeErr != nil {
		return "", fmt.Errorf("decode replay data: %w", decodeErr)
	}

	// Double-check decoded size
	if len(data) > MaxPCAPUploadSize {
		return "", ErrDecodedPCAPExceedsSizeLimit
	}

	// SECURITY FIX LOW-2: Validate PCAP file magic number
	if magicErr := validatePCAPStructure(data); magicErr != nil {
		return "", fmt.Errorf("invalid PCAP file: %w", magicErr)
	}

	return s.writeUploadedFile(data)
}

func (s *Server) prepareReplayRequest(req ReplayRequest) (ReplayRequest, error) {
	if strings.TrimSpace(req.File) == "" && req.InlineData == "" {
		return req, ErrPcapFilePathOrDataRequired
	}

	if req.InlineData == "" {
		// SECURITY FIX #162: Validate PCAP file path to prevent arbitrary file access
		validatedPath, err := s.validatePcapFilePath(req.File)
		if err != nil {
			return req, err
		}

		req.File = validatedPath

		return req, nil
	}

	path, err := s.processInlineData(req.InlineData)
	if err != nil {
		return req, err
	}

	req.File = path
	req.Uploaded = true
	req.InlineData = ""

	return req, nil
}

// SECURITY FIX #162: validatePcapFilePath ensures the file path is safe.
func (s *Server) validatePcapFilePath(filename string) (string, error) {
	if filename == "" {
		return "", errors.New("filename cannot be empty")
	}

	cleanPath := filepath.Clean(filename)

	if strings.ContainsRune(cleanPath, 0) {
		return "", errors.New("filename contains invalid characters")
	}

	cfgPath := s.configPath()
	var allowedDir string
	if cfgPath != "" {
		allowedDir = filepath.Dir(cfgPath)
	} else {
		var err error
		allowedDir, err = os.Getwd()
		if err != nil {
			return "", fmt.Errorf("cannot determine allowed directory: %w", err)
		}
	}

	var absPath string
	if !filepath.IsAbs(cleanPath) {
		absPath = filepath.Join(allowedDir, cleanPath)
	} else {
		absPath = cleanPath
	}

	absPath = filepath.Clean(absPath)
	realPath, err := filepath.EvalSymlinks(absPath)
	if err != nil {
		realPath = absPath
	}

	realAllowedDir, err := filepath.EvalSymlinks(allowedDir)
	if err != nil {
		realAllowedDir = allowedDir
	}

	if !strings.HasPrefix(realPath, realAllowedDir+string(filepath.Separator)) &&
		realPath != realAllowedDir {
		return "", fmt.Errorf("access denied: file must be within %s", allowedDir)
	}

	info, err := os.Stat(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("pcap file not found: %s", filename)
		}
		return "", fmt.Errorf("cannot access pcap file: %w", err)
	}
	if info.IsDir() {
		return "", fmt.Errorf("%w: %s", ErrPathIsADirectory, absPath)
	}

	ext := strings.ToLower(filepath.Ext(absPath))
	validExts := map[string]bool{".pcap": true, ".pcapng": true, ".cap": true}
	if !validExts[ext] {
		return "", fmt.Errorf(
			"invalid pcap file extension: %s (allowed: .pcap, .pcapng, .cap)",
			ext,
		)
	}

	return absPath, nil
}

func (s *Server) writeUploadedFile(data []byte) (string, error) {
	// SECURITY FIX #167: Use restrictive permissions for temp directory (owner-only)
	dir := filepath.Join(os.TempDir(), "niac-replay")

	mkdirErr := os.MkdirAll(dir, 0o700)
	if mkdirErr != nil {
		return "", fmt.Errorf("create upload dir: %w", mkdirErr)
	}

	const secureDirPerm = 0o700

	chmodErr := os.Chmod(dir, secureDirPerm)
	if chmodErr != nil {
		return "", fmt.Errorf("secure upload dir: %w", chmodErr)
	}

	tmp, createErr := os.CreateTemp(dir, "upload-*.pcap")
	if createErr != nil {
		return "", fmt.Errorf("create temp file: %w", createErr)
	}

	tmpPath := tmp.Name()

	// SECURITY FIX #167: Write data while file is still open (no race window)
	_, writeErr := tmp.Write(data)
	if writeErr != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)

		return "", fmt.Errorf("write upload: %w", writeErr)
	}

	syncErr := tmp.Sync()
	if syncErr != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)

		return "", fmt.Errorf("sync upload: %w", syncErr)
	}

	closeErr := tmp.Close()
	if closeErr != nil {
		_ = os.Remove(tmpPath)

		return "", fmt.Errorf("close temp file: %w", closeErr)
	}

	return tmpPath, nil
}
