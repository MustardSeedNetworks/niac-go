package support_test

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"testing"
)

// traversalArchive builds a gzipped tar holding one entry under the given
// name, so a test can hand Restore a name no honest Backup would produce.
func traversalArchive(t *testing.T, name string) []byte {
	t.Helper()

	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	body := []byte("devices: []\n")
	header := &tar.Header{
		Name:     name,
		Mode:     0o644,
		Size:     int64(len(body)),
		Typeflag: tar.TypeReg,
	}
	if err := tw.WriteHeader(header); err != nil {
		t.Fatalf("write header: %v", err)
	}
	if _, err := tw.Write(body); err != nil {
		t.Fatalf("write body: %v", err)
	}
	if err := tw.Close(); err != nil {
		t.Fatalf("close tar: %v", err)
	}
	if err := gz.Close(); err != nil {
		t.Fatalf("close gzip: %v", err)
	}
	return buf.Bytes()
}
