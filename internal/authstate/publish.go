// SPDX-License-Identifier: BUSL-1.1

package authstate

import (
	"crypto/rand"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// ErrUncertain means publication may have occurred without confirmed durability.
// The manager must stop serving authentication until state is reloaded; it must
// neither publish the proposed in-memory state nor continue using stale state.
var ErrUncertain = errors.New("auth state durability is uncertain")

// Publish durably replaces auth.json in an existing absolute operator-controlled
// data directory. Callers serialize mutations and publish memory only on success.
// It does not validate the encoded record; the state adapter owns that boundary.
func Publish(directory string, record []byte) error {
	if !filepath.IsAbs(directory) {
		return errors.New("auth state directory must be absolute")
	}
	return publish(directory, record, writeRecord)
}

// publish keeps the write boundary injectable for filesystem failure tests.
func publish(directory string, record []byte, write func(*os.File, []byte) error) error {
	temporary := filepath.Join(directory, ".auth-"+rand.Text())
	file, err := createPrivate(temporary)
	if err != nil {
		return fmt.Errorf("create auth state: %w", err)
	}
	if err = write(file, record); err != nil {
		return fmt.Errorf("write auth state: %w", errors.Join(err, os.Remove(temporary)))
	}
	if err = replaceRecord(temporary, filepath.Join(directory, "auth.json")); err != nil {
		removeErr := os.Remove(temporary)
		if errors.Is(removeErr, os.ErrNotExist) {
			removeErr = nil
		}
		return fmt.Errorf("publish auth state: %w", errors.Join(err, removeErr))
	}
	return nil
}

// writeRecord closes even on failure and never publishes an unsynced file.
func writeRecord(file *os.File, record []byte) error {
	_, err := file.Write(record)
	if err == nil {
		err = file.Sync()
	}
	return errors.Join(err, file.Close())
}
