package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"testing"
)

// TestTokenStore_LookupAndReplace exercises both the lock-free read
// path and the serialised Replace path, with concurrent readers
// running across a Replace. The test passes if (a) every lookup either
// returns the old scope, the new scope, or false (no torn reads) and
// (b) after Replace settles, the old token is gone and the new one
// resolves to its declared scope.
// TestTokenStore_InitialLookups covers the lock-free read path with a
// static token set — both hits and misses.
func TestTokenStore_InitialLookups(t *testing.T) {
	store := NewTokenStore([]ScopedToken{
		{Value: "old-ro", Scope: ScopeReadOnly},
		{Value: "old-rw", Scope: ScopeReadWrite},
	})
	if _, ok := store.Lookup("old-ro"); !ok {
		t.Fatalf("old-ro not in initial store")
	}
	if _, ok := store.Lookup("old-rw"); !ok {
		t.Fatalf("old-rw not in initial store")
	}
	if _, ok := store.Lookup("never-set"); ok {
		t.Fatalf("never-set unexpectedly present")
	}
}

// readerLoop hammers Lookup on three keys and reports any torn read
// (a lookup that returns ok=true but with an inconsistent scope).
// Extracted from the main test body so gocognit can see the inner
// goroutine as a single thunk rather than nested control flow.
func readerLoop(t *testing.T, store *TokenStore, iterations int) {
	t.Helper()
	for range iterations {
		if scope, ok := store.Lookup("old-ro"); ok && scope != ScopeReadOnly {
			t.Errorf("old-ro lookup returned wrong scope: %v", scope)
		}
		if scope, ok := store.Lookup("old-rw"); ok && scope != ScopeReadWrite {
			t.Errorf("old-rw lookup returned wrong scope: %v", scope)
		}
		if scope, ok := store.Lookup("new-rw"); ok && scope != ScopeReadWrite {
			t.Errorf("new-rw lookup returned wrong scope: %v", scope)
		}
	}
}

// TestTokenStore_LookupAndReplace exercises both the lock-free read
// path and the serialised Replace path, with concurrent readers
// running across a Replace. The test passes if (a) every lookup either
// returns the old scope, the new scope, or false (no torn reads) and
// (b) after Replace settles, the old token is gone and the new one
// resolves to its declared scope.
func TestTokenStore_LookupAndReplace(t *testing.T) {
	store := NewTokenStore([]ScopedToken{
		{Value: "old-ro", Scope: ScopeReadOnly},
		{Value: "old-rw", Scope: ScopeReadWrite},
	})

	const readers = 16
	const iterations = 1000

	var wg sync.WaitGroup
	wg.Add(readers + 1)

	// Writer: half-way through, swap the entire token set.
	go func() {
		defer wg.Done()
		store.Replace([]ScopedToken{
			{Value: "new-rw", Scope: ScopeReadWrite},
		})
	}()

	for range readers {
		go func() {
			defer wg.Done()
			readerLoop(t, store, iterations)
		}()
	}

	wg.Wait()

	// Post-Replace: old tokens must be gone, new token must be present.
	if _, ok := store.Lookup("old-ro"); ok {
		t.Errorf("old-ro still present after Replace")
	}
	if _, ok := store.Lookup("old-rw"); ok {
		t.Errorf("old-rw still present after Replace")
	}
	scope, ok := store.Lookup("new-rw")
	if !ok {
		t.Fatalf("new-rw missing after Replace")
	}
	if scope != ScopeReadWrite {
		t.Errorf("new-rw scope = %v, want ScopeReadWrite", scope)
	}
	if got := store.Len(); got != 1 {
		t.Errorf("store size after Replace = %d, want 1", got)
	}
	ro, rw, admin := store.ScopeCounts()
	if ro != 0 || rw != 1 || admin != 0 {
		t.Errorf("ScopeCounts after Replace = (ro=%d, rw=%d, admin=%d), want (0, 1, 0)", ro, rw, admin)
	}
}

// TestTokenStore_EmptyStore covers the unauth-default path. An empty
// store must report Len() == 0 and refuse any Lookup, so the auth
// middleware short-circuits to the unauthed branch only when the store
// is truly empty.
func TestTokenStore_EmptyStore(t *testing.T) {
	store := NewTokenStore(nil)
	if store.Len() != 0 {
		t.Errorf("empty store: Len = %d, want 0", store.Len())
	}
	if _, ok := store.Lookup("anything"); ok {
		t.Errorf("empty store accepted lookup")
	}
}

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

// TestTokenStore_LoadTokenFile_HappyPath covers the canonical
// successful load: 0o600 file, two entries (RO + RW), correctly
// decoded into ScopedToken values that the auth middleware can use.
func TestTokenStore_LoadTokenFile_HappyPath(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "tokens.json")

	body := `{"tokens":[
		{"value":"ro-secret","scope":"read-only"},
		{"value":"rw-secret","scope":"read-write"}
	]}`
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("write token file: %v", err)
	}

	tokens, err := LoadTokenFile(path)
	if err != nil {
		t.Fatalf("LoadTokenFile: %v", err)
	}
	if len(tokens) != 2 {
		t.Fatalf("decoded token count = %d, want 2", len(tokens))
	}
	wantScopes := map[string]TokenScope{
		"ro-secret": ScopeReadOnly,
		"rw-secret": ScopeReadWrite,
	}
	for _, tok := range tokens {
		want, ok := wantScopes[tok.Value]
		if !ok {
			t.Errorf("unexpected token %q", tok.Value)
			continue
		}
		if tok.Scope != want {
			t.Errorf("token %q scope = %v, want %v", tok.Value, tok.Scope, want)
		}
	}
}

// TestTokenStore_LoadTokenFile_RejectsEmptyValue asserts an empty
// "value" field is a load error — an empty-token entry would silently
// degrade the auth check to always-pass.
func TestTokenStore_LoadTokenFile_RejectsEmptyValue(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "tokens.json")

	body := `{"tokens":[{"value":"","scope":"read-write"}]}`
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("write token file: %v", err)
	}

	_, err := LoadTokenFile(path)
	if err == nil {
		t.Fatal("LoadTokenFile accepted an empty value; expected refusal")
	}
	if !errors.Is(err, ErrEmptyTokenValue) {
		t.Errorf("LoadTokenFile returned %v, want ErrEmptyTokenValue", err)
	}
}

// TestTokenStore_LoadTokenFile_RejectsInvalidScope asserts an unknown
// scope literal is rejected at load time so operators get a clear
// error instead of a token that silently downgrades to RO.
func TestTokenStore_LoadTokenFile_RejectsInvalidScope(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "tokens.json")

	// "admin" became a valid scope in #743; "superuser" remains
	// unknown and exercises the reject path.
	body := `{"tokens":[{"value":"x","scope":"superuser"}]}`
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("write token file: %v", err)
	}

	_, err := LoadTokenFile(path)
	if err == nil {
		t.Fatal("LoadTokenFile accepted an unknown scope; expected refusal")
	}
	if !errors.Is(err, ErrInvalidTokenScope) {
		t.Errorf("LoadTokenFile returned %v, want ErrInvalidTokenScope", err)
	}
}

// TestRequiredScopeForMethod covers the HTTP-method-to-scope mapping
// so a future change to the safety table is caught by CI before it
// reaches the middleware.
func TestRequiredScopeForMethod(t *testing.T) {
	cases := []struct {
		method string
		want   TokenScope
	}{
		{http.MethodGet, ScopeReadOnly},
		{http.MethodHead, ScopeReadOnly},
		{http.MethodOptions, ScopeReadOnly},
		{"TRACE", ScopeReadOnly},
		{http.MethodPost, ScopeReadWrite},
		{http.MethodPut, ScopeReadWrite},
		{http.MethodPatch, ScopeReadWrite},
		{http.MethodDelete, ScopeReadWrite},
		{"CONNECT", ScopeReadWrite},           // unknown -> fail closed
		{"PROPFIND", ScopeReadWrite},          // WebDAV: unknown -> fail closed
		{"" + "lowercaseget", ScopeReadWrite}, // genuinely unknown
	}
	for _, c := range cases {
		t.Run(c.method+"_"+strconv.Itoa(int(c.want)), func(t *testing.T) {
			if got := RequiredScopeForMethod(c.method); got != c.want {
				t.Errorf("RequiredScopeForMethod(%q) = %v, want %v", c.method, got, c.want)
			}
		})
	}
}

// TestAuthMiddleware_PublicPaths_NoAuthRequired confirms /__version
// stays unauthenticated even with a populated TokenStore. The Wave 2
// allowlist is identical to Wave 1's — the only public endpoint is
// /__version (and /healthz if it ever lands).
func TestAuthMiddleware_PublicPaths_NoAuthRequired(t *testing.T) {
	server := createTestServerForMiddleware(t)
	server.SetTokens([]ScopedToken{{Value: "rw-secret", Scope: ScopeReadWrite}})
	server.rateLimiter = NewRateLimiter(100, 200)

	// /__version is wired without s.auth() in registerAPIRoutes, so it
	// reaches the handler directly. The test mirrors that wiring by
	// invoking the build-version handler without the auth wrapper.
	req := httptest.NewRequest(http.MethodGet, "/__version", nil)
	req.RemoteAddr = "192.168.1.1:1234"
	rec := httptest.NewRecorder()

	server.recoverMiddleware(server.handleBuildVersion)(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("/__version unauthed: status = %d, want 200", rec.Code)
	}
}
