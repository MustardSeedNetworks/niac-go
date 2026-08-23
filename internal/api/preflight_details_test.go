package api

import (
	"errors"
	"fmt"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
)

// TestPreflightErrorDetailsEnumeratesValidationErrors guards #1461.
//
// config.ListError.Error() deliberately collapses to "N configuration errors
// found" once there is more than one — fine for a single-line CLI string, but
// preflight consumed only that string and threw away ListError.Errors, which
// carries a structured *config.Error per finding with the field path on it.
//
// The operator saw "Simulation preflight failed" and nothing else; diagnosing
// the two real errors meant reproducing the config offline and running the
// validator by hand.
func TestPreflightErrorDetailsEnumeratesValidationErrors(t *testing.T) {
	listErr := &config.ListError{}
	listErr.Add(config.NewConfigError(
		"draft.yaml", "devices[0].snmp_agent.community", "SNMPv1/v2c requires an explicit community"))
	listErr.Add(config.NewConfigError(
		"draft.yaml", "devices[1].snmp_agent.community", "SNMPv1/v2c requires an explicit community"))

	// Preflight wraps the validator's error before it reaches the handler.
	wrapped := fmt.Errorf("simulation configuration failed semantic validation: %w", listErr)

	details := preflightErrorDetails(wrapped)

	if len(details) != 2 {
		t.Fatalf("got %d details, want one per validation error: %+v", len(details), details)
	}
	for i, want := range []string{"devices[0].snmp_agent.community", "devices[1].snmp_agent.community"} {
		if details[i].Field != want {
			t.Errorf("details[%d].Field = %q, want %q", i, details[i].Field, want)
		}
		if details[i].Issue != "SNMPv1/v2c requires an explicit community" {
			t.Errorf("details[%d].Issue = %q, want the underlying message", i, details[i].Issue)
		}
	}
}

// Errors that are not validation lists keep the existing "; "-splitting
// behaviour, which several non-validation preflight failures rely on.
func TestPreflightErrorDetailsStillSplitsPlainErrors(t *testing.T) {
	details := preflightErrorDetails(errors.New("first problem; second problem"))

	if len(details) != 2 {
		t.Fatalf("got %d details, want 2: %+v", len(details), details)
	}
	if details[0].Issue != "first problem" || details[1].Issue != "second problem" {
		t.Errorf("unexpected split: %+v", details)
	}
}
