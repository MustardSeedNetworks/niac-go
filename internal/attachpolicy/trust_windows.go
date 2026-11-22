// SPDX-License-Identifier: BUSL-1.1

//go:build windows

package attachpolicy

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"unsafe"

	"golang.org/x/sys/windows"

	"github.com/MustardSeedNetworks/foundation/pkg/securedir"
)

// Trust says whose files count as the operator's. On Windows that is SYSTEM,
// Administrators and TrustedInstaller, which securedir always trusts, plus any
// accounts a test adds.
type Trust struct{ opts securedir.Options }

// OperatorTrust accepts files owned by SYSTEM or Administrators.
func OperatorTrust() Trust { return Trust{} }

// TrustSIDs also accepts files owned by the accounts with these SIDs, so tests
// can use files they own. Nothing in the daemon or the CLI calls it.
func TrustSIDs(sids ...string) Trust {
	return Trust{opts: securedir.Options{ExtraTrustedSIDs: sids}}
}

// openTrusted opens the file through securedir, which judges the file and
// every directory above it: a trusted owner, no one else able to write, delete
// or re-ACL, and no link anywhere on the path. It reads from the handle it
// checked, so nothing can be swapped between the check and the read. Read
// access is not restricted, which is why the policy file may stay readable by
// Users. A path securedir refuses is ErrUntrusted; one that does not exist is
// fs.ErrNotExist.
func openTrusted(path string, trust Trust) (*os.File, error) {
	dir, err := securedir.Canonical(filepath.Dir(path), trust.opts)
	if err != nil {
		return nil, classify(path, err)
	}
	file, err := securedir.OpenFile(filepath.Join(dir, filepath.Base(path)), os.O_RDONLY, trust.opts)
	if err != nil {
		return nil, classify(path, err)
	}
	return file, nil
}

func classify(path string, err error) error {
	if errors.Is(err, securedir.ErrUntrustedOwner) ||
		errors.Is(err, securedir.ErrReparsePoint) ||
		errors.Is(err, securedir.ErrUnsafeAccess) {
		return untrusted(path, err)
	}
	return err
}

// createPrivateTemp attaches a protected DACL at creation: SYSTEM and
// Administrators full control, everyone else read. Attaching it afterwards
// would leave a window under whatever the directory's ACL inherits, and the
// rename that follows keeps the descriptor, so the finished file passes
// checkTrusted.
func createPrivateTemp(_ *os.Root, dir, name string) (*os.File, error) {
	descriptor, err := windows.SecurityDescriptorFromString("D:P(A;;FA;;;SY)(A;;FA;;;BA)(A;;FR;;;BU)")
	if err != nil {
		return nil, fmt.Errorf("build security descriptor: %w", err)
	}
	attributes := windows.SecurityAttributes{
		Length:             uint32(unsafe.Sizeof(windows.SecurityAttributes{})),
		SecurityDescriptor: descriptor,
	}
	path, err := windows.UTF16PtrFromString(dir + `\` + name)
	if err != nil {
		return nil, err
	}
	handle, err := windows.CreateFile(path, windows.GENERIC_WRITE, 0, &attributes,
		windows.CREATE_NEW, windows.FILE_ATTRIBUTE_NORMAL, 0)
	if err != nil {
		return nil, err
	}
	return os.NewFile(uintptr(handle), name), nil
}
