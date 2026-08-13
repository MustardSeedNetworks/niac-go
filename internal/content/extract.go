// Package content owns the on-disk content bundle: extract and
// inventory. The bundle is a gzip-compressed tar that mirrors the
// library layout (networks/, walks/, pcaps/) and is extracted into a
// library root resolved by internal/library. Bundles are always local
// — embedded essentials, the niac-content deb/rpm package, or a UI
// upload — niac never fetches content over the network.
//
// Two surfaces consume this package:
//
//   - `niac content install --bundle` (cmd/niac/content.go) — install
//     of a local bundle file.
//   - The daemon's POST /api/v1/library/install handler (PR 3) —
//     UI uploads of the same tarball format.
//
// Security: the library root is opened as an [os.Root] and every path
// coming out of the tarball is written relative to it. The kernel
// rejects any entry that would escape the root — including one that
// traverses a symlink already present on disk — so a crafted bundle
// cannot drop files outside the library or follow a symlink to
// /etc/passwd. A cheap pathEscapes() pre-check rejects obvious ../
// entries early with ErrPathEscape; os.Root is the real containment.
package content

import (
	"archive/tar"
	"compress/gzip"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/MustardSeedNetworks/niac-go/internal/library"
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
	// content.
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
	// Paths lists what was written, relative to the library root. A bundle's
	// files are indistinguishable from the operator's own once they are on
	// disk, so this is what lets them keep being reported as bundle content.
	Paths []string
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

	// Open the library root as an os.Root so every write below is
	// contained by the kernel, not a string prefix check. The directory
	// must exist to be opened, so create it first (idempotent).
	if mkErr := os.MkdirAll(abs, libraryDirMode); mkErr != nil {
		return Manifest{}, fmt.Errorf("create library root: %w", mkErr)
	}
	root, rootErr := os.OpenRoot(abs)
	if rootErr != nil {
		return Manifest{}, fmt.Errorf("open library root: %w", rootErr)
	}
	defer func() { _ = root.Close() }()

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

		if procErr := processEntry(tr, hdr, root, opts, overwrite, &manifest, &totalBytes); procErr != nil {
			return manifest, procErr
		}
	}

	if !opts.DryRun {
		if recErr := library.RecordBundleInstall(abs, manifest.Paths); recErr != nil {
			return manifest, fmt.Errorf("record bundle contents: %w", recErr)
		}
	}

	return manifest, nil
}

// processEntry handles a single tar entry: resolves its destination,
// validates type/size, and writes it (unless DryRun). Pulled out of
// Extract to keep the loop body under the cognitive-complexity cap.
//
// The path-traversal defence is layered: resolveEntryPath rejects any
// entry with a ../ component or leading / (ErrPathEscape) and returns
// a path relative to the library root. Every filesystem call then goes
// through os.Root, which resolves that relative path inside the root
// and has the kernel reject anything that escapes — including a
// traversal through a symlink already on disk, which the old string
// prefix check could not catch (a TOCTOU window between check and open).
func processEntry(
	tr *tar.Reader,
	hdr *tar.Header,
	root *os.Root,
	opts ExtractOptions,
	overwrite bool,
	manifest *Manifest,
	totalBytes *int64,
) error {
	rel, kind, skipReason, err := resolveEntryPath(hdr.Name)
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
			if mkErr := root.MkdirAll(rel, libraryDirMode); mkErr != nil {
				return fmt.Errorf("mkdir %s: %w", rel, mkErr)
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
			if writeErr := writeFile(root, rel, tr, hdr.Size, overwrite); writeErr != nil {
				return writeErr
			}
		}
		manifest.Files++
		manifest.Bytes += hdr.Size
		manifest.PerKind[kind]++
		manifest.Paths = append(manifest.Paths, rel)
		return nil

	default:
		return fmt.Errorf("%w: %s (typeflag %d)", ErrUnsupportedType, hdr.Name, hdr.Typeflag)
	}
}

// resolveEntryPath validates a tar entry path against the library
// layout and returns its destination RELATIVE to the library root plus
// the kind it belongs to. A non-empty skipReason means the entry is
// structurally valid but carries no payload worth writing (e.g. the
// bare "networks/" header).
func resolveEntryPath(name string) (string, library.Kind, string, error) {
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

	// Path relative to the library root; os.Root resolves and contains it
	// at every filesystem call in processEntry / writeFile.
	rel := filepath.Join(top, filepath.FromSlash(parts[1]))
	return rel, kind, "", nil
}

// pathEscapes returns true when the normalised tar entry path contains
// a `..` component that would let it leave its kind directory. This is a
// cheap early reject that yields the ErrPathEscape sentinel; os.Root is
// the authoritative containment at write time.
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

// writeFile creates or overwrites rel (relative to root) with size bytes
// from r. Writes via a temp file in the same directory then atomically
// renames, so a crash mid-extract leaves the previous version intact.
// All operations go through os.Root, so the kernel guarantees rel cannot
// escape the library root — including through a symlink already on disk.
func writeFile(root *os.Root, rel string, r io.Reader, size int64, overwrite bool) error {
	if !overwrite {
		if _, statErr := root.Stat(rel); statErr == nil {
			return nil
		}
	}

	dir := filepath.Dir(rel)
	if mkErr := root.MkdirAll(dir, libraryDirMode); mkErr != nil {
		return fmt.Errorf("mkdir parent of %s: %w", rel, mkErr)
	}

	tmpRel, tmp, createErr := createTemp(root, dir, filepath.Base(rel))
	if createErr != nil {
		return fmt.Errorf("create temp for %s: %w", rel, createErr)
	}
	cleanup := func() { _ = root.Remove(tmpRel) }

	if _, copyErr := io.CopyN(tmp, r, size); copyErr != nil {
		_ = tmp.Close()
		cleanup()
		return fmt.Errorf("copy bytes for %s: %w", rel, copyErr)
	}
	if chmodErr := tmp.Chmod(libraryFileMode); chmodErr != nil {
		_ = tmp.Close()
		cleanup()
		return fmt.Errorf("chmod temp for %s: %w", rel, chmodErr)
	}
	if closeErr := tmp.Close(); closeErr != nil {
		cleanup()
		return fmt.Errorf("close temp for %s: %w", rel, closeErr)
	}
	if renameErr := root.Rename(tmpRel, rel); renameErr != nil {
		cleanup()
		return fmt.Errorf("rename %s -> %s: %w", tmpRel, rel, renameErr)
	}
	return nil
}

// createTemp opens a fresh temp file under dir (relative to root) with a
// random name and O_EXCL, so it can never open — and thus never follow —
// a file or symlink already present at the path. os.Root has no
// CreateTemp of its own; this mirrors os.CreateTemp's safety inside the
// root (a predictable name + O_TRUNC would let a pre-planted in-root
// symlink alias the write to another library file).
func createTemp(root *os.Root, dir, base string) (string, *os.File, error) {
	for range 10 {
		var b [8]byte
		if _, err := rand.Read(b[:]); err != nil {
			return "", nil, err
		}
		name := filepath.Join(dir, fmt.Sprintf(".content-%s-%x.tmp", base, b))
		f, err := root.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_EXCL, libraryFileMode)
		if err == nil {
			return name, f, nil
		}
		if !errors.Is(err, os.ErrExist) {
			return "", nil, err
		}
	}
	return "", nil, errors.New("exhausted temp-name attempts")
}

// File modes for files/dirs we create. Match internal/library so the
// extracted content sits comfortably alongside hand-dropped files.
const (
	libraryDirMode  = 0o755
	libraryFileMode = 0o644
)
