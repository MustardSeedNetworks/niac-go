// Package content owns the on-disk content bundle: download, verify,
// extract, and inventory. The bundle is a gzip-compressed tar that
// mirrors the library layout (networks/, walks/, pcaps/) and is
// extracted into a library root resolved by internal/library.
//
// Two CLI surfaces consume this package:
//
//   - `niac content install` (cmd/niac/content.go) — local or remote
//     install of the versioned bundle.
//   - The daemon's POST /api/v1/library/install handler (PR 3) —
//     UI uploads of the same tarball format.
//
// Security: every path coming out of the tarball is normalised and
// re-rooted under <library>/<kind>/ before any file is touched, so a
// crafted bundle can't drop files outside the library or follow a
// symlink to /etc/passwd.
package content

import (
	"archive/tar"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/krisarmstrong/niac-go/internal/library"
)

// Sentinel errors callers can match on.
var (
	ErrUnknownKind     = errors.New("entry references an unknown library kind")
	ErrPathEscape      = errors.New("entry path escapes the library root")
	ErrUnsafeFileMode  = errors.New("entry would have been created with an unsafe mode")
	ErrUnsupportedType = errors.New("entry is neither a regular file nor a directory")
	ErrEntryTooLarge   = errors.New("entry exceeds the maximum allowed size")
	ErrBundleTooLarge  = errors.New("bundle exceeds the maximum total uncompressed size")
)

// Extraction caps. Picked to fit the largest realistic bundle (≈1.2 GB
// of sanitised walks per the issue) with headroom, while still
// preventing a malicious tar from filling the disk.
const (
	maxEntrySize  int64 = 256 << 20 // 256 MiB per file
	maxBundleSize int64 = 4 << 30   // 4 GiB total uncompressed
)

// topLevelKindParts is the SplitN count used to peel the "kind/"
// prefix off a tar entry path: 2 → ["networks", "rest/of/path"].
const topLevelKindParts = 2

// ExtractOptions tunes Extract behaviour. The zero value is the
// sensible default: extract everything, overwrite existing files,
// don't dry-run.
type ExtractOptions struct {
	// DryRun reports what would be installed without touching the disk.
	DryRun bool
	// SkipExisting flips the default overwrite-on-conflict behaviour to
	// "skip existing files" (additive merge). The zero value (false)
	// overwrites, so reinstalling a newer bundle replaces older
	// content — which is what `niac content update` wants.
	SkipExisting bool
}

// Manifest summarises an extraction. Returned by Extract; printed by
// `niac content install` so the user sees what landed.
type Manifest struct {
	Files       int
	Directories int
	Bytes       int64
	// PerKind reports the count of entries dropped under each library
	// kind directory. Useful for the install summary line.
	PerKind map[library.Kind]int
}

// Extract reads a gzip-tar bundle from r and writes its contents into
// libRoot/<kind>/<entry-path>. Top-level directories in the tar must
// be one of the known library kinds (networks/, walks/, pcaps/);
// anything else is rejected with ErrUnknownKind.
func Extract(r io.Reader, libRoot string, opts ExtractOptions) (Manifest, error) {
	abs, absErr := filepath.Abs(libRoot)
	if absErr != nil {
		return Manifest{}, fmt.Errorf("resolve library root: %w", absErr)
	}

	gz, gzErr := gzip.NewReader(r)
	if gzErr != nil {
		return Manifest{}, fmt.Errorf("open gzip stream: %w", gzErr)
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	manifest := Manifest{PerKind: map[library.Kind]int{}}
	overwrite := !opts.SkipExisting

	var totalBytes int64
	for {
		hdr, hdrErr := tr.Next()
		if errors.Is(hdrErr, io.EOF) {
			break
		}
		if hdrErr != nil {
			return manifest, fmt.Errorf("read tar entry: %w", hdrErr)
		}

		if procErr := processEntry(tr, hdr, abs, opts, overwrite, &manifest, &totalBytes); procErr != nil {
			return manifest, procErr
		}
	}

	return manifest, nil
}

// processEntry handles a single tar entry: resolves its destination,
// validates type/size, and writes it (unless DryRun). Pulled out of
// Extract to keep the loop body under the cognitive-complexity cap.
func processEntry(
	tr *tar.Reader,
	hdr *tar.Header,
	libRoot string,
	opts ExtractOptions,
	overwrite bool,
	manifest *Manifest,
	totalBytes *int64,
) error {
	dest, kind, skipReason, err := resolveEntryPath(libRoot, hdr.Name)
	if err != nil {
		return err
	}
	if skipReason != "" {
		// Entries like the bare "networks/" directory header carry no
		// payload and don't need to land on disk — ensureLayout already
		// created them. Silently skip.
		return nil
	}

	switch hdr.Typeflag {
	case tar.TypeDir:
		if !opts.DryRun {
			if mkErr := os.MkdirAll(dest, libraryDirMode); mkErr != nil {
				return fmt.Errorf("mkdir %s: %w", dest, mkErr)
			}
		}
		manifest.Directories++
		return nil

	case tar.TypeReg:
		if hdr.Size < 0 || hdr.Size > maxEntrySize {
			return fmt.Errorf("%w: %s is %d bytes", ErrEntryTooLarge, hdr.Name, hdr.Size)
		}
		*totalBytes += hdr.Size
		if *totalBytes > maxBundleSize {
			return fmt.Errorf("%w (running total %d bytes)", ErrBundleTooLarge, *totalBytes)
		}
		if !opts.DryRun {
			if writeErr := writeFile(dest, tr, hdr.Size, overwrite); writeErr != nil {
				return writeErr
			}
		}
		manifest.Files++
		manifest.Bytes += hdr.Size
		manifest.PerKind[kind]++
		return nil

	default:
		return fmt.Errorf("%w: %s (typeflag %d)", ErrUnsupportedType, hdr.Name, hdr.Typeflag)
	}
}

// resolveEntryPath validates a tar entry path against the library
// layout and returns its on-disk destination plus the kind it belongs
// to. A non-empty skipReason means the entry is structurally valid but
// carries no payload worth writing (e.g. the bare "networks/" header).
func resolveEntryPath(libRoot, name string) (string, library.Kind, string, error) {
	clean := strings.TrimPrefix(filepath.ToSlash(name), "./")
	clean = strings.TrimPrefix(clean, "/")
	if clean == "" || clean == "." {
		return "", "", "empty entry name", nil
	}

	if pathEscapes(clean) {
		return "", "", "", fmt.Errorf("%w: %s", ErrPathEscape, name)
	}

	parts := strings.SplitN(clean, "/", topLevelKindParts)
	top := parts[0]
	kind, knownKind := matchKind(top)
	if !knownKind {
		return "", "", "", fmt.Errorf("%w: top-level %q from entry %q", ErrUnknownKind, top, name)
	}

	if len(parts) == 1 {
		// Bare kind directory header, no payload.
		return "", kind, "bare kind directory", nil
	}

	dest := filepath.Join(libRoot, top, filepath.FromSlash(parts[1]))
	prefix := filepath.Join(libRoot, top) + string(filepath.Separator)
	if !strings.HasPrefix(dest+string(filepath.Separator), prefix) && dest != filepath.Join(libRoot, top) {
		return "", "", "", fmt.Errorf("%w: %s resolved to %s", ErrPathEscape, name, dest)
	}
	return dest, kind, "", nil
}

// pathEscapes returns true when the normalised tar entry path contains
// a `..` component that would let it leave its kind directory.
func pathEscapes(clean string) bool {
	if clean == ".." {
		return true
	}
	if strings.HasPrefix(clean, "../") {
		return true
	}
	if strings.Contains(clean, "/../") {
		return true
	}
	return strings.HasSuffix(clean, "/..")
}

func matchKind(top string) (library.Kind, bool) {
	for _, k := range library.AllKinds() {
		if string(k) == top {
			return k, true
		}
	}
	return "", false
}

// writeFile creates or overwrites dest with size bytes from r. Always
// writes via a temp file in the same dir then atomically renames so a
// crash mid-extract leaves the previous version intact.
func writeFile(dest string, r io.Reader, size int64, overwrite bool) error {
	if !overwrite {
		if _, statErr := os.Stat(dest); statErr == nil {
			return nil
		}
	}

	if mkErr := os.MkdirAll(filepath.Dir(dest), libraryDirMode); mkErr != nil {
		return fmt.Errorf("mkdir parent of %s: %w", dest, mkErr)
	}

	tmp, createErr := os.CreateTemp(filepath.Dir(dest), ".content-*.tmp")
	if createErr != nil {
		return fmt.Errorf("create temp for %s: %w", dest, createErr)
	}
	tmpPath := tmp.Name()
	cleanup := func() { _ = os.Remove(tmpPath) }

	if _, copyErr := io.CopyN(tmp, r, size); copyErr != nil {
		_ = tmp.Close()
		cleanup()
		return fmt.Errorf("copy bytes for %s: %w", dest, copyErr)
	}
	if chmodErr := tmp.Chmod(libraryFileMode); chmodErr != nil {
		_ = tmp.Close()
		cleanup()
		return fmt.Errorf("chmod temp for %s: %w", dest, chmodErr)
	}
	if closeErr := tmp.Close(); closeErr != nil {
		cleanup()
		return fmt.Errorf("close temp for %s: %w", dest, closeErr)
	}
	if renameErr := os.Rename(tmpPath, dest); renameErr != nil {
		cleanup()
		return fmt.Errorf("rename %s -> %s: %w", tmpPath, dest, renameErr)
	}
	return nil
}

// File modes for files/dirs we create. Match internal/library so the
// extracted content sits comfortably alongside hand-dropped files.
const (
	libraryDirMode  = 0o755
	libraryFileMode = 0o644
)
