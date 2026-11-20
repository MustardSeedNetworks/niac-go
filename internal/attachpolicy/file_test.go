// SPDX-License-Identifier: BUSL-1.1

//go:build unix

package attachpolicy_test

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/attachpolicy"
)

func TestJudgeTrust(t *testing.T) {
	const root, other = 0, 501
	tests := []struct {
		name  string
		mode  fs.FileMode
		uid   uint32
		match string
	}{
		{name: "root 0644", mode: 0o644, uid: root},
		{name: "root 0600", mode: 0o600, uid: root},
		{name: "root 0444", mode: 0o444, uid: root},
		{name: "owned by another user", mode: 0o644, uid: other, match: "owned by uid 501"},
		{name: "group writable", mode: 0o664, uid: root, match: "group or others write"},
		{name: "world writable", mode: 0o646, uid: root, match: "group or others write"},
		{name: "directory", mode: fs.ModeDir | 0o755, uid: root, match: "not a regular file"},
		{name: "symlink", mode: fs.ModeSymlink | 0o777, uid: root, match: "not a regular file"},
		{name: "named pipe", mode: fs.ModeNamedPipe | 0o644, uid: root, match: "not a regular file"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := attachpolicy.JudgeTrust(tt.mode, tt.uid, root)
			if tt.match == "" {
				if err != nil {
					t.Fatalf("JudgeTrust() = %v, want trusted", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.match) {
				t.Fatalf("JudgeTrust() = %v, want containing %q", err, tt.match)
			}
		})
	}
}

// trustSelf makes the files a test creates count as the operator's.
func trustSelf() attachpolicy.Trust { return attachpolicy.TrustUID(uint32(os.Getuid())) }

func writeFile(t *testing.T, content string, perm os.FileMode) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), attachpolicy.FileName)
	if err := os.WriteFile(path, []byte(content), perm); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, perm); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLoadMissingFileHoldsNothing(t *testing.T) {
	policies, err := attachpolicy.Load(filepath.Join(t.TempDir(), attachpolicy.FileName), trustSelf())
	if err != nil || len(policies) != 0 {
		t.Fatalf("Load(missing) = %v, %v; want no policies and no error", policies, err)
	}
}

func TestLoadParsesEntries(t *testing.T) {
	content := "# lab NICs\r\n\r\neth0=access:200\r\n  eth1=direct  \n# eth2=trunk:200\neth3=trunk:200,201\n"
	path := writeFile(t, content, 0o644)

	got, err := attachpolicy.Load(path, trustSelf())
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	var values []string
	for _, policy := range got {
		values = append(values, attachpolicy.Format(policy))
	}
	want := []string{"eth0=access:200", "eth1=direct", "eth3=trunk:200,201"}
	if !reflect.DeepEqual(values, want) {
		t.Fatalf("Load() = %v, want %v", values, want)
	}
}

func TestLoadRejectsBadContent(t *testing.T) {
	tests := []struct {
		name    string
		content string
		match   string
	}{
		{name: "syntax names the line", content: "eth0=direct\nbogus\n", match: "line 2"},
		{name: "bad VLAN", content: "eth0=access:5000\n", match: "between 1 and 4094"},
		{name: "contradiction in the file", content: "eth0=direct\neth0=access:200\n", match: "cannot combine direct"},
		{name: "oversized", content: strings.Repeat("# padding\n", attachpolicy.MaxFileBytes/8), match: "larger than"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := attachpolicy.Load(writeFile(t, tt.content, 0o644), trustSelf())
			if err == nil || !strings.Contains(err.Error(), tt.match) {
				t.Fatalf("Load() = %v, want containing %q", err, tt.match)
			}
		})
	}
}

func TestLoadRefusesUntrustedFile(t *testing.T) {
	t.Run("writable by others", func(t *testing.T) {
		_, err := attachpolicy.Load(writeFile(t, "eth0=direct\n", 0o666), trustSelf())
		if !errors.Is(err, attachpolicy.ErrUntrusted) || !strings.Contains(err.Error(), "chmod go-w") {
			t.Fatalf("Load() = %v, want ErrUntrusted naming the fix", err)
		}
	})
	t.Run("owned by someone else", func(t *testing.T) {
		_, err := attachpolicy.Load(writeFile(t, "eth0=direct\n", 0o644), attachpolicy.TrustUID(uint32(os.Getuid())+1))
		if !errors.Is(err, attachpolicy.ErrUntrusted) || !strings.Contains(err.Error(), "owned by uid") {
			t.Fatalf("Load() = %v, want ErrUntrusted for the owner", err)
		}
	})
	t.Run("a directory", func(t *testing.T) {
		if _, err := attachpolicy.Load(t.TempDir(), trustSelf()); err == nil {
			t.Fatal("Load(directory) = nil error")
		}
	})
}

// The production owner is root. This needs a real root-owned file, so it runs
// only when the suite does; otherwise it proves the other half, that a file
// the invoking user owns is refused.
func TestLoadTrustsOnlyRootByDefault(t *testing.T) {
	path := writeFile(t, "eth0=direct\n", 0o644)
	_, err := attachpolicy.Load(path, attachpolicy.OperatorTrust())
	if os.Geteuid() == 0 {
		if err != nil {
			t.Fatalf("Load(root-owned 0644) = %v, want trusted", err)
		}
		return
	}
	t.Log("not root: checking only that a non-root-owned file is refused")
	if !errors.Is(err, attachpolicy.ErrUntrusted) {
		t.Fatalf("Load(file owned by uid %d) = %v, want ErrUntrusted", os.Getuid(), err)
	}
}

func TestEdit(t *testing.T) {
	appendLine := func(line string) func([]string) ([]string, error) {
		return func(lines []string) ([]string, error) { return append(lines, line), nil }
	}
	tests := []struct {
		name    string
		initial string // empty means the file does not exist
		edit    func([]string) ([]string, error)
		want    string
		match   string
	}{
		{name: "creates the file", edit: appendLine("eth0=direct"), want: "eth0=direct\n"},
		{
			name: "keeps comments", initial: "# lab\neth0=direct\n\n# spare\n",
			edit: appendLine("eth1=access:200"), want: "# lab\neth0=direct\n\n# spare\neth1=access:200\n",
		},
		{
			name: "refuses a contradiction", initial: "eth0=direct\n",
			edit: appendLine("eth0=access:200"), want: "eth0=direct\n", match: "cannot combine direct",
		},
		{
			name: "edit error leaves the file", initial: "eth0=direct\n",
			edit:  func([]string) ([]string, error) { return nil, errors.New("nope") },
			want:  "eth0=direct\n",
			match: "nope",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "sub", attachpolicy.FileName)
			if tt.initial != "" {
				path = writeFile(t, tt.initial, 0o644)
			}

			err := attachpolicy.Edit(path, trustSelf(), tt.edit)
			if tt.match != "" && (err == nil || !strings.Contains(err.Error(), tt.match)) {
				t.Fatalf("Edit() = %v, want containing %q", err, tt.match)
			}
			if tt.match == "" && err != nil {
				t.Fatalf("Edit() error = %v", err)
			}

			data, readErr := os.ReadFile(path)
			if readErr != nil {
				t.Fatal(readErr)
			}
			if string(data) != tt.want {
				t.Errorf("file = %q, want %q", data, tt.want)
			}
			assertNoTemp(t, filepath.Dir(path))
		})
	}
}

func TestEditWritesReadableOwnerOnlyWritable(t *testing.T) {
	previous := setUmask(0o077)
	t.Cleanup(func() { setUmask(previous) })

	path := filepath.Join(t.TempDir(), attachpolicy.FileName)
	add := func(lines []string) ([]string, error) { return append(lines, "eth0=direct"), nil }
	if err := attachpolicy.Edit(path, trustSelf(), add); err != nil {
		t.Fatalf("Edit() error = %v", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o644 {
		t.Errorf("mode = %04o, want 0644 so the service account can read it under any umask", info.Mode().Perm())
	}
	if _, err = attachpolicy.Load(path, trustSelf()); err != nil {
		t.Errorf("Load() of an edited file = %v, want trusted", err)
	}
}

func TestEditRefusesUntrustedFile(t *testing.T) {
	path := writeFile(t, "eth0=direct\n", 0o666)
	add := func(lines []string) ([]string, error) { return append(lines, "eth1=direct"), nil }
	err := attachpolicy.Edit(path, trustSelf(), add)
	if !errors.Is(err, attachpolicy.ErrUntrusted) {
		t.Fatalf("Edit() = %v, want ErrUntrusted", err)
	}
	data, _ := os.ReadFile(path)
	if string(data) != "eth0=direct\n" {
		t.Errorf("an untrusted file was rewritten: %q", data)
	}
}

func TestEditWithoutChangeDoesNotRewrite(t *testing.T) {
	path := writeFile(t, "# keep\neth0=direct\n", 0o644)
	before, _ := os.Stat(path)
	unchanged := func(lines []string) ([]string, error) { return lines, nil }
	if err := attachpolicy.Edit(path, trustSelf(), unchanged); err != nil {
		t.Fatalf("Edit() error = %v", err)
	}
	after, _ := os.Stat(path)
	if !os.SameFile(before, after) || !after.ModTime().Equal(before.ModTime()) {
		t.Error("an unchanged edit rewrote the file")
	}
}

func assertNoTemp(t *testing.T, dir string) {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".tmp") {
			t.Errorf("temporary file %s left behind", entry.Name())
		}
	}
}

func TestDefaultPathIsFixedPerOS(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("NIAC_LIBRARY_ROOT", t.TempDir())
	got := attachpolicy.DefaultPath()
	if !filepath.IsAbs(got) || filepath.Base(got) != attachpolicy.FileName {
		t.Fatalf("DefaultPath() = %q, want an absolute path ending in %s", got, attachpolicy.FileName)
	}
	for _, env := range []string{"HOME", "XDG_DATA_HOME", "NIAC_LIBRARY_ROOT"} {
		if value := os.Getenv(env); strings.HasPrefix(got, value) {
			t.Errorf("DefaultPath() = %q follows %s", got, env)
		}
	}
}
