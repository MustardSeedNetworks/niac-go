package truststore_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/truststore"
)

func TestWriteAnchorFileRejectsEscape(t *testing.T) {
	t.Parallel()
	for _, name := range []string{"parent", "symlink"} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			root, outside := t.TempDir(), filepath.Join(t.TempDir(), "outside.crt")
			if err := os.WriteFile(outside, []byte("unchanged"), 0o600); err != nil {
				t.Fatal(err)
			}
			dst := outside
			if name == "symlink" {
				dst = filepath.Join(root, "anchor.crt")
				if err := os.Symlink(outside, dst); err != nil {
					t.Fatal(err)
				}
			}
			if _, err := truststore.WriteAnchorFile(root, dst, []byte("replacement")); err == nil {
				t.Error("expected an error for a destination outside the trust store")
			}
			data, err := os.ReadFile(outside)
			if err != nil || string(data) != "unchanged" {
				t.Fatalf("outside file changed: %q, %v", data, err)
			}
		})
	}
}

func TestWriteAnchorFileWritesPublicCertificate(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	dst := filepath.Join(root, "anchor.crt")
	written, err := truststore.WriteAnchorFile(root, dst, []byte("certificate"))
	if err != nil || written != dst {
		t.Fatalf("write anchor: %q, %v", written, err)
	}
	data, err := os.ReadFile(dst)
	if err != nil || string(data) != "certificate" {
		t.Fatalf("anchor contents: %q, %v", data, err)
	}
	info, err := os.Stat(dst)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o644 {
		t.Errorf("anchor mode = %o, want 644", info.Mode().Perm())
	}
}
