//go:build unix

package main

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/MustardSeedNetworks/niac-go/internal/attachpolicy"
)

func testTrust() attachpolicy.Trust { return attachpolicy.TrustUID(uint32(os.Getuid())) }

func policyFile(t *testing.T, content string, perm os.FileMode) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), attachpolicy.FileName)
	if content != "" {
		if err := os.WriteFile(path, []byte(content), perm); err != nil {
			t.Fatal(err)
		}
		if err := os.Chmod(path, perm); err != nil {
			t.Fatal(err)
		}
	}
	return path
}

func formatted(policies []attachpolicy.Policy) []string {
	values := make([]string, len(policies))
	for i, policy := range policies {
		values[i] = attachpolicy.Format(policy)
	}
	return values
}

func TestMergeAttachmentPolicies(t *testing.T) {
	tests := []struct {
		name    string
		file    string
		perm    os.FileMode
		flags   []string
		want    []string
		wantErr string
	}{
		{name: "no file, no flags", want: []string{}},
		{name: "file only", file: "eth0=direct\n", perm: 0o644, want: []string{"eth0=direct"}},
		{name: "flags only", flags: []string{"eth0=access:200"}, want: []string{"eth0=access:200"}},
		{
			name: "file and flags merge, flags first", file: "eth1=direct\n", perm: 0o644,
			flags: []string{"eth0=access:200"}, want: []string{"eth0=access:200", "eth1=direct"},
		},
		{
			name: "the same policy in both is one", file: "eth0=trunk:200,201\n", perm: 0o644,
			flags: []string{"eth0=trunk:200,201"}, want: []string{"eth0=trunk:200,201"},
		},
		{
			name: "flag contradicts file", file: "eth0=direct\n", perm: 0o644,
			flags: []string{"eth0=access:200"}, wantErr: "contradict each other",
		},
		{
			name: "untrusted file is ignored, flags stand", file: "eth1=direct\n", perm: 0o666,
			flags: []string{"eth0=access:200"}, want: []string{"eth0=access:200"},
		},
		{
			name: "unparsable file is ignored, flags stand", file: "garbage\n", perm: 0o644,
			flags: []string{"eth0=access:200"}, want: []string{"eth0=access:200"},
		},
		{name: "bad flag stops the start", flags: []string{"eth0"}, wantErr: "invalid attachment policy"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := mergeAttachmentPolicies(policyFile(t, tt.file, tt.perm), testTrust(), tt.flags)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("error = %v, want containing %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("error = %v", err)
			}
			if !reflect.DeepEqual(formatted(got), tt.want) {
				t.Fatalf("policies = %v, want %v", formatted(got), tt.want)
			}
		})
	}
}

// runPolicyCmd runs `niac attachment-policy <args>` against path as root or as
// an ordinary user, returning what it printed and how it ended.
func runPolicyCmd(t *testing.T, path string, root bool, args ...string) (string, error) {
	t.Helper()
	parent := &cobra.Command{Use: "niac", SilenceUsage: true}
	editor := policyEditor{path: path, trust: testTrust(), privileged: func() bool { return root }}
	parent.AddCommand(newAttachmentPolicyCmd(editor))
	var out bytes.Buffer
	parent.SetOut(&out)
	parent.SetErr(&out)
	parent.SetArgs(append([]string{"attachment-policy"}, args...))
	err := parent.Execute()
	return out.String(), err
}

func readPolicyFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestAttachmentPolicyAdd(t *testing.T) {
	path := policyFile(t, "# lab NICs\neth0=access:200\n", 0o644)

	out, err := runPolicyCmd(t, path, true, "add", "eth1=trunk:200,201")
	if err != nil {
		t.Fatalf("add error = %v", err)
	}
	if got, want := readPolicyFile(t, path), "# lab NICs\neth0=access:200\neth1=trunk:200,201\n"; got != want {
		t.Errorf("file = %q, want %q", got, want)
	}
	if !strings.Contains(out, "Added eth1=trunk:200,201") || !strings.Contains(out, "Restart it to apply") {
		t.Errorf("output lacks the result and the restart step:\n%s", out)
	}

	out, err = runPolicyCmd(t, path, true, "add", "eth1=trunk:200,201")
	if err != nil {
		t.Fatalf("repeat add error = %v", err)
	}
	if !strings.Contains(out, "already in") || strings.Contains(out, "Restart") {
		t.Errorf("a repeated add should report and ask for no restart:\n%s", out)
	}
	if strings.Count(readPolicyFile(t, path), "eth1=trunk:200,201") != 1 {
		t.Errorf("a repeated add duplicated the line:\n%s", readPolicyFile(t, path))
	}
}

func TestAttachmentPolicyAddCreatesFile(t *testing.T) {
	path := policyFile(t, "", 0)
	if _, err := runPolicyCmd(t, path, true, "add", "en16=direct"); err != nil {
		t.Fatalf("add error = %v", err)
	}
	if got := readPolicyFile(t, path); got != "en16=direct\n" {
		t.Errorf("file = %q", got)
	}
}

func TestAttachmentPolicyAddRefusals(t *testing.T) {
	tests := []struct {
		name    string
		initial string
		args    []string
		root    bool
		match   string
	}{
		{
			name: "not root names the exact sudo command", initial: "eth0=direct\n",
			args: []string{"add", "eth1=direct"}, root: false,
			match: "sudo niac attachment-policy add eth1=direct",
		},
		{name: "bad value", args: []string{"add", "eth1"}, root: true, match: "invalid attachment policy"},
		{
			name: "contradicts the file", initial: "eth0=direct\n",
			args: []string{"add", "eth0=access:200"}, root: true, match: "cannot combine direct",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := policyFile(t, tt.initial, 0o644)
			_, err := runPolicyCmd(t, path, tt.root, tt.args...)
			if err == nil || !strings.Contains(err.Error(), tt.match) {
				t.Fatalf("error = %v, want containing %q", err, tt.match)
			}
			if tt.initial != "" && readPolicyFile(t, path) != tt.initial {
				t.Errorf("a refused add changed the file: %q", readPolicyFile(t, path))
			}
		})
	}
}

func TestAttachmentPolicyAddRefusesUntrustedFile(t *testing.T) {
	path := policyFile(t, "eth0=direct\n", 0o666)
	_, err := runPolicyCmd(t, path, true, "add", "eth1=direct")
	if err == nil || !strings.Contains(err.Error(), "not trusted") {
		t.Fatalf("error = %v, want the trust refusal", err)
	}
}

func TestAttachmentPolicyRemove(t *testing.T) {
	path := policyFile(t, "# keep me\neth0=direct\neth1=access:200\n", 0o644)

	out, err := runPolicyCmd(t, path, true, "remove", "eth0=direct")
	if err != nil {
		t.Fatalf("remove error = %v", err)
	}
	if got, want := readPolicyFile(t, path), "# keep me\neth1=access:200\n"; got != want {
		t.Errorf("file = %q, want %q", got, want)
	}
	if !strings.Contains(out, "Removed eth0=direct") || !strings.Contains(out, "Restart it to apply") {
		t.Errorf("output lacks the result and the restart step:\n%s", out)
	}

	if _, err = runPolicyCmd(t, path, true, "remove", "eth0=direct"); err == nil ||
		!strings.Contains(err.Error(), "is not in") {
		t.Fatalf("removing an absent policy: error = %v, want 'is not in'", err)
	}
	if _, err = runPolicyCmd(t, path, false, "remove", "eth1=access:200"); err == nil ||
		!strings.Contains(err.Error(), "sudo niac attachment-policy remove eth1=access:200") {
		t.Fatalf("remove as non-root: error = %v, want the sudo command", err)
	}
}

func TestAttachmentPolicyList(t *testing.T) {
	path := policyFile(t, "# lab\neth0=direct\n\neth1=trunk:200,201\n", 0o644)
	out, err := runPolicyCmd(t, path, false, "list")
	if err != nil {
		t.Fatalf("list error = %v", err)
	}
	if out != "eth0=direct\neth1=trunk:200,201\n" {
		t.Errorf("list output = %q", out)
	}

	out, err = runPolicyCmd(t, policyFile(t, "", 0), false, "list")
	if err != nil || !strings.Contains(out, "No attachment policies in") {
		t.Errorf("list of a missing file = %q, %v", out, err)
	}

	if _, err = runPolicyCmd(t, policyFile(t, "eth0=direct\n", 0o666), false, "list"); err == nil {
		t.Error("list read a file anyone could write")
	}
}
