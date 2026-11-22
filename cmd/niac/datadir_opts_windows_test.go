//go:build windows

package main

import (
	"testing"

	"golang.org/x/sys/windows"

	"github.com/MustardSeedNetworks/foundation/pkg/securedir"
)

// testDirOptions trusts the account running the tests: a temporary directory
// is owned by that account, not by SYSTEM or Administrators.
func testDirOptions(t testing.TB) securedir.Options {
	t.Helper()
	user, err := windows.GetCurrentProcessToken().GetTokenUser()
	if err != nil {
		t.Fatalf("read process token user: %v", err)
	}
	return securedir.Options{ExtraTrustedSIDs: []string{user.User.Sid.String()}}
}
