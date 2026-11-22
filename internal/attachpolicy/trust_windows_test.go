// SPDX-License-Identifier: BUSL-1.1

//go:build windows

package attachpolicy_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/sys/windows"

	"github.com/MustardSeedNetworks/niac-go/internal/attachpolicy"
)

// A temporary directory belongs to the account running the tests, not to
// SYSTEM or Administrators, so the tests trust that account.
func trustSelf(t *testing.T) (attachpolicy.Trust, string) {
	t.Helper()
	user, err := windows.GetCurrentProcessToken().GetTokenUser()
	if err != nil {
		t.Fatalf("read process token user: %v", err)
	}
	sid := user.User.Sid.String()
	return attachpolicy.TrustSIDs(sid), sid
}

// setDACL replaces a file or directory's DACL with a protected one written in
// SDDL. "SY" is SYSTEM, "BA" Administrators, "BU" Users, "FA" full control,
// "FR" read, "FW" write.
func setDACL(t *testing.T, path, sddl string) {
	t.Helper()
	descriptor, err := windows.SecurityDescriptorFromString(sddl)
	if err != nil {
		t.Fatalf("parse %q: %v", sddl, err)
	}
	dacl, _, err := descriptor.DACL()
	if err != nil {
		t.Fatalf("read DACL of %q: %v", sddl, err)
	}
	err = windows.SetNamedSecurityInfo(path, windows.SE_FILE_OBJECT,
		windows.DACL_SECURITY_INFORMATION|windows.PROTECTED_DACL_SECURITY_INFORMATION, nil, nil, dacl, nil)
	if err != nil {
		t.Fatalf("set DACL on %s: %v", path, err)
	}
}

func policyFile(t *testing.T, sddlTail string) (string, attachpolicy.Trust) {
	t.Helper()
	trust, sid := trustSelf(t)
	path := filepath.Join(t.TempDir(), attachpolicy.FileName)
	if err := os.WriteFile(path, []byte("eth0=direct\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	setDACL(t, path, "D:P(A;;FA;;;SY)(A;;FA;;;BA)(A;;FA;;;"+sid+")"+sddlTail)
	return path, trust
}

func TestLoadAcceptsOperatorFileOnWindows(t *testing.T) {
	tests := []struct {
		name string
		tail string
	}{
		{name: "operators only"},
		{name: "users may read it, as the installer leaves it", tail: "(A;;FR;;;BU)"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path, trust := policyFile(t, tt.tail)
			policies, err := attachpolicy.Load(path, trust)
			if err != nil || len(policies) != 1 {
				t.Fatalf("Load() = %v, %v; want one policy and no error", policies, err)
			}
		})
	}
}

func TestLoadRefusesUnsafeFileOnWindows(t *testing.T) {
	tests := []struct {
		name string
		tail string
	}{
		{name: "users may write it", tail: "(A;;FW;;;BU)"},
		{name: "everyone may append to it", tail: "(A;;0x4;;;WD)"},
		{name: "users may delete it", tail: "(A;;SD;;;BU)"},
		{name: "users may rewrite its DACL", tail: "(A;;WD;;;BU)"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path, trust := policyFile(t, tt.tail)
			if _, err := attachpolicy.Load(path, trust); !errors.Is(err, attachpolicy.ErrUntrusted) {
				t.Fatalf("Load() = %v, want ErrUntrusted", err)
			}
		})
	}
}

func TestLoadRefusesFileInWritableDirectoryOnWindows(t *testing.T) {
	path, trust := policyFile(t, "")
	_, sid := trustSelf(t)
	setDACL(t, filepath.Dir(path), "D:P(A;OICI;FA;;;SY)(A;OICI;FA;;;BA)(A;OICI;FA;;;"+sid+")(A;;FW;;;BU)")

	if _, err := attachpolicy.Load(path, trust); !errors.Is(err, attachpolicy.ErrUntrusted) {
		t.Fatalf("Load() = %v, want ErrUntrusted", err)
	}
}

func TestLoadRefusesLinkedFileOnWindows(t *testing.T) {
	target, trust := policyFile(t, "")
	link := filepath.Join(t.TempDir(), attachpolicy.FileName)
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("cannot create symlinks here: %v", err)
	}
	if _, err := attachpolicy.Load(link, trust); !errors.Is(err, attachpolicy.ErrUntrusted) {
		t.Fatalf("Load(symlink) = %v, want ErrUntrusted", err)
	}
}

func TestLoadMissingFileHoldsNothingOnWindows(t *testing.T) {
	trust, _ := trustSelf(t)
	for name, path := range map[string]string{
		"file":      filepath.Join(t.TempDir(), attachpolicy.FileName),
		"directory": filepath.Join(t.TempDir(), "gone", attachpolicy.FileName),
	} {
		t.Run(name, func(t *testing.T) {
			policies, err := attachpolicy.Load(path, trust)
			if err != nil || len(policies) != 0 {
				t.Fatalf("Load(missing %s) = %v, %v; want no policies and no error", name, policies, err)
			}
		})
	}
}
