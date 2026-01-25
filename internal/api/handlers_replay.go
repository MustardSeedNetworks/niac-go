package api

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

func (s *Server) handleReplay(w http.ResponseWriter, r *http.Request) {
	// FEATURE #132: Graceful degradation when replay engine is unavailable
	if s.cfg.Replay == nil {
		writeError(
			w,
			r,
			http.StatusServiceUnavailable,
			"replay_unavailable",
			"PCAP replay functionality is not available in this mode. Start niac with a configuration to enable replay.",
			nil,
		)

		return
	}

	switch r.Method {
	case http.MethodGet:
		s.writeJSON(w, s.cfg.Replay.Status())
	case http.MethodPost:
		// SECURITY FIX #97: Enforce request body size limit for PCAP uploads
		r.Body = http.MaxBytesReader(w, r.Body, MaxPCAPUploadSize)

		var req ReplayRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			if err.Error() == ErrMsgRequestBodyTooLarge {
				writeError(w, r, http.StatusRequestEntityTooLarge, "request_too_large",
					"PCAP file too large (max 100MB)", nil)

				return
			}

			writeError(w, r, http.StatusBadRequest, "invalid_request",
				"Failed to parse request body", nil)

			return
		}

		// SECURITY FIX MEDIUM-3: Comprehensive validation
		if validationErrors := validateReplayRequest(req); len(validationErrors) > 0 {
			writeError(w, r, http.StatusBadRequest, "validation_failed",
				"Replay request validation failed", validationErrors)

			return
		}

		prepared, err := s.prepareReplayRequest(req)
		if err != nil {
			writeError(w, r, http.StatusBadRequest, "replay_preparation_failed",
				"Failed to prepare replay request", nil)

			return
		}

		state, err := s.cfg.Replay.Start(prepared)
		if err != nil {
			writeError(w, r, http.StatusBadRequest, "replay_start_failed",
				"Failed to start replay", nil)

			return
		}

		s.writeJSON(w, state)
	case http.MethodDelete:
		state, err := s.cfg.Replay.Stop()
		if err != nil {
			// SECURITY FIX MEDIUM-6: Don't expose internal error details
			s.logger.Error("[API] Failed to stop replay", "error", err)
			writeError(w, r, http.StatusInternalServerError, "replay_stop_failed",
				"Failed to stop replay", nil)

			return
		}

		s.writeJSON(w, state)
	default:
		w.Header().Set("Allow", "GET, POST, DELETE")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleFiles(w http.ResponseWriter, r *http.Request) {
	kind := r.URL.Query().Get("kind")

	// SECURITY FIX MEDIUM-3: Validate query parameter
	allowedKinds := []string{"", "snmp", "config", "pcap", "walks", "pcaps"}
	if err := validateQueryParam("kind", kind, allowedKinds); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_parameter",
			"Invalid query parameter", []ErrorDetail{*err})

		return
	}

	entries, err := s.collectFiles(kind)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "file_collection_failed",
			"Failed to collect files", nil)

		return
	}

	s.writeJSON(w, entries)
}

// validatePCAPMagic validates that the file begins with a valid PCAP magic number
// SECURITY FIX LOW-2: Prevents processing of non-PCAP files that could exploit parser bugs.
func validatePCAPMagic(data []byte) error {
	if len(data) < minPCAPSize {
		return ErrFileTooSmallForPCAP
	}

	// Check for valid PCAP magic numbers
	// 0xa1b2c3d4 = standard pcap (microsecond precision, big-endian)
	// 0xd4c3b2a1 = standard pcap (microsecond precision, little-endian)
	// 0xa1b23c4d = pcap with nanosecond precision (big-endian)
	// 0x4d3cb2a1 = pcap with nanosecond precision (little-endian)
	// 0x0a0d0d0a = pcapng format
	magic := uint32(data[0])<<24 | uint32(data[1])<<16 | uint32(data[2])<<8 | uint32(data[3])

	validMagics := []uint32{
		0xa1b2c3d4, // pcap microsecond BE
		0xd4c3b2a1, // pcap microsecond LE
		0xa1b23c4d, // pcap nanosecond BE
		0x4d3cb2a1, // pcap nanosecond LE
		0x0a0d0d0a, // pcapng
	}

	if slices.Contains(validMagics, magic) {
		return nil
	}

	return fmt.Errorf(
		"%w: 0x%08x (expected pcap or pcapng format)",
		ErrInvalidPCAPMagicNumber,
		magic,
	)
}

// processInlineData decodes and validates inline PCAP data, writes it to a temp file.
// Returns the file path and any error encountered.
func (s *Server) processInlineData(inlineData string) (string, error) {
	// SECURITY FIX #97: Additional check on base64 encoded data size
	// Base64 encoding increases size by ~4/3, so check before decode
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
	if magicErr := validatePCAPMagic(data); magicErr != nil {
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

// SECURITY FIX #162: validatePcapFilePath ensures the file path is safe and doesn't traverse outside allowed directories.
func (s *Server) validatePcapFilePath(filename string) (string, error) {
	// Empty filename is invalid
	if filename == "" {
		return "", errors.New("filename cannot be empty")
	}

	// Clean the path to normalize it
	cleanPath := filepath.Clean(filename)

	// Reject paths containing null bytes (potential bypass attempt)
	if strings.ContainsRune(cleanPath, 0) {
		return "", errors.New("filename contains invalid characters")
	}

	// Get config directory as the allowed base directory
	cfgPath := s.configPath()
	var allowedDir string
	if cfgPath != "" {
		allowedDir = filepath.Dir(cfgPath)
	} else {
		// If no config path, use current working directory
		var err error
		allowedDir, err = os.Getwd()
		if err != nil {
			return "", fmt.Errorf("cannot determine allowed directory: %w", err)
		}
	}

	// Resolve to absolute path
	var absPath string
	if !filepath.IsAbs(cleanPath) {
		absPath = filepath.Join(allowedDir, cleanPath)
	} else {
		absPath = cleanPath
	}

	// Clean the absolute path and resolve symlinks for security
	absPath = filepath.Clean(absPath)
	realPath, err := filepath.EvalSymlinks(absPath)
	if err != nil {
		// If symlink resolution fails, file may not exist yet or symlink is broken
		realPath = absPath
	}

	// Resolve allowed directory to real path for comparison
	realAllowedDir, err := filepath.EvalSymlinks(allowedDir)
	if err != nil {
		realAllowedDir = allowedDir
	}

	// Verify the file is within the allowed directory (prevent path traversal)
	if !strings.HasPrefix(realPath, realAllowedDir+string(filepath.Separator)) &&
		realPath != realAllowedDir {
		return "", fmt.Errorf("access denied: file must be within %s", allowedDir)
	}

	// Verify the path exists and is a file (not a directory)
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

	// Verify file has a valid pcap extension
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

	// Ensure directory permissions are correct even if it already exists.
	// 0o700 = owner rwx only, restrictive for upload directories.
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

	// Close file but don't use defer - we need to return path on success
	closeErr := tmp.Close()
	if closeErr != nil {
		_ = os.Remove(tmpPath)

		return "", fmt.Errorf("close temp file: %w", closeErr)
	}

	return tmpPath, nil
}

// getFileKindConfig returns root path and extensions for a file kind.
func (s *Server) getFileKindConfig(kind string) (string, []string, error) {
	switch kind {
	case "walks":
		return s.resolveIncludePath(), []string{".walk"}, nil
	case "pcaps":
		if cfgPath := s.configPath(); cfgPath != "" {
			return filepath.Dir(cfgPath), []string{".pcap", ".pcapng"}, nil
		}

		return "", []string{".pcap", ".pcapng"}, nil
	default:
		return "", nil, fmt.Errorf("%w: %s", ErrUnsupportedFileKind, kind)
	}
}

// resolveSecureRoot validates and resolves the root path for file collection.
func resolveSecureRoot(root string) (string, error) {
	rootInfo, statErr := os.Stat(root)
	if statErr != nil {
		return "", statErr
	}

	if !rootInfo.IsDir() {
		return "", ErrNotADirectory
	}

	rootAbs, absErr := filepath.Abs(root)
	if absErr != nil {
		return "", fmt.Errorf("failed to resolve root path: %w", absErr)
	}

	// Fall back to absolute path if symlink resolution fails
	rootReal, _ := filepath.EvalSymlinks(rootAbs)
	if rootReal == "" {
		return rootAbs, nil
	}

	return rootReal, nil
}

// processFileEntry validates and creates a FileEntry for a collected file.
func processFileEntry(path string, d fs.DirEntry, rootReal string) (*FileEntry, error) {
	info, infoErr := d.Info()
	if infoErr != nil {
		return nil, fmt.Errorf("failed to get file info: %w", infoErr)
	}

	absPath, absErr := filepath.Abs(path)
	if absErr != nil {
		return nil, fmt.Errorf("failed to get absolute path: %w", absErr)
	}

	// SECURITY FIX #95: Validate path stays within root directory
	realPath, evalErr := filepath.EvalSymlinks(absPath)
	if evalErr != nil {
		return nil, fmt.Errorf("failed to evaluate symlinks: %w", evalErr)
	}

	// Ensure resolved path is within the allowed root directory
	if !strings.HasPrefix(realPath, rootReal+string(os.PathSeparator)) && realPath != rootReal {
		return nil, ErrPathOutsideRoot
	}

	return &FileEntry{
		Path:      absPath,
		Name:      filepath.Base(path),
		SizeBytes: info.Size(),
		Modified:  info.ModTime().UTC(),
	}, nil
}

func (s *Server) collectFiles(kind string) ([]FileEntry, error) {
	root, exts, err := s.getFileKindConfig(kind)
	if err != nil {
		return nil, err
	}

	if root == "" {
		return []FileEntry{}, nil
	}

	rootReal, err := resolveSecureRoot(root)
	if err != nil {
		if errors.Is(err, ErrNotADirectory) {
			return []FileEntry{}, nil
		}

		return []FileEntry{}, fmt.Errorf("failed to stat root: %w", err)
	}

	var entries []FileEntry

	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		if d.IsDir() {
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		if !slices.Contains(exts, ext) {
			return nil
		}

		entry, procErr := processFileEntry(path, d, rootReal)
		if procErr != nil {
			if errors.Is(procErr, ErrPathOutsideRoot) {
				return nil // Skip files outside allowed directory
			}

			return procErr
		}

		entries = append(entries, *entry)
		if len(entries) >= maxFileEntries {
			return filepath.SkipDir
		}

		return nil
	})
	if err != nil && !errors.Is(err, filepath.SkipDir) {
		return nil, fmt.Errorf("failed to walk directory: %w", err)
	}

	return entries, nil
}

func (s *Server) resolveIncludePath() string {
	cfg := s.currentConfig()
	if cfg == nil || cfg.IncludePath == "" {
		return ""
	}

	includePath := cfg.IncludePath
	// SECURITY FIX #161: Thread-safe access to ConfigPath
	if cfgPath := s.configPath(); !filepath.IsAbs(includePath) && cfgPath != "" {
		includePath = filepath.Join(filepath.Dir(cfgPath), includePath)
	}

	if abs, err := filepath.Abs(includePath); err == nil {
		return abs
	}

	return includePath
}
