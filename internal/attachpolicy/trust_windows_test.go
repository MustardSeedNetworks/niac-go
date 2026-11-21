// SPDX-License-Identifier: BUSL-1.1

//go:build windows

package attachpolicy_test

import (
	"path/filepath"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/attachpolicy"
)

// Each row is an SDDL descriptor. "D:P" is a protected DACL, SY SYSTEM, BA
// Administrators, BU Users, WD Everyone, CO Creator Owner. FA is full control,
// FR read, FW write, SD is DELETE, "WD" in the rights field is WRITE_DAC and
// 0x40 is FILE_DELETE_CHILD.
func TestJudgeDescriptor(t *testing.T) {
	const operators = "(A;;FA;;;SY)(A;;FA;;;BA)"
	tests := []struct {
		name      string
		sddl      string
		directory bool
		wantErr   bool
	}{
		{name: "file owned by SYSTEM", sddl: "O:SYD:P" + operators},
		{name: "file owned by Administrators", sddl: "O:BAD:P" + operators},
		{name: "users may read a file", sddl: "O:BAD:P" + operators + "(A;;FR;;;BU)"},
		{name: "file owned by a user", sddl: "O:BUD:P" + operators, wantErr: true},
		{name: "file owned by another account", sddl: "O:S-1-5-21-1-2-3-1001D:P" + operators, wantErr: true},
		{name: "users may write a file", sddl: "O:BAD:P" + operators + "(A;;FW;;;BU)", wantErr: true},
		{name: "everyone may append to a file", sddl: "O:BAD:P" + operators + "(A;;0x4;;;WD)", wantErr: true},
		{name: "users may delete a file", sddl: "O:BAD:P" + operators + "(A;;SD;;;BU)", wantErr: true},
		{name: "users may rewrite the DACL", sddl: "O:BAD:P" + operators + "(A;;WD;;;BU)", wantErr: true},
		{name: "a deny entry is not a grant", sddl: "O:BAD:P(D;;FW;;;BU)" + operators},
		{name: "no DACL", sddl: "O:BA", wantErr: true},

		{
			name:      "directory of SYSTEM and Administrators",
			sddl:      "O:BAD:P" + operators + "(A;OICIIO;FA;;;SY)",
			directory: true,
		},
		{name: "users may read a directory", sddl: "O:BAD:P" + operators + "(A;;FR;;;BU)", directory: true},
		{name: "users may add files", sddl: "O:BAD:P" + operators + "(A;;0x2;;;BU)", directory: true, wantErr: true},
		{
			name:      "users may add subdirectories",
			sddl:      "O:BAD:P" + operators + "(A;;0x4;;;BU)",
			directory: true,
			wantErr:   true,
		},
		{
			name:      "everyone may delete children",
			sddl:      "O:BAD:P" + operators + "(A;;0x40;;;WD)",
			directory: true,
			wantErr:   true,
		},
		{
			name:      "users may delete the directory",
			sddl:      "O:BAD:P" + operators + "(A;;SD;;;BU)",
			directory: true,
			wantErr:   true,
		},
		{name: "directory owned by a user", sddl: "O:BUD:P" + operators, directory: true, wantErr: true},
		{
			name:      "inherit-only write does not touch the directory itself",
			sddl:      "O:BAD:P" + operators + "(A;OICIIO;FW;;;BU)",
			directory: true,
		},
		{
			name:      "inherited write still applies to the directory",
			sddl:      "O:BAD:P" + operators + "(A;OICI;FW;;;BU)",
			directory: true,
			wantErr:   true,
		},
		{
			name:    "inherit-only write on a file is still refused",
			sddl:    "O:BAD:P" + operators + "(A;OICIIO;FW;;;BU)",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := attachpolicy.JudgeSDDL(tt.sddl, tt.directory)
			if (err != nil) != tt.wantErr {
				t.Fatalf("JudgeSDDL(%q) = %v, wantErr %v", tt.sddl, err, tt.wantErr)
			}
		})
	}
}

// A parent directory that does not exist is refused.
func TestCheckParentRefusesMissingDirectory(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "gone", attachpolicy.FileName)
	if err := attachpolicy.CheckParent(missing); err == nil {
		t.Fatal("CheckParent accepted a file in a directory that does not exist")
	}
}
