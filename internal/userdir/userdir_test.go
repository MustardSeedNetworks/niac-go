package userdir_test

import (
	"os/user"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/userdir"
)

func TestConfigHomeUsesInvokingUserForPrivilegedRun(t *testing.T) {
	invoking, err := user.Current()
	if err != nil {
		t.Fatal(err)
	}
	root := &user.User{Uid: "0", Username: "root", HomeDir: "/root"}

	if got := userdir.ResolveHome(root, invoking.Username); got != invoking.HomeDir {
		t.Fatalf("ResolveHome() = %q, want invoking user home %q", got, invoking.HomeDir)
	}
}

func TestConfigHomeIgnoresSudoUserWithoutPrivilege(t *testing.T) {
	current := &user.User{Uid: "501", Username: "operator", HomeDir: "/Users/operator"}
	if got := userdir.ResolveHome(current, "someone-else"); got != current.HomeDir {
		t.Fatalf("ResolveHome() = %q, want current user home %q", got, current.HomeDir)
	}
}
