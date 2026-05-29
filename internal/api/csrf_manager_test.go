// SPDX-License-Identifier: BUSL-1.1

package api

import (
	"errors"
	"strings"
	"testing"
)

// TestCSRFManager_PerSessionIsolation is the load-bearing #1257 test:
// the pre-port single-token model let any client replay another's
// token. Per-session means a token minted for session A must be
// invalid for session B even with identical content.
func TestCSRFManager_PerSessionIsolation(t *testing.T) {
	t.Parallel()
	m := NewCSRFManager()
	t.Cleanup(m.Stop)

	tokenA, genErrA := m.Generate(SessionKey("bearer-A"))
	if genErrA != nil {
		t.Fatalf("generate A: %v", genErrA)
	}
	tokenB, genErrB := m.Generate(SessionKey("bearer-B"))
	if genErrB != nil {
		t.Fatalf("generate B: %v", genErrB)
	}
	if tokenA == tokenB {
		t.Fatal("two sessions must not share a CSRF token")
	}

	if vErr := m.Validate(SessionKey("bearer-A"), tokenA); vErr != nil {
		t.Errorf("A's token must validate against A's session: %v", vErr)
	}
	if vErr := m.Validate(SessionKey("bearer-B"), tokenB); vErr != nil {
		t.Errorf("B's token must validate against B's session: %v", vErr)
	}
	// The actual regression guard.
	if vErr := m.Validate(SessionKey("bearer-A"), tokenB); !errors.Is(vErr, errCSRFTokenInvalid) {
		t.Errorf("B's token presented to A's session must be invalid, got %v", vErr)
	}
	if vErr := m.Validate(SessionKey("bearer-B"), tokenA); !errors.Is(vErr, errCSRFTokenInvalid) {
		t.Errorf("A's token presented to B's session must be invalid, got %v", vErr)
	}
}

// TestCSRFManager_LoopbackBypassSharesKey proves that the no-token
// bypass path (loopback or NIAC_AUTH_DISABLED) maps every request to
// a single key — different empty bearers still share state, which
// preserves the pre-port behavior for that dev-only path.
func TestCSRFManager_LoopbackBypassSharesKey(t *testing.T) {
	t.Parallel()
	if SessionKey("") != csrfLoopbackSessionKey {
		t.Errorf("empty bearer should map to loopback key, got %q", SessionKey(""))
	}
}

// TestCSRFManager_ValidateErrorClassification proves the three failure
// modes are distinguishable so the middleware can render different
// error codes / messages per cause.
func TestCSRFManager_ValidateErrorClassification(t *testing.T) {
	t.Parallel()
	m := NewCSRFManager()
	t.Cleanup(m.Stop)

	// Missing.
	if err := m.Validate("session", ""); !errors.Is(err, errCSRFTokenMissing) {
		t.Errorf("empty token => %v, want errCSRFTokenMissing", err)
	}
	// Invalid (no token ever minted for this session).
	if err := m.Validate("session-with-no-mint", "anything"); !errors.Is(err, errCSRFTokenInvalid) {
		t.Errorf("unminted session => %v, want errCSRFTokenInvalid", err)
	}
	// Wrong value for a minted session.
	tok, genErr := m.Generate("session2")
	if genErr != nil {
		t.Fatalf("generate: %v", genErr)
	}
	if vErr := m.Validate("session2", tok+"X"); !errors.Is(vErr, errCSRFTokenInvalid) {
		t.Errorf("wrong token => %v, want errCSRFTokenInvalid", vErr)
	}
}

// TestCSRFManager_GetOrCreateIsIdempotent proves a UI polling
// /api/v1/csrf-token within the expiry window receives the same
// token back — important so the UI can cache and not invalidate
// every in-flight request when it polls.
func TestCSRFManager_GetOrCreateIsIdempotent(t *testing.T) {
	t.Parallel()
	m := NewCSRFManager()
	t.Cleanup(m.Stop)

	first, err := m.GetOrCreate("session")
	if err != nil {
		t.Fatalf("first: %v", err)
	}
	second, err := m.GetOrCreate("session")
	if err != nil {
		t.Fatalf("second: %v", err)
	}
	if first != second {
		t.Errorf("repeat GetOrCreate must return same token: %q vs %q", first, second)
	}
}

// TestSessionKey_HashedNotPlaintext proves the bearer never lands in
// the manager's internal map as plaintext — important so a heap dump
// or debug log of the manager doesn't leak live bearer tokens.
func TestSessionKey_HashedNotPlaintext(t *testing.T) {
	t.Parallel()
	bearer := "extremely-secret-bearer-token"
	key := SessionKey(bearer)
	if strings.Contains(key, bearer) {
		t.Errorf("session key must not embed bearer plaintext (key=%q)", key)
	}
	if len(key) != 64 { // sha256 hex = 64 chars
		t.Errorf("session key length = %d, want 64 (sha256 hex)", len(key))
	}
}
