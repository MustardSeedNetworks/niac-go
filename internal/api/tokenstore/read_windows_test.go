//go:build windows

package tokenstore

import (
	"errors"
	"path/filepath"
	"testing"

	"golang.org/x/sys/windows"

	"github.com/MustardSeedNetworks/foundation/pkg/securedir"
)

const tokenBody = `{"tokens":[{"value":"rw-secret","scope":"read-write"}]}`

func currentUserSID(t testing.TB) string {
	t.Helper()
	user, err := windows.GetCurrentProcessToken().GetTokenUser()
	if err != nil {
		t.Fatalf("read process token user: %v", err)
	}
	return user.User.Sid.String()
}

// trustedOptions trusts the account running the tests: a temporary directory
// belongs to that account, not to SYSTEM or Administrators.
func trustedOptions(t testing.TB) securedir.Options {
	return securedir.Options{ExtraTrustedSIDs: []string{currentUserSID(t)}}
}

func loadTokens(t *testing.T, path string) ([]ScopedToken, error) {
	t.Helper()
	opts := trustedOptions(t)
	return loadTokenFile(path, func(path string) ([]byte, error) { return readPrivateFile(path, opts) })
}

// setDACL replaces the file's DACL with a protected one written in SDDL.
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

func TestLoadTokenFileAcceptsPrivateFileOnWindows(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tokens.json")
	if err := securedir.WriteFile(path, []byte(tokenBody), trustedOptions(t)); err != nil {
		t.Fatal(err)
	}

	tokens, err := loadTokens(t, path)
	if err != nil {
		t.Fatalf("loadTokens refused a private file: %v", err)
	}
	if len(tokens) != 1 || tokens[0].Value != "rw-secret" || tokens[0].Scope != ScopeReadWrite {
		t.Fatalf("tokens = %+v", tokens)
	}
}

func TestLoadTokenFileRefusesUserWritableFileOnWindows(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tokens.json")
	if err := securedir.WriteFile(path, []byte(tokenBody), trustedOptions(t)); err != nil {
		t.Fatal(err)
	}
	setDACL(t, path, "D:P(A;;FA;;;SY)(A;;FA;;;BA)(A;;FA;;;"+currentUserSID(t)+")(A;;FW;;;BU)")

	_, err := loadTokens(t, path)
	if !errors.Is(err, securedir.ErrUnsafeAccess) {
		t.Fatalf("loadTokens of a Users-writable file = %v, want ErrUnsafeAccess", err)
	}
}
