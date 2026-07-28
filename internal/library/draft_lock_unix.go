//go:build darwin || linux

package library

import (
	"os"

	"golang.org/x/sys/unix"
)

func lockDraftFile(file *os.File, exclusive bool) (func(), error) {
	mode := unix.LOCK_SH
	if exclusive {
		mode = unix.LOCK_EX
	}
	if err := unix.Flock(int(file.Fd()), mode); err != nil {
		return nil, err
	}
	return func() { _ = unix.Flock(int(file.Fd()), unix.LOCK_UN) }, nil
}
