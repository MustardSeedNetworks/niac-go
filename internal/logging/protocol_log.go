package logging

import (
	"context"
	"fmt"
	"log/slog"
	"os"
)

// ProtocolLevel is the typed log severity for ProtocolLogf. Using a
// named type instead of a raw string prevents the silent
// "warning"/"INFO"/"err" demotion-to-Info bug the audit flagged:
// any caller passing a string outside the four constants below now
// fails at compile time.
type ProtocolLevel int

// ProtocolLevel constants map 1:1 onto slog's standard severities.
const (
	LevelDebug ProtocolLevel = iota
	LevelInfo
	LevelWarn
	LevelError
)

func (l ProtocolLevel) slogLevel() slog.Level {
	switch l {
	case LevelDebug:
		return slog.LevelDebug
	case LevelWarn:
		return slog.LevelWarn
	case LevelError:
		return slog.LevelError
	case LevelInfo:
		fallthrough
	default:
		return slog.LevelInfo
	}
}

// ProtocolLogf emits a single line via both stdout (preserving the
// long-standing operator log format that pipes through `niac logs`
// and journalctl) AND slog (so the daemon's SSE log tee can ship the
// message to the Debug Console page in the web UI).
//
// ctx threads through to slog so handlers that read context can act
// on it (trace IDs, cancellation, request scoping). Pass
// context.Background() at the top of the protocol stack where there
// is no upstream context; the helper does not accept nil.
//
// Format args are Sprintf-formatted; the trailing newline is added
// for the stdout copy (slog records don't need it).
func ProtocolLogf(ctx context.Context, protocol string, level ProtocolLevel, format string, args ...any) {
	if ctx == nil {
		ctx = context.Background()
	}
	msg := fmt.Sprintf(format, args...)

	// Stdout copy: preserves "PROTOCOL: ..." prefix so existing
	// log-scraping tools (operator's grep / journalctl filters) keep
	// working without surprise.
	_, _ = fmt.Fprintf(os.Stdout, "%s: %s\n", protocol, msg)

	// SSE-visible copy via slog. Tagged with the protocol so the UI
	// can group / filter; the message itself stays unprefixed because
	// the UI renders the protocol column separately.
	slog.Default().LogAttrs(ctx, level.slogLevel(), msg, slog.String("protocol", protocol))
}
