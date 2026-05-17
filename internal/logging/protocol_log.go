package logging

import (
	"context"
	"fmt"
	"log/slog"
	"os"
)

// ProtocolLogf emits a single line via both stdout (preserving the
// long-standing operator log format that pipes through `niac logs`
// and journalctl) AND slog (so the daemon's SSE log tee can ship the
// message to the Debug Console page in the web UI).
//
// Until this helper existed, protocol handlers only wrote to stdout
// via fmt.Fprintf, which meant the Debug Console — wired through
// slog → SSE — never saw per-packet activity. Operators reported the
// console as "empty" even when the simulator was sending hundreds of
// LLDP/CDP/EDP/FDP frames per minute.
//
// Format args are Sprintf-formatted; the trailing newline is added
// for the stdout copy (slog records don't need it).
//
// Levels map straight onto slog's standard levels. Callers can also
// use plain slog.* if they already have an attribute-keyed record to
// emit — this helper exists for the legacy "format string + args"
// callers that dominate the protocols package.
func ProtocolLogf(protocol, level, format string, args ...any) {
	msg := fmt.Sprintf(format, args...)

	// Stdout copy: preserves "PROTOCOL: ..." prefix so existing
	// log-scraping tools (operator's grep / journalctl filters) keep
	// working without surprise.
	_, _ = fmt.Fprintf(os.Stdout, "%s: %s\n", protocol, msg)

	// SSE-visible copy via slog. Tagged with the protocol so the UI
	// can group / filter; the message itself stays unprefixed because
	// the UI renders the protocol column separately.
	slogLevel := slog.LevelInfo
	switch level {
	case "debug":
		slogLevel = slog.LevelDebug
	case "warn":
		slogLevel = slog.LevelWarn
	case "error":
		slogLevel = slog.LevelError
	}
	slog.Default().LogAttrs(context.TODO(), slogLevel, msg, slog.String("protocol", protocol))
}
