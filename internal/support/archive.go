// Package support implements the operator-facing recovery and diagnostics
// surfaces: a backup of the content library, its restore, and a support
// bundle carrying diagnostics with every credential removed.
//
// Backup and Restore move authored content -- networks, walks, captures and
// drafts -- and nothing else. Certificates, the run-history database and the
// daemon's token are deliberately outside the archive: they are host identity,
// not authored truth, and a backup that carried them would move a private key
// between machines.
package support

import (
	"archive/tar"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// Sentinel errors callers classify on.
var (
	// ErrUnsafeEntry reports an archive entry that names a path outside the
	// restore root or a type the archive is not allowed to carry.
	ErrUnsafeEntry = errors.New("unsafe archive entry")
)

const (
	// archiveFileMode and archiveDirMode are the modes every entry is written
	// with. Recording the source mode would make a backup depend on the umask
	// of whoever took it, and the library only ever holds data files.
	archiveFileMode = 0o644
	archiveDirMode  = 0o755
)

// Backup writes every regular file under root into w as a gzipped tar.
//
// The archive is deterministic: entries are walked in lexical order and carry
// no modification time, owner or source mode, so two backups of an unchanged
// tree are byte-identical and an operator can tell a content change from
// archive noise.
func Backup(root string, w io.Writer) error {
	gz, err := gzip.NewWriterLevel(w, gzip.BestCompression)
	if err != nil {
		return fmt.Errorf("open gzip writer: %w", err)
	}
	tw := tar.NewWriter(gz)

	if walkErr := filepath.WalkDir(root, func(current string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		name, relErr := archiveName(root, current)
		if relErr != nil || name == "" {
			return relErr
		}
		// Symlinks and devices are skipped rather than refused: the library
		// holds data files, and a stray link is not worth failing an
		// operator's backup over.
		if !entry.IsDir() && !entry.Type().IsRegular() {
			return nil
		}
		return writeEntry(tw, current, name, entry.IsDir())
	}); walkErr != nil {
		return fmt.Errorf("archive %s: %w", root, walkErr)
	}

	if closeErr := tw.Close(); closeErr != nil {
		return fmt.Errorf("close tar: %w", closeErr)
	}
	if closeErr := gz.Close(); closeErr != nil {
		return fmt.Errorf("close gzip: %w", closeErr)
	}
	return nil
}

// archiveName renders current as a slash-separated path relative to root.
// The root itself returns "", which the caller skips.
func archiveName(root, current string) (string, error) {
	relative, err := filepath.Rel(root, current)
	if err != nil {
		return "", fmt.Errorf("relative path for %s: %w", current, err)
	}
	if relative == "." {
		return "", nil
	}
	return filepath.ToSlash(relative), nil
}

func writeEntry(tw *tar.Writer, source, name string, isDir bool) error {
	header := &tar.Header{
		Name:     name,
		Mode:     archiveFileMode,
		Typeflag: tar.TypeReg,
		Format:   tar.FormatPAX,
	}
	if isDir {
		header.Name = name + "/"
		header.Mode = archiveDirMode
		header.Typeflag = tar.TypeDir
	}

	if !isDir {
		info, err := os.Stat(source)
		if err != nil {
			return fmt.Errorf("stat %s: %w", source, err)
		}
		header.Size = info.Size()
	}
	if err := tw.WriteHeader(header); err != nil {
		return fmt.Errorf("write header %s: %w", name, err)
	}
	if isDir {
		return nil
	}

	file, openErr := os.Open(source)
	if openErr != nil {
		return fmt.Errorf("open %s: %w", source, openErr)
	}
	defer func() { _ = file.Close() }()

	if _, copyErr := io.CopyN(tw, file, header.Size); copyErr != nil {
		return fmt.Errorf("copy %s: %w", source, copyErr)
	}
	return nil
}

// Restore replaces the tree at root with the archive read from r.
//
// The archive is expanded into a sibling directory first and swapped into
// place only once every entry has landed, so a refused or truncated archive
// leaves the operator's existing library exactly as it was.
func Restore(r io.Reader, root string) error {
	absolute, err := filepath.Abs(root)
	if err != nil {
		return fmt.Errorf("resolve %s: %w", root, err)
	}

	staging, err := os.MkdirTemp(filepath.Dir(absolute), filepath.Base(absolute)+".restore-")
	if err != nil {
		return fmt.Errorf("stage restore next to %s: %w", absolute, err)
	}
	defer func() { _ = os.RemoveAll(staging) }()

	if expandErr := expand(r, staging); expandErr != nil {
		return expandErr
	}
	return swap(staging, absolute)
}

// expand writes the archive into an already-created staging directory. Every
// path is opened through os.Root, so an entry naming its way outside the
// directory fails instead of writing there.
func expand(r io.Reader, staging string) error {
	gz, err := gzip.NewReader(r)
	if err != nil {
		return fmt.Errorf("open gzip reader: %w", err)
	}
	defer func() { _ = gz.Close() }()

	stagingRoot, err := os.OpenRoot(staging)
	if err != nil {
		return fmt.Errorf("open staging root: %w", err)
	}
	defer func() { _ = stagingRoot.Close() }()

	tr := tar.NewReader(gz)
	for {
		header, nextErr := tr.Next()
		if errors.Is(nextErr, io.EOF) {
			return nil
		}
		if nextErr != nil {
			return fmt.Errorf("read archive: %w", nextErr)
		}
		if extractErr := extractEntry(stagingRoot, tr, header); extractErr != nil {
			return extractErr
		}
	}
}

func extractEntry(stagingRoot *os.Root, tr io.Reader, header *tar.Header) error {
	name, err := safeEntryName(header)
	if err != nil {
		return err
	}

	if header.Typeflag == tar.TypeDir {
		if mkdirErr := stagingRoot.MkdirAll(name, archiveDirMode); mkdirErr != nil {
			return fmt.Errorf("create %s: %w", name, mkdirErr)
		}
		return nil
	}

	if parent := path.Dir(name); parent != "." {
		if mkdirErr := stagingRoot.MkdirAll(parent, archiveDirMode); mkdirErr != nil {
			return fmt.Errorf("create %s: %w", parent, mkdirErr)
		}
	}
	file, createErr := stagingRoot.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_EXCL, archiveFileMode)
	if createErr != nil {
		return fmt.Errorf("create %s: %w", name, createErr)
	}
	defer func() { _ = file.Close() }()

	// CopyN bounds the write by the declared size, so a crafted archive
	// cannot expand past what its own header promised.
	if _, copyErr := io.CopyN(file, tr, header.Size); copyErr != nil {
		return fmt.Errorf("write %s: %w", name, copyErr)
	}
	return nil
}

// safeEntryName refuses everything a backup this package wrote would never
// contain: a link or device entry, an absolute path, or one climbing out of
// the root. os.Root enforces the same boundary; this reports it as the
// archive's fault rather than as a filesystem error.
func safeEntryName(header *tar.Header) (string, error) {
	switch header.Typeflag {
	case tar.TypeReg, tar.TypeDir:
	default:
		return "", fmt.Errorf("%w: %s is type %q", ErrUnsafeEntry, header.Name, header.Typeflag)
	}

	name := path.Clean(strings.TrimSuffix(header.Name, "/"))
	if name == "" || name == "." {
		return "", fmt.Errorf("%w: empty name", ErrUnsafeEntry)
	}
	if path.IsAbs(name) || filepath.IsAbs(header.Name) {
		return "", fmt.Errorf("%w: %s is absolute", ErrUnsafeEntry, header.Name)
	}
	if name == ".." || strings.HasPrefix(name, "../") {
		return "", fmt.Errorf("%w: %s escapes the root", ErrUnsafeEntry, header.Name)
	}
	return name, nil
}

// swap moves staging into place at target, keeping the previous tree until the
// new one is installed so a failed rename cannot leave the operator with
// neither.
func swap(staging, target string) error {
	previous := target + ".replaced"
	_ = os.RemoveAll(previous)

	_, statErr := os.Stat(target)
	switch {
	case statErr == nil:
		if renameErr := os.Rename(target, previous); renameErr != nil {
			return fmt.Errorf("move aside %s: %w", target, renameErr)
		}
	case !errors.Is(statErr, os.ErrNotExist):
		return fmt.Errorf("inspect %s: %w", target, statErr)
	default:
		if mkdirErr := os.MkdirAll(filepath.Dir(target), archiveDirMode); mkdirErr != nil {
			return fmt.Errorf("create parent of %s: %w", target, mkdirErr)
		}
	}

	if renameErr := os.Rename(staging, target); renameErr != nil {
		if statErr == nil {
			_ = os.Rename(previous, target)
		}
		return fmt.Errorf("install %s: %w", target, renameErr)
	}
	return os.RemoveAll(previous)
}
