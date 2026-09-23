// SPDX-License-Identifier: BUSL-1.1

//go:build !windows

package authstate

import (
	"errors"
	"os"
	"path/filepath"
)

// createPrivate never reuses a file whose permissions might be too broad.
func createPrivate(path string) (*os.File, error) {
	return os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
}

// replaceRecord syncs the directory entry as well as the already-synced contents.
func replaceRecord(temporary, target string) error {
	directory, err := os.Open(filepath.Dir(target))
	if err != nil {
		return err
	}
	if err = os.Rename(temporary, target); err != nil {
		return errors.Join(err, directory.Close())
	}
	return syncDirectory(directory)
}

// syncDirectory fences callers when a visible replacement cannot be confirmed.
func syncDirectory(directory *os.File) error {
	if err := errors.Join(directory.Sync(), directory.Close()); err != nil {
		return errors.Join(ErrUncertain, err)
	}
	return nil
}
