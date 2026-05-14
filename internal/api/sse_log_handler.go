package api

import (
	"context"
	"log/slog"
	"strings"
)

// sseLogHandler is a slog.Handler that tees every record through the
// upstream handler AND broadcasts it on the SSE logs stream so the
// Protocol Debug Console page sees daemon logs in real time.
//
// The stack used to emit lots of fmt.Fprintf(os.Stdout, ...) lines that
// are invisible to slog — those won't appear here. Migrating those to
// slog is a separate cleanup. For now this captures everything that
// already routes through slog.Default (which is most of the new code).
type sseLogHandler struct {
	hub  *SSEHub
	next slog.Handler
}

// NewSSELogHandler wraps the given handler. The wrapper forwards every
// record to next AND to hub.BroadcastLog. A nil hub falls through to
// next so callers can install the wrapper unconditionally.
func NewSSELogHandler(hub *SSEHub, next slog.Handler) slog.Handler {
	if next == nil {
		next = slog.Default().Handler()
	}
	return &sseLogHandler{hub: hub, next: next}
}

func (h *sseLogHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.next.Enabled(ctx, level)
}

func (h *sseLogHandler) Handle(ctx context.Context, rec slog.Record) error {
	// Tee to the SSE hub first so a downstream handler error doesn't
	// silently drop the broadcast.
	if h.hub != nil {
		h.hub.BroadcastLog(levelString(rec.Level), formatRecord(rec))
	}
	return h.next.Handle(ctx, rec)
}

func (h *sseLogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &sseLogHandler{hub: h.hub, next: h.next.WithAttrs(attrs)}
}

func (h *sseLogHandler) WithGroup(name string) slog.Handler {
	return &sseLogHandler{hub: h.hub, next: h.next.WithGroup(name)}
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
