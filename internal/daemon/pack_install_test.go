package daemon

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/content"
	"github.com/MustardSeedNetworks/niac-go/internal/library"
)

func packBundle(t *testing.T, names ...string) []byte {
	t.Helper()
	var raw bytes.Buffer
	gz := gzip.NewWriter(&raw)
	tw := tar.NewWriter(gz)
	for _, name := range names {
		body := name + "\n"
		hdr := &tar.Header{
			Name: name, Mode: 0o644,
			Size: int64(len(body)), Typeflag: tar.TypeReg,
		}
		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatalf("write tar header: %v", err)
		}
		if _, err := tw.Write([]byte(body)); err != nil {
			t.Fatalf("write tar body: %v", err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatalf("close tar: %v", err)
	}
	if err := gz.Close(); err != nil {
		t.Fatalf("close gzip: %v", err)
	}
	return raw.Bytes()
}

func packRoot(t *testing.T) string {
	t.Helper()
	root := filepath.Join(t.TempDir(), "library")
	if _, err := library.Open(root); err != nil {
		t.Fatalf("open library: %v", err)
	}
	return root
}

// TestInstallPackWritesTheBundle is the chokepoint doing its one job.
func TestInstallPackWritesTheBundle(t *testing.T) {
	root := packRoot(t)

	manifest, err := InstallPack(
		bytes.NewReader(packBundle(t, "walks/switch1.walk")), root, content.ExtractOptions{})
	if err != nil {
		t.Fatalf("install pack: %v", err)
	}
	if manifest.Files != 1 {
		t.Fatalf("manifest.Files = %d, want 1", manifest.Files)
	}
	if _, statErr := os.Stat(filepath.Join(root, "walks", "switch1.walk")); statErr != nil {
		t.Fatalf("bundle did not land: %v", statErr)
	}
}

// TestDaemonInstallPackSerialisesConcurrentInstalls pins why the daemon owns
// the write. Extraction finishes by folding what it wrote into the library's
// bundle index, and that fold is a read-modify-write of one shared file: two
// installs running at once both read the index, then both write it, and the
// slower write drops the faster one's entries. Files stay on disk with no
// record that a bundle put them there, so the next install mistakes them for
// the operator's own and preserves them instead of updating them.
//
// Each installer here ships its own file, so a lost update is visible as a
// missing index entry.
func TestDaemonInstallPackSerialisesConcurrentInstalls(t *testing.T) {
	root := packRoot(t)
	d := &Daemon{}

	const installers = 8
	var wg sync.WaitGroup
	errs := make([]error, installers)
	start := make(chan struct{})
	for i := range installers {
		wg.Go(func() {
			name := fmt.Sprintf("walks/switch%d.walk", i)
			<-start
			_, errs[i] = d.InstallPack(
				bytes.NewReader(packBundle(t, name)), root, content.ExtractOptions{})
		})
	}
	close(start)
	wg.Wait()

	for i, err := range errs {
		if err != nil {
			t.Fatalf("installer %d: %v", i, err)
		}
	}

	index, err := library.ReadBundleIndex(root)
	if err != nil {
		t.Fatalf("read bundle index: %v", err)
	}
	for i := range installers {
		name := fmt.Sprintf("walks/switch%d.walk", i)
		if _, statErr := os.Stat(filepath.Join(root, filepath.FromSlash(name))); statErr != nil {
			t.Fatalf("%s did not land: %v", name, statErr)
		}
		if _, recorded := index.Files[name]; !recorded {
			t.Fatalf(
				"%s is on disk but missing from the bundle index: a concurrent "+
					"install dropped it (index has %d of %d entries)",
				name, len(index.Files), installers)
		}
	}
}
