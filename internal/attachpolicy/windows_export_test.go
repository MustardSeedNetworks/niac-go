// SPDX-License-Identifier: BUSL-1.1

//go:build windows

package attachpolicy

import "golang.org/x/sys/windows"

// JudgeSDDL runs the trust decision on a descriptor written in SDDL, judged as
// a file or as a directory, so the external tests need no real ACLs.
func JudgeSDDL(sddl string, directory bool) error {
	descriptor, err := windows.SecurityDescriptorFromString(sddl)
	if err != nil {
		return err
	}
	trusted, err := operatorSIDSet()
	if err != nil {
		return err
	}
	var mask windows.ACCESS_MASK = writeMask
	if directory {
		mask = directoryMask
	}
	return judgeDescriptor(descriptor, trusted, mask, directory)
}

// CheckParent judges the directory that holds path.
func CheckParent(path string) error {
	trusted, err := operatorSIDSet()
	if err != nil {
		return err
	}
	return checkParent(path, trusted)
}
