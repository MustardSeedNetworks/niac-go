//go:build windows

package library

import (
	"os"

	"golang.org/x/sys/windows"
)

func lockDraftFile(file *os.File, exclusive bool) (func(), error) {
	overlapped := &windows.Overlapped{}
	handle := windows.Handle(file.Fd())
	var flags uint32
	if exclusive {
		flags = windows.LOCKFILE_EXCLUSIVE_LOCK
	}
	if err := windows.LockFileEx(handle, flags, 0, 1, 0, overlapped); err != nil {
		return nil, err
	}
	return func() { _ = windows.UnlockFileEx(handle, 0, 1, 0, overlapped) }, nil
}
