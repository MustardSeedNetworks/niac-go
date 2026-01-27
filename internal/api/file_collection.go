package api

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

// FileEntry represents a discovered file (pcap, walk, etc.).
type FileEntry struct {
	Path      string    `json:"path"`
	Name      string    `json:"name"`
	SizeBytes int64     `json:"size_bytes"`
	Modified  time.Time `json:"modified_at"`
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

	return walkAndCollectFiles(root, rootReal, exts)
}

func walkAndCollectFiles(root, rootReal string, exts []string) ([]FileEntry, error) {
	var entries []FileEntry
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil || d.IsDir() {
			return walkErr
		}
		if !slices.Contains(exts, strings.ToLower(filepath.Ext(path))) {
			return nil
		}
		entry, procErr := processFileEntry(path, d, rootReal)
		if procErr != nil {
			if errors.Is(procErr, ErrPathOutsideRoot) {
				return nil
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
