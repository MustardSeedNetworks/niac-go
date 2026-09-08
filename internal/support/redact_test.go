package support_test

import (
	"strings"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/support"
)

// A field the operator never set must stay unset: writing a placeholder into
// it would make the bundle claim a credential exists where none does.
func TestRedactLeavesUnsetFieldsAlone(t *testing.T) {
	cfg := &config.Config{Devices: []config.Device{{
		Name:       "sw1",
		SNMPConfig: config.SNMPConfig{SysName: "sw1"},
	}}}

	support.Redact(cfg)

	if got := cfg.Devices[0].SNMPConfig.Community; got != "" {
		t.Errorf("unset community became %q", got)
	}
	if got := cfg.Devices[0].SNMPConfig.SysName; got != "sw1" {
		t.Errorf("sysName is %q, want sw1", got)
	}
}

// ScrubText's patterns are the backstop for a secret this package was never
// told about -- one the daemon logged that no scenario declares.
func TestScrubTextMasksUndeclaredCredentials(t *testing.T) {
	log := strings.Join([]string{
		`msg="get" community=unknown-to-us`,
		`msg="auth" header="Bearer never-seen-token"`,
		`msg="login" password: hunter2`,
		`msg="listening" addr=0.0.0.0:8445`,
	}, "\n")

	scrubbed := support.ScrubText(log, nil)

	for _, secret := range []string{"unknown-to-us", "never-seen-token", "hunter2"} {
		if strings.Contains(scrubbed, secret) {
			t.Errorf("scrubbed text still holds %q:\n%s", secret, scrubbed)
		}
	}
	if !strings.Contains(scrubbed, "addr=0.0.0.0:8445") {
		t.Errorf("scrubbing removed a diagnostic line:\n%s", scrubbed)
	}
}

// A known value is removed wherever it appears, including in text no pattern
// would match -- that is why the caller passes them.
func TestScrubTextRemovesKnownValues(t *testing.T) {
	scrubbed := support.ScrubText("walking sw1 with s3cret then done", []string{"s3cret"})

	if strings.Contains(scrubbed, "s3cret") {
		t.Errorf("known value survived: %s", scrubbed)
	}
	if !strings.Contains(scrubbed, support.Placeholder) {
		t.Errorf("no placeholder written: %s", scrubbed)
	}
}
