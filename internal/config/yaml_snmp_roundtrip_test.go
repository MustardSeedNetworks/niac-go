package config

import (
	"strings"
	"testing"
)

// The New Simulation wizard's Start-empty seed, verbatim from
// ui/src/pages/NewSimulationWizardPage.tsx.
const wizardStartEmptyYAML = `devices:
  - name: new-device
    type: host
    mac: 02:00:00:00:00:01
`

// TestStartEmptyDraftSurvivesTheSaveReloadRoundTrip guards #1460.
//
// Device.SNMPConfig is a value field, so yaml.go's snmpToYAML(&device.SNMPConfig)
// is handed a pointer that is never nil and its cfg == nil guard can never fire.
// Every device was marshalled with a non-nil &converter.SnmpAgent{}, which
// `omitempty` does not omit, so every saved config gained `snmp_agent: {}`.
//
// On reload the present key makes SNMP count as configured, and
// validateSNMPCommunity then demands a community nobody asked for — so a draft
// the wizard itself produced could not pass its own preflight.
//
// This asserts the round trip rather than the marshal output alone, because the
// defect only bites on the second load: the first parse is clean.
func TestStartEmptyDraftSurvivesTheSaveReloadRoundTrip(t *testing.T) {
	cfg, err := LoadYAMLBytes([]byte(wizardStartEmptyYAML))
	if err != nil {
		t.Fatalf("load the wizard seed: %v", err)
	}

	saved, err := MarshalConfigYAML(cfg)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if strings.Contains(string(saved), "snmp_agent") {
		t.Errorf("saved config grew an snmp_agent block the source never had:\n%s", saved)
	}

	reloaded, err := LoadYAMLBytes(saved)
	if err != nil {
		t.Fatalf("reload the saved config: %v", err)
	}

	if verr := NewValidator("draft.yaml").Validate(reloaded); verr != nil && len(verr.Errors) > 0 {
		for _, e := range verr.Errors {
			t.Errorf("round-tripped draft fails validation: %s", e.Error())
		}
	}
}

// A device that really does configure SNMP must still marshal its block — the
// omission is for the entirely-unset case only.
func TestConfiguredSNMPStillMarshals(t *testing.T) {
	cfg, err := LoadYAMLBytes([]byte(`devices:
  - name: sw-1
    type: switch
    mac: 02:00:00:00:00:02
    snmp_agent:
      community: public
      sysname: sw-1
`))
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	saved, err := MarshalConfigYAML(cfg)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	for _, want := range []string{"snmp_agent", "community: public"} {
		if !strings.Contains(string(saved), want) {
			t.Errorf("saved config lost %q:\n%s", want, saved)
		}
	}
}
