// SPDX-License-Identifier: BUSL-1.1

package authstate_test

import (
	"path/filepath"
	"testing"

	"golang.org/x/sys/windows"

	"github.com/MustardSeedNetworks/niac-go/internal/authstate"
)

func TestPublishProtectedDACL(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	user, err := windows.GetCurrentProcessToken().GetTokenUser()
	if err != nil {
		t.Fatal(err)
	}
	want, err := windows.SecurityDescriptorFromString("D:P(A;;FA;;;" + user.User.Sid.String() + ")(A;;FA;;;SY)")
	if err != nil {
		t.Fatal(err)
	}
	for range 2 {
		if err = authstate.Publish(dir, []byte("state")); err != nil {
			t.Fatal(err)
		}
		got, readErr := windows.GetNamedSecurityInfo(filepath.Join(dir, "auth.json"),
			windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION)
		if readErr != nil {
			t.Fatal(readErr)
		}
		if got.String() != want.String() {
			t.Fatalf("DACL = %s, want %s", got.String(), want.String())
		}
	}
}
