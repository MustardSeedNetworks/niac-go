package sse

import (
	"context"
	"log/slog"
	"os"
	"strings"
	"sync"
	"sync/atomic"
)

// logHandler is a slog.Handler that tees every record through the
// upstream handler AND broadcasts it on the SSE logs stream so the
// Protocol Debug Console page sees daemon logs in real time.
//
// The handler is installed exactly once per process via InstallLogTee.
// Subsequent calls (a second daemon in tests, hot reload, etc.) rotate
// the active hub via an atomic pointer instead of wrapping slog.Default
// again. That keeps the handler chain shallow no matter how many daemons
// the test suite spins up.
//
// The stack's own diagnostics go through internal/logging, which writes to
// its own io.Writer rather than slog, so they still do not appear here.
// (They used to go straight to os.Stdout; niac#1805 moved them onto that
// writer so `daemon --once` can keep stdout clean, which is a different
// problem from making them SSE-visible.) This captures everything that
// routes through slog.Default, which is most of the newer code.
type logHandler struct {
	next slog.Handler
	// hub is read via atomic.Pointer so daemon shutdown / re-init can
	// swap or clear the active broadcast target without rebuilding the
	// slog handler chain.
	hub atomic.Pointer[Hub]
}

// getLogTee is a package-global because slog.SetDefault is itself a
// process-global. The install-once + atomic hub rotation pattern is
// what prevents the test-suite deadlock described on InstallLogTee; any
// alternative that stuffs state somewhere else would either duplicate
// the chain (the original bug) or hide globals inside an unexported
// singleton with no practical difference.
//
//nolint:gochecknoglobals // process-global tee paired with slog.SetDefault.
var getLogTee = sync.OnceValue(func() *logHandler {
	h := &logHandler{
		next: slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}),
	}
	slog.SetDefault(slog.New(h))
	return h
})

// InstallLogTee makes sure a single logHandler is wrapped around
// the slog default and points its broadcast target at hub. Safe to call
// multiple times — subsequent calls just rotate the active hub. Pass
// nil to detach (e.g. on daemon shutdown).
//
// The "next" handler is intentionally a fresh text handler writing
// straight to stderr rather than the existing slog default. Reason:
// Go's log/log.go patches the standard log package at init so its
// output writer routes through slog.Default().Handler. If sseLogTee
// preserved slog.Default()'s prior handler (which is the package
// default — defaultHandler), every log it emits would run:
//
//	slog.* → sseLogTee → defaultHandler → log.Output → log writer
//	  → slog.Default()  ← which is sseLogTee again
//
// Same-goroutine recursion on log.mu, deadlocking the hub goroutine
// during shutdown. Writing straight to stderr breaks the cycle and
// preserves the operator-log behaviour callers expect.
//
// Idempotent + atomic-rotation is also the reentrancy strategy.
// The previous design re-wrapped slog.Default on every daemon.Start(),
// which in the test suite produced a chain that grew one layer per
// test and eventually hung shutdown.
func InstallLogTee(hub *Hub) {
	getLogTee().hub.Store(hub)
}

// NewLogHandler is retained for callers / tests that want a fresh
// wrapper without touching the process-wide default. The returned
// handler has its own internal hub pointer; callers can update it via
// SetHub. A nil hub falls through to next.
//
// Production code should prefer InstallLogTee — it makes sure the
// process has exactly one tee no matter how many times the daemon
// restarts.
func NewLogHandler(hub *Hub, next slog.Handler) slog.Handler {
	if next == nil {
		next = slog.Default().Handler()
	}
	h := &logHandler{next: next}
	h.hub.Store(hub)
	return h
}

func (h *logHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.next.Enabled(ctx, level)
}

// internalLogPrefix tags log lines emitted by the SSE subsystem
// itself ("[SSE] Client connected", "[SSE] Failed to marshal message",
// etc). These must NOT be teed back to the hub or we'd recurse —
// Broadcast emits no slog calls on the success path, but the hub's
// own register/unregister/broadcast log lines do, and they fire from
// inside the hub's goroutine. Filtering by message prefix is a tiny,
// inspection-friendly cycle-breaker.
const internalLogPrefix = "[SSE]"

func (h *logHandler) Handle(ctx context.Context, rec slog.Record) error {
	// Tee to the SSE hub first so a downstream handler error doesn't
	// silently drop the broadcast. Skip SSE-internal lines to break
	// the tee → BroadcastLog → hub-logs → tee → ... cycle.
	if hub := h.hub.Load(); hub != nil && !strings.HasPrefix(rec.Message, internalLogPrefix) {
		hub.BroadcastLog(levelString(rec.Level), formatRecord(rec))
	}
	return h.next.Handle(ctx, rec)
}

// WithAttrs and WithGroup intentionally return a fresh wrapper that
// SHARES the same hub atomic pointer, not a copy of it. Otherwise
// loggers derived via slog.With(...) would freeze the hub at the
// moment of derivation and miss subsequent rotations.
func (h *logHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	clone := &logHandler{next: h.next.WithAttrs(attrs)}
	clone.hub.Store(h.hub.Load())
	return clone
}

func (h *logHandler) WithGroup(name string) slog.Handler {
	clone := &logHandler{next: h.next.WithGroup(name)}
	clone.hub.Store(h.hub.Load())
	return clone
}

func levelString(l slog.Level) string {
	switch {
	case l >= slog.LevelError:
		return "error"
	case l >= slog.LevelWarn:
		return "warn"
	case l >= slog.LevelInfo:
		return "info"
	default:
		return "debug"
	}
}

// formatRecord builds a "<msg> key1=v1 key2=v2" string from the record.
// Matches the shape the Debug Console UI already renders.
func formatRecord(rec slog.Record) string {
	if rec.NumAttrs() == 0 {
		return rec.Message
	}
	var b strings.Builder
	b.WriteString(rec.Message)
	rec.Attrs(func(a slog.Attr) bool {
		b.WriteByte(' ')
		b.WriteString(a.Key)
		b.WriteByte('=')
		b.WriteString(a.Value.String())
		return true
	})
	return b.String()
}
