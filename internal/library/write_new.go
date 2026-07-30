package library

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
)

// WriteFileNew creates content without replacing an existing library entry.
func (l *Library) WriteFileNew(kind Kind, relPath string, content []byte) error {
	if kind != KindWalks && kind != KindPcaps {
		return fmt.Errorf("%w: %s", ErrUnsupportedKind, kind)
	}
	if len(content) == 0 {
		return ErrEmptyContent
	}
	if err := validateRelPath(relPath); err != nil {
		return err
	}

	root, err := l.openKindRoot(kind)
	if err != nil {
		return err
	}
	defer func() { _ = root.Close() }()
	dir := path.Dir(relPath)
	if dir != "." {
		if err = root.MkdirAll(dir, libraryDirMode); err != nil {
			return fmt.Errorf("create library content directory: %w", err)
		}
	}
	file, err := root.OpenFile(relPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, libraryFileMode)
	if errors.Is(err, fs.ErrExist) {
		return fmt.Errorf("%w: %s", ErrAlreadyExists, relPath)
	}
	if err != nil {
		return fmt.Errorf("create library content: %w", err)
	}
	written, writeErr := file.Write(content)
	if writeErr == nil && written != len(content) {
		writeErr = io.ErrShortWrite
	}
	if closeErr := file.Close(); writeErr == nil {
		writeErr = closeErr
	}
	if writeErr != nil {
		_ = root.Remove(relPath)
		return fmt.Errorf("write library content: %w", writeErr)
	}
	return nil
}
