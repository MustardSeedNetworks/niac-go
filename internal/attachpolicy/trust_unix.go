// SPDX-License-Identifier: BUSL-1.1

//go:build unix

package attachpolicy

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"syscall"
)

// Trust says whose files count as the operator's.
type Trust struct{ uid uint32 }

// OperatorTrust accepts files owned by root, the only account the daemon and
// the CLI use.
func OperatorTrust() Trust { return Trust{} }

// TrustUID accepts files owned by uid in place of root, so tests can use files
// they own. Nothing in the daemon or the CLI calls it.
func TrustUID(uid uint32) Trust { return Trust{uid: uid} }

// checkTrusted judges the opened file, not its path, so nothing can be swapped
// between the check and the read.
func checkTrusted(file *os.File, trust Trust) error {
	info, err := file.Stat()
	if err != nil {
		return err
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return errors.New("cannot read file ownership")
	}
	return judgeTrust(info.Mode(), stat.Uid, trust.uid)
}

// judgeTrust is the whole decision: a regular file, owned by the operator
// account, that no one else can write. Read access is not restricted, so the
// unprivileged service account can load what root wrote.
func judgeTrust(mode fs.FileMode, uid, owner uint32) error {
	if !mode.IsRegular() {
		return errors.New("not a regular file")
	}
	if uid != owner {
		return fmt.Errorf("owned by uid %d, want uid %d (root)", uid, owner)
	}
	if mode.Perm()&0o022 != 0 {
		return fmt.Errorf("mode %04o lets group or others write it; remove that with chmod go-w", mode.Perm())
	}
	return nil
}

const privateFilePerm = 0o600

// createPrivateTemp makes the temporary file and sets its mode explicitly, so a
// restrictive umask cannot leave the unprivileged service account unable to
// read what root wrote.
func createPrivateTemp(root *os.Root, _, name string) (*os.File, error) {
	file, err := root.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_EXCL, privateFilePerm)
	if err != nil {
		return nil, err
	}
	if err = file.Chmod(filePerm); err != nil {
		_ = file.Close()
		return nil, err
	}
	return file, nil
}
