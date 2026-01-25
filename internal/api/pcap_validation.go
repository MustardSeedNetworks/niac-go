package api

import (
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Known PCAP link-layer types for validation.
// SECURITY FIX #170: Only accept recognized link-layer types.
var validLinkTypes = map[uint32]bool{
	0:   true, // NULL/Loopback
	1:   true, // Ethernet
	6:   true, // Token Ring
	8:   true, // SLIP
	9:   true, // PPP
	12:  true, // Raw IP (BSD)
	101: true, // Raw IP
	105: true, // IEEE 802.11
	113: true, // Linux cooked capture
	127: true, // IEEE 802.11 radiotap
	140: true, // MTP2 with pseudo-header
	147: true, // 802.11 with AVS header
	162: true, // Raw IP (DLT_RAW on OpenBSD)
	163: true, // NFLOG
	187: true, // Bluetooth HCI
	189: true, // USB 2.0/1.1/1.0 packets
	195: true, // IEEE 802.15.4
	204: true, // PPP
	228: true, // Raw IPv4
	229: true, // Raw IPv6
	253: true, // NFQUEUE
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
	case 0xa1b2c3d4, 0xa1b23c4d:
		// Big-endian pcap format
		return validatePCAPGlobalHeader(data, true)
	case 0xd4c3b2a1, 0x4d3cb2a1:
		// Little-endian pcap format
		return validatePCAPGlobalHeader(data, false)
	case 0x0a0d0d0a:
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
		versionMajor = uint32(data[4])<<8 | uint32(data[5])
		versionMinor = uint32(data[6])<<8 | uint32(data[7])
		linkType = uint32(data[20])<<24 | uint32(data[21])<<16 | uint32(data[22])<<8 | uint32(data[23])
	} else {
		versionMajor = uint32(data[5])<<8 | uint32(data[4])
		versionMinor = uint32(data[7])<<8 | uint32(data[6])
		linkType = uint32(data[23])<<24 | uint32(data[22])<<16 | uint32(data[21])<<8 | uint32(data[20])
	}

	// Validate version (must be 2.4)
	if versionMajor != 2 || versionMinor != 4 {
		return fmt.Errorf("%w: unsupported pcap version %d.%d (expected 2.4)",
			ErrInvalidPCAPMagicNumber, versionMajor, versionMinor)
	}

	// Validate link-layer type
	if !validLinkTypes[linkType] {
		return fmt.Errorf("%w: unknown link-layer type %d",
			ErrInvalidPCAPMagicNumber, linkType)
	}

	return nil
}

// validatePCAPngSHB validates the pcapng Section Header Block.
func validatePCAPngSHB(data []byte) error {
	const minSHBLen = 28

	if len(data) < minSHBLen {
		return fmt.Errorf("%w: file too small for pcapng Section Header Block (need %d bytes, got %d)",
			ErrInvalidPCAPMagicNumber, minSHBLen, len(data))
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
