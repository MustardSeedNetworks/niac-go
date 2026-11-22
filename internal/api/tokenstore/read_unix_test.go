//go:build !windows

package tokenstore

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// TestTokenStore_RejectsWorldReadableFile asserts LoadTokenFile
// refuses to read a token file whose mode bits are wider than 0o600.
// This is the security invariant the gate enforces — a leaked token
// is worse than no auth.
func TestTokenStore_RejectsWorldReadableFile(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "tokens.json")

	body, err := json.Marshal(tokenFileSchema{
		Tokens: []tokenFileEntry{{Value: "rw-secret", Scope: "read-write"}},
	})
	if err != nil {
		t.Fatalf("marshal token file: %v", err)
	}
	if writeErr := os.WriteFile(path, body, 0o644); writeErr != nil {
		t.Fatalf("write token file: %v", writeErr)
	}

	_, loadErr := LoadTokenFile(path)
	if loadErr == nil {
		t.Fatal("LoadTokenFile accepted a 0o644 file; expected refusal")
	}
	if !errors.Is(loadErr, ErrTokenFileWorldReadable) {
		t.Errorf("LoadTokenFile returned %v, want ErrTokenFileWorldReadable", loadErr)
	}
}
