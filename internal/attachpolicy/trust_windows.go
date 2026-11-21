// SPDX-License-Identifier: BUSL-1.1

//go:build windows

package attachpolicy

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"unsafe"

	"golang.org/x/sys/windows"
)

// Trust says whose files count as the operator's. On Windows that is always
// SYSTEM and Administrators.
type Trust struct{}

// OperatorTrust accepts files owned by SYSTEM or Administrators.
func OperatorTrust() Trust { return Trust{} }

// writeMask is every access right that lets an account change the file or take
// it over. An allow entry granting any of them to an account outside the
// operator set makes the file untrusted.
const writeMask = windows.FILE_WRITE_DATA | windows.FILE_APPEND_DATA | windows.FILE_WRITE_EA |
	windows.FILE_WRITE_ATTRIBUTES | windows.DELETE | windows.WRITE_DAC | windows.WRITE_OWNER |
	windows.GENERIC_WRITE | windows.GENERIC_ALL

// operatorSIDs are the accounts allowed to own or write the file: SYSTEM, which
// the service runs as, and Administrators, who install it.
func operatorSIDs() []windows.WELL_KNOWN_SID_TYPE {
	return []windows.WELL_KNOWN_SID_TYPE{windows.WinLocalSystemSid, windows.WinBuiltinAdministratorsSid}
}

// fileDeleteChild lets an account delete entries of a directory regardless of
// the entries' own ACLs; windows.FILE_DELETE_CHILD is not exported.
const fileDeleteChild = 0x40

// directoryMask is writeMask plus the right to delete children: write on a
// directory is add-file and add-subdirectory, and either lets an account plant
// or replace the policy file's siblings.
const directoryMask = writeMask | fileDeleteChild

// inheritOnlyACE marks an entry that applies to a directory's children only.
const inheritOnlyACE = 0x08

// checkTrusted is the Windows form of "owned by root and not writable by
// anyone else": SYSTEM or Administrators own it, and no allow entry grants
// write access to another account. The same holds for its parent directory,
// where write, DELETE or FILE_DELETE_CHILD would let an account swap the file
// out from under a clean DACL. A file with no DACL grants everyone everything
// and is refused. It judges the opened handle, so nothing can be swapped
// between the check and the read.
func checkTrusted(file *os.File, _ Trust) error {
	info, err := file.Stat()
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return errors.New("not a regular file")
	}
	trusted, err := operatorSIDSet()
	if err != nil {
		return err
	}
	descriptor, err := windows.GetSecurityInfo(
		windows.Handle(file.Fd()), windows.SE_FILE_OBJECT,
		windows.OWNER_SECURITY_INFORMATION|windows.DACL_SECURITY_INFORMATION,
	)
	if err != nil {
		return fmt.Errorf("read security descriptor: %w", err)
	}
	if judgeErr := judgeDescriptor(descriptor, trusted, writeMask, false); judgeErr != nil {
		return judgeErr
	}
	return checkParent(file.Name(), trusted)
}

// checkParent judges the directory holding the file. It opens the directory
// itself, not a path through it, and refuses a reparse point: a junction in
// that place sends the file's neighbours somewhere else.
func checkParent(filePath string, trusted []*windows.SID) error {
	absolute, err := filepath.Abs(filePath)
	if err != nil {
		return err
	}
	dir := filepath.Dir(absolute)
	pointer, err := windows.UTF16PtrFromString(dir)
	if err != nil {
		return err
	}
	handle, err := windows.CreateFile(pointer, windows.READ_CONTROL|windows.FILE_READ_ATTRIBUTES,
		windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE|windows.FILE_SHARE_DELETE, nil,
		windows.OPEN_EXISTING, windows.FILE_FLAG_BACKUP_SEMANTICS|windows.FILE_FLAG_OPEN_REPARSE_POINT, 0)
	if err != nil {
		return fmt.Errorf("open parent directory %s: %w", dir, err)
	}
	defer func() { _ = windows.CloseHandle(handle) }()

	var attributes windows.ByHandleFileInformation
	if infoErr := windows.GetFileInformationByHandle(handle, &attributes); infoErr != nil {
		return fmt.Errorf("inspect parent directory %s: %w", dir, infoErr)
	}
	if attributes.FileAttributes&windows.FILE_ATTRIBUTE_REPARSE_POINT != 0 {
		return fmt.Errorf("parent directory %s is a reparse point", dir)
	}
	descriptor, err := windows.GetSecurityInfo(handle, windows.SE_FILE_OBJECT,
		windows.OWNER_SECURITY_INFORMATION|windows.DACL_SECURITY_INFORMATION)
	if err != nil {
		return fmt.Errorf("read security descriptor of %s: %w", dir, err)
	}
	if judgeErr := judgeDescriptor(descriptor, trusted, directoryMask, true); judgeErr != nil {
		return fmt.Errorf("parent directory %s: %w", dir, judgeErr)
	}
	return nil
}

// judgeDescriptor is the decision: an operator owns the object and its DACL
// grants none of mask to anyone else. For a directory, entries that only
// inherit to children do not touch the directory itself.
func judgeDescriptor(
	descriptor *windows.SECURITY_DESCRIPTOR,
	trusted []*windows.SID,
	mask windows.ACCESS_MASK,
	directory bool,
) error {
	owner, _, err := descriptor.Owner()
	if err != nil || owner == nil {
		return errors.New("cannot read owner")
	}
	if !inSet(owner, trusted) {
		return fmt.Errorf("owned by %s, want SYSTEM or Administrators", owner)
	}
	acl, _, err := descriptor.DACL()
	if err != nil || acl == nil {
		return errors.New("has no access control list, so anyone can write it")
	}
	return checkACL(acl, trusted, mask, directory)
}

func checkACL(acl *windows.ACL, trusted []*windows.SID, mask windows.ACCESS_MASK, directory bool) error {
	for index := range uint32(acl.AceCount) {
		var ace *windows.ACCESS_ALLOWED_ACE
		if err := windows.GetAce(acl, index, &ace); err != nil {
			return fmt.Errorf("read access control entry %d: %w", index, err)
		}
		if ace.Header.AceType != windows.ACCESS_ALLOWED_ACE_TYPE || ace.Mask&mask == 0 {
			continue
		}
		if directory && ace.Header.AceFlags&inheritOnlyACE != 0 {
			continue
		}
		sid := (*windows.SID)(unsafe.Pointer(&ace.SidStart))
		if !inSet(sid, trusted) {
			return fmt.Errorf("%s may write it; only SYSTEM and Administrators may", sid)
		}
	}
	return nil
}

func operatorSIDSet() ([]*windows.SID, error) {
	kinds := operatorSIDs()
	sids := make([]*windows.SID, 0, len(kinds))
	for _, kind := range kinds {
		sid, err := windows.CreateWellKnownSid(kind)
		if err != nil {
			return nil, fmt.Errorf("resolve well-known account: %w", err)
		}
		sids = append(sids, sid)
	}
	return sids, nil
}

func inSet(sid *windows.SID, set []*windows.SID) bool {
	return slices.ContainsFunc(set, sid.Equals)
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
