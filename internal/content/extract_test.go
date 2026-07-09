package content_test

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/content"
)

// maxEntrySize mirrors the package-internal cap. Duplicated rather
// than exported because the cap is an implementation choice the
// callers shouldn't see, but the oversized-entry test needs to build
// a header just over it.
const maxEntrySize int64 = 256 << 20

// tarballEntry is the minimal description of a file we want present in
// a synthesised test bundle. Body is empty for directories.
type tarballEntry struct {
	name string
	body string
	dir  bool
}

func buildTarball(t *testing.T, entries []tarballEntry) []byte {
	t.Helper()
	var raw bytes.Buffer
	gz := gzip.NewWriter(&raw)
	tw := tar.NewWriter(gz)
	for _, e := range entries {
		hdr := &tar.Header{
			Name: e.name,
			Mode: 0o644,
			Size: int64(len(e.body)),
		}
		if e.dir {
			hdr.Typeflag = tar.TypeDir
			hdr.Size = 0
		} else {
			hdr.Typeflag = tar.TypeReg
		}
		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatalf("write tar header: %v", err)
		}
		if !e.dir {
			if _, err := tw.Write([]byte(e.body)); err != nil {
				t.Fatalf("write tar body: %v", err)
			}
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatalf("close tar: %v", err)
	}
	if err := gz.Close(); err != nil {
		t.Fatalf("close gz: %v", err)
	}
	return raw.Bytes()
}

func TestExtractRejectsPathTraversal(t *testing.T) {
	cases := []string{
		"../escape.yaml",
		"networks/../../../etc/passwd",
		"networks/../escape.yaml",
		"./../escape.yaml",
	}
	for _, name := range cases {
		t.Run(name, func(t *testing.T) {
			tarball := buildTarball(t, []tarballEntry{{name: name, body: "x"}})
			tmp := t.TempDir()
			_, err := content.Extract(bytes.NewReader(tarball), tmp, content.ExtractOptions{})
			if !errors.Is(err, content.ErrPathEscape) && !errors.Is(err, content.ErrUnknownKind) {
				t.Fatalf("expected ErrPathEscape or ErrUnknownKind for %q, got %v", name, err)
			}
		})
	}
}

func TestExtractRejectsUnknownTopLevel(t *testing.T) {
	tarball := buildTarball(t, []tarballEntry{
		{name: "secrets/passwd", body: "root:x:0:0::/:/bin/sh"},
	})
	_, err := content.Extract(bytes.NewReader(tarball), t.TempDir(), content.ExtractOptions{})
	if !errors.Is(err, content.ErrUnknownKind) {
		t.Fatalf("expected ErrUnknownKind, got %v", err)
	}
}

func TestExtractAcceptsValidBundle(t *testing.T) {
	tarball := buildTarball(t, []tarballEntry{
		{name: "networks/", dir: true},
		{name: "networks/example.yaml", body: "devices: []\n"},
		{name: "walks/router.walk", body: "1.3.6.1 = STRING: \"x\"\n"},
		{name: "pcaps/sample.pcap", body: "fake-pcap-bytes"},
	})
	root := t.TempDir()
	manifest, err := content.Extract(bytes.NewReader(tarball), root, content.ExtractOptions{})
	if err != nil {
		t.Fatalf("extract: %v", err)
	}
	if manifest.Files != 3 {
		t.Errorf("files: got %d want 3", manifest.Files)
	}
	if manifest.PerKind["networks"] != 1 {
		t.Errorf("networks count: got %d want 1", manifest.PerKind["networks"])
	}
	for _, rel := range []string{"networks/example.yaml", "walks/router.walk", "pcaps/sample.pcap"} {
		if _, statErr := os.Stat(filepath.Join(root, rel)); statErr != nil {
			t.Errorf("expected %s on disk: %v", rel, statErr)
		}
	}
}

func TestExtractDryRunWritesNothing(t *testing.T) {
	tarball := buildTarball(t, []tarballEntry{
		{name: "networks/x.yaml", body: "x"},
	})
	root := t.TempDir()
	manifest, err := content.Extract(bytes.NewReader(tarball), root, content.ExtractOptions{DryRun: true})
	if err != nil {
		t.Fatalf("extract: %v", err)
	}
	if manifest.Files != 1 {
		t.Errorf("manifest still counts the file: got %d", manifest.Files)
	}
	if _, statErr := os.Stat(filepath.Join(root, "networks/x.yaml")); !os.IsNotExist(statErr) {
		t.Errorf("dry run wrote a file: %v", statErr)
	}
}

func TestExtractSkipExistingPreservesFiles(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "networks"), 0o755); err != nil {
		t.Fatal(err)
	}
	preExisting := filepath.Join(root, "networks", "x.yaml")
	if err := os.WriteFile(preExisting, []byte("ORIGINAL"), 0o644); err != nil {
		t.Fatal(err)
	}

	tarball := buildTarball(t, []tarballEntry{{name: "networks/x.yaml", body: "FROM_BUNDLE"}})
	opts := content.ExtractOptions{SkipExisting: true}
	if _, err := content.Extract(bytes.NewReader(tarball), root, opts); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(preExisting)
	if string(got) != "ORIGINAL" {
		t.Errorf("SkipExisting did not preserve original: got %q", got)
	}

	if _, err := content.Extract(bytes.NewReader(tarball), root, content.ExtractOptions{}); err != nil {
		t.Fatal(err)
	}
	got, _ = os.ReadFile(preExisting)
	if string(got) != "FROM_BUNDLE" {
		t.Errorf("default overwrite did not replace original: got %q", got)
	}
}

func TestExtractRejectsOversizedEntry(t *testing.T) {
	var raw bytes.Buffer
	gz := gzip.NewWriter(&raw)
	tw := tar.NewWriter(gz)
	hdr := &tar.Header{
		Name:     "networks/huge.yaml",
		Typeflag: tar.TypeReg,
		Size:     maxEntrySize + 1,
		Mode:     0o644,
	}
	if err := tw.WriteHeader(hdr); err != nil {
		t.Fatalf("write header: %v", err)
	}
	if err := tw.Close(); err != nil {
		t.Logf("close tar (expected error since bytes were not written): %v", err)
	}
	if err := gz.Close(); err != nil {
		t.Fatalf("close gz: %v", err)
	}
	_, err := content.Extract(bytes.NewReader(raw.Bytes()), t.TempDir(), content.ExtractOptions{})
	if !errors.Is(err, content.ErrEntryTooLarge) {
		t.Fatalf("want ErrEntryTooLarge, got %v", err)
	}
}

func TestExtractRejectsSymlinkEntries(t *testing.T) {
	var raw bytes.Buffer
	gz := gzip.NewWriter(&raw)
	tw := tar.NewWriter(gz)
	hdr := &tar.Header{
		Name:     "networks/link",
		Typeflag: tar.TypeSymlink,
		Linkname: "/etc/passwd",
		Mode:     0o644,
	}
	if err := tw.WriteHeader(hdr); err != nil {
		t.Fatalf("write header: %v", err)
	}
	if err := tw.Close(); err != nil {
		t.Fatalf("close tar: %v", err)
	}
	if err := gz.Close(); err != nil {
		t.Fatalf("close gz: %v", err)
	}
	_, err := content.Extract(bytes.NewReader(raw.Bytes()), t.TempDir(), content.ExtractOptions{})
	if !errors.Is(err, content.ErrUnsupportedType) {
		t.Fatalf("want ErrUnsupportedType, got %v", err)
	}
}

// TestExtractRejectsSymlinkTraversal proves the os.Root containment
// closes a symlink-TOCTOU escape that the old string prefix check could
// not: a symlink planted inside the library root pointing outside it
// must not let a bundle entry write through it. The cleaned path
// "networks/evil/pwned.yaml" is textually under the root (so the old
// HasPrefix check passed and would have written to the symlink target);
// os.Root refuses to traverse the escaping symlink at the syscall.
func TestExtractRejectsSymlinkTraversal(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()

	if err := os.MkdirAll(filepath.Join(root, "networks"), 0o755); err != nil {
		t.Fatalf("mkdir networks: %v", err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "networks", "evil")); err != nil {
		t.Skipf("symlinks unsupported on this platform: %v", err)
	}

	tarball := buildTarball(t, []tarballEntry{
		{name: "networks/evil/pwned.yaml", body: "owned"},
	})

	_, err := content.Extract(bytes.NewReader(tarball), root, content.ExtractOptions{})
	if err == nil {
		t.Fatal("expected extraction through an escaping symlink to be rejected, got nil")
	}
	if _, statErr := os.Stat(filepath.Join(outside, "pwned.yaml")); statErr == nil {
		t.Fatal("file escaped the library root through the symlink")
	}
}

func TestExtractRoundTripPreservesContent(t *testing.T) {
	want := "name: example\ndevices: []\n"
	tarball := buildTarball(t, []tarballEntry{{name: "networks/x.yaml", body: want}})
	root := t.TempDir()
	if _, err := content.Extract(bytes.NewReader(tarball), root, content.ExtractOptions{}); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(root, "networks/x.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != want {
		t.Errorf("round-trip mismatch.\n want %q\n  got %q", want, got)
	}
}

// Sanity: the test helper itself should produce a valid bundle.
func TestBuildTarballYieldsValidGzip(t *testing.T) {
	bs := buildTarball(t, []tarballEntry{{name: "networks/x.yaml", body: "x"}})
	gz, err := gzip.NewReader(bytes.NewReader(bs))
	if err != nil {
		t.Fatalf("gzip reader: %v", err)
	}
	if _, readErr := io.ReadAll(gz); readErr != nil {
		t.Fatalf("read all: %v", readErr)
	}
}
