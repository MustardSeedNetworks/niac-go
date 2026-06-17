package api

import (
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/MustardSeedNetworks/niac-go/internal/api/capture"
)

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
	if magicErr := capture.ValidateStructure(data); magicErr != nil {
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
// Pcaps are resolved in two passes (mirroring validateWalkFilePath
// in #558): first the legacy config-dir / cwd allow-list, then the
// unified library pcaps dir (~/.niac/library/pcaps/) so the new
// picker integrations from #556 can send library-relative names.
//
// Per-dir checks live in tryPcapInDir so this top-level function
// stays under the cognitive-complexity cap. Soft failures (access
// denied / not found) propagate via lastErr and fall through to the
// next allowed dir; hard failures (extension, null byte, dir-not-file)
// short-circuit out.
func (s *Server) validatePcapFilePath(filename string) (string, error) {
	if filename == "" {
		return "", errors.New("filename cannot be empty")
	}

	cleanPath := filepath.Clean(filename)
	if strings.ContainsRune(cleanPath, 0) {
		return "", errors.New("filename contains invalid characters")
	}

	allowedDirs, err := s.resolvePcapAllowedDirs()
	if err != nil {
		return "", err
	}

	var lastErr error
	for _, allowedDir := range allowedDirs {
		absPath, hardFail, attemptErr := tryPcapInDir(cleanPath, filename, allowedDir)
		if hardFail {
			return "", attemptErr
		}
		if attemptErr != nil {
			lastErr = attemptErr
			continue
		}
		return absPath, nil
	}

	if lastErr != nil {
		return "", lastErr
	}
	return "", fmt.Errorf("pcap file not found: %s", filename)
}

// tryPcapInDir resolves cleanPath against allowedDir and returns the
// validated absolute path on success. The second return is "hard
// fail" — true when the error is dir-independent (extension, null
// byte, dir-not-file) and the caller should stop trying other dirs.
func tryPcapInDir(cleanPath, filename, allowedDir string) (string, bool, error) {
	absPath := cleanPath
	if !filepath.IsAbs(cleanPath) {
		absPath = filepath.Join(allowedDir, cleanPath)
	}
	absPath = filepath.Clean(absPath)

	// Inline barrier on the EvalSymlinks/Stat sinks below so static
	// analysers see the absolute path is bounded before any filesystem
	// access.
	if !filepath.IsAbs(absPath) || strings.Contains(absPath, "..") {
		return "", true, errors.New("pcap file path failed safety check")
	}

	realPath, evalErr := filepath.EvalSymlinks(absPath)
	if evalErr != nil {
		realPath = absPath
	}
	realAllowedDir, evalDirErr := filepath.EvalSymlinks(allowedDir)
	if evalDirErr != nil {
		realAllowedDir = allowedDir
	}
	if !strings.HasPrefix(realPath, realAllowedDir+string(filepath.Separator)) &&
		realPath != realAllowedDir {
		return "", false, fmt.Errorf("access denied: file must be within %s", allowedDir)
	}

	info, statErr := os.Stat(absPath)
	if statErr != nil {
		if os.IsNotExist(statErr) {
			return "", false, fmt.Errorf("pcap file not found: %s", filename)
		}
		return "", true, fmt.Errorf("cannot access pcap file: %w", statErr)
	}
	if info.IsDir() {
		return "", true, fmt.Errorf("%w: %s", ErrPathIsADirectory, absPath)
	}

	ext := strings.ToLower(filepath.Ext(absPath))
	validExts := map[string]bool{".pcap": true, ".pcapng": true, ".cap": true}
	if !validExts[ext] {
		return "", true, fmt.Errorf(
			"invalid pcap file extension: %s (allowed: .pcap, .pcapng, .cap)",
			ext,
		)
	}
	return absPath, false, nil
}

// resolvePcapAllowedDirs returns the ordered list of directories pcap
// files may live under. Mirrors resolveWalkAllowedDirs in walk.go —
// config-dir / cwd first, library pcaps dir second when the library
// opened cleanly. Library failure stays non-fatal: clients that need
// pcap content from the library will already hit a 503 on
// /api/v1/library/pcaps, so silently omitting it here matches the
// rest of the daemon's posture.
func (s *Server) resolvePcapAllowedDirs() ([]string, error) {
	var dirs []string

	cfgPath := s.configPath()
	if cfgPath != "" {
		dirs = append(dirs, filepath.Dir(cfgPath))
	} else {
		cwd, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("cannot determine allowed directory: %w", err)
		}
		dirs = append(dirs, cwd)
	}

	if s.library != nil {
		dirs = append(dirs, s.library.SubDir("pcaps"))
	}

	return dirs, nil
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
