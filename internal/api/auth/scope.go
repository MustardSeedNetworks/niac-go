// SPDX-License-Identifier: BUSL-1.1

// Package auth is the api transport's authentication middleware, extracted
// from internal/api as a leaf (ADR-0006). It composes the tokenstore and
// ratelimit leaves behind a request-time bearer-token check + scope model,
// taking the remaining api-specific concerns (security headers, request IDs,
// client-IP extraction, error rendering, non-loopback detection) as injected
// function values so it never imports the api transport layer.
package auth

import (
	"context"

	"github.com/MustardSeedNetworks/niac-go/internal/api/tokenstore"
)

// scopeContextKey is the request-context key under which Middleware stashes the
// resolved tokenstore.TokenScope. AdminProtect (and any future scope-aware
// middleware) reads it via ScopeFromContext so the gate stays a single concern
// rather than re-doing token lookup.
type scopeContextKey struct{}

// WithScope returns a context carrying scope, for the auth middleware to stash
// the resolved token scope (and for tests to simulate it).
func WithScope(ctx context.Context, scope tokenstore.TokenScope) context.Context {
	return context.WithValue(ctx, scopeContextKey{}, scope)
}

// ScopeFromContext returns the scope stashed by Middleware. The second return
// reports whether a value was actually stashed — false means the request never
// went through Middleware (a wiring bug), in which case callers must fail
// closed.
func ScopeFromContext(ctx context.Context) (tokenstore.TokenScope, bool) {
	v, ok := ctx.Value(scopeContextKey{}).(tokenstore.TokenScope)
	return v, ok
}
