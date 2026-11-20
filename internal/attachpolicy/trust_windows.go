// SPDX-License-Identifier: BUSL-1.1

//go:build windows

package attachpolicy

import (
	"errors"
	"fmt"
	"os"
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
var operatorSIDs = []windows.WELL_KNOWN_SID_TYPE{
	windows.WinLocalSystemSid,
	windows.WinBuiltinAdministratorsSid,
}

// checkTrusted is the Windows form of "owned by root and not writable by
// anyone else": SYSTEM or Administrators own it, and no allow entry grants
// write access to another account. A file with no DACL grants everyone
// everything and is refused. It judges the opened handle, so nothing can be
// swapped between the check and the read.
func checkTrusted(file *os.File, _ Trust) error {
	info, err := file.Stat()
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return errors.New("not a regular file")
	}
	descriptor, err := windows.GetSecurityInfo(
		windows.Handle(file.Fd()), windows.SE_FILE_OBJECT,
		windows.OWNER_SECURITY_INFORMATION|windows.DACL_SECURITY_INFORMATION,
	)
	if err != nil {
		return fmt.Errorf("read security descriptor: %w", err)
	}
	trusted, err := operatorSIDSet()
	if err != nil {
		return err
	}
	owner, _, err := descriptor.Owner()
	if err != nil || owner == nil {
		return errors.New("cannot read file owner")
	}
	if !inSet(owner, trusted) {
		return fmt.Errorf("owned by %s, want SYSTEM or Administrators", owner)
	}
	acl, _, err := descriptor.DACL()
	if err != nil || acl == nil {
		return errors.New("has no access control list, so anyone can write it")
	}
	return checkACL(acl, trusted)
}

func checkACL(acl *windows.ACL, trusted []*windows.SID) error {
	for index := range uint32(acl.AceCount) {
		var ace *windows.ACCESS_ALLOWED_ACE
		if err := windows.GetAce(acl, index, &ace); err != nil {
			return fmt.Errorf("read access control entry %d: %w", index, err)
		}
		if ace.Header.AceType != windows.ACCESS_ALLOWED_ACE_TYPE || ace.Mask&writeMask == 0 {
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
	sids := make([]*windows.SID, 0, len(operatorSIDs))
	for _, kind := range operatorSIDs {
		sid, err := windows.CreateWellKnownSid(kind)
		if err != nil {
			return nil, fmt.Errorf("resolve well-known account: %w", err)
		}
		sids = append(sids, sid)
	}
	return sids, nil
}

func inSet(sid *windows.SID, set []*windows.SID) bool {
	for _, member := range set {
		if sid.Equals(member) {
			return true
		}
	}
	return false
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
