package scenario_test

import (
	"strings"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/scenario"
)

func TestHospitalCatalogIncludesPhilipsAndGEPatientMonitors(t *testing.T) {
	want := map[string]struct {
		vendor string
		model  string
	}{
		"philips-patient-monitor": {vendor: "philips healthcare", model: "IntelliVue MX850"},
		"ge-patient-monitor":      {vendor: "ge healthcare", model: "CARESCAPE B850"},
	}

	for _, profile := range scenario.Profiles() {
		expected, found := want[profile.Role]
		if !found {
			continue
		}
		if profile.Vendor != expected.vendor || profile.Model != expected.model {
			t.Errorf("%s identity = %q/%q, want %q/%q", profile.Role,
				profile.Vendor, profile.Model, expected.vendor, expected.model)
		}
		delete(want, profile.Role)
	}
	for role := range want {
		t.Errorf("missing hospital profile %q", role)
	}
}

func TestHospitalPackGeneratesPhilipsAndGEPatientMonitors(t *testing.T) {
	pack := hospitalPack(t)
	result, err := scenario.Generate(pack.Request)
	if err != nil {
		t.Fatal(err)
	}
	cfg, err := config.LoadYAMLBytes(result.YAML)
	if err != nil {
		t.Fatal(err)
	}

	want := map[string]struct {
		namePrefix string
		macPrefix  string
	}{
		"philips-patient-monitor": {namePrefix: "MED-PHMX850-", macPrefix: "7c:94:b2:"},
		"ge-patient-monitor":      {namePrefix: "MED-GEB850-", macPrefix: "44:4b:5d:"},
	}
	seen := make(map[string]bool, len(want))
	for _, device := range cfg.Devices {
		role := device.Properties["role"]
		expected, found := want[role]
		if found && strings.HasPrefix(device.Name, expected.namePrefix) &&
			strings.HasPrefix(device.MACAddress.String(), expected.macPrefix) {
			seen[role] = true
		}
	}
	for role := range want {
		if !seen[role] {
			t.Errorf("hospital pack has no correctly named %s", role)
		}
	}
}

func hospitalPack(t *testing.T) scenario.Pack {
	t.Helper()
	for _, pack := range scenario.Packs() {
		if pack.ID == "hospital" {
			return pack
		}
	}
	t.Fatal("hospital pack missing")
	return scenario.Pack{}
}
