package api

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/walkcapture"
)

// TestWalkCaptureFailureDetail guards #1488.
//
// Every non-validation capture failure returned the same opaque string with a
// nil details slice, so the operator could not tell a timeout from a wrong
// community from a walk that blew a size limit. Driven live on CT304 with a
// deliberately wrong community, the response was exactly:
//
//	502 {"error":"walk_capture_failed","message":"SNMP walk capture failed"}
//
// walkcapture distinguishes these cases and two of them are typed sentinels the
// operator can act on directly. The details are curated rather than echoed, so
// no upstream internals reach the client.
func TestWalkCaptureFailureDetail(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{
			name: "entry limit",
			err:  fmt.Errorf("capture SNMP walk: %w", walkcapture.ErrEntryLimit),
			want: "walk capture exceeded 100000 entries",
		},
		{
			name: "size limit",
			err:  fmt.Errorf("capture SNMP walk: %w", walkcapture.ErrSizeLimit),
			want: "walk capture exceeded 16 MiB",
		},
		{
			name: "unreachable target",
			err:  errors.New("connect to SNMP target: dial udp 10.0.0.1:161: no route to host"),
			want: "could not reach the target on UDP/161",
		},
		{
			name: "no answer or wrong credentials",
			err:  errors.New("capture SNMP walk: request timeout (after 3 retries)"),
			want: "no response from the target",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			details := walkCaptureFailureDetails(tt.err)
			if len(details) != 1 {
				t.Fatalf("got %d details, want exactly one: %+v", len(details), details)
			}
			// Contains, not equality: the assertion pins which case was
			// recognised, not the exact wording of operator-facing copy.
			if got := details[0].Issue; !strings.Contains(got, tt.want) {
				t.Errorf("issue = %q, want it to mention %q", got, tt.want)
			}
		})
	}
}

// The raw upstream text must not reach the client: it is curated per case.
func TestWalkCaptureFailureDetailDoesNotEchoUpstream(t *testing.T) {
	err := errors.New("capture SNMP walk: dial udp 10.44.40.10:161: i/o timeout")
	details := walkCaptureFailureDetails(err)
	if len(details) != 1 {
		t.Fatalf("got %d details, want one", len(details))
	}
	if details[0].Issue == err.Error() {
		t.Error("detail echoed the upstream error verbatim")
	}
}
