package templates_test

import (
	"testing"

	"gopkg.in/yaml.v3"

	tmpl "github.com/MustardSeedNetworks/niac-go/internal/templates"
)

// TestBuiltinTemplatesTypeEveryDevice guards the wire, not the icons.
//
// An absent device type is not a cosmetic gap: buildSystemCapabilitiesTLV and
// buildCapabilitiesTLV both switch on it and fall to a default of
// LLDPCapStationOnly / CDPCapHost. Eight of the eleven builtin templates once
// shipped with no type on any device, so `niac template use enterprise-campus`
// produced a campus whose core routers announced themselves to every discovery
// tool as end stations (#2096).
func TestBuiltinTemplatesTypeEveryDevice(t *testing.T) {
	for _, meta := range tmpl.List() {
		template, err := tmpl.Get(meta.Name)
		if err != nil {
			t.Fatalf("get %s: %v", meta.Name, err)
		}

		var doc struct {
			Devices []struct {
				Name string `yaml:"name"`
				Type string `yaml:"type"`
			} `yaml:"devices"`
		}
		if unmarshalErr := yaml.Unmarshal([]byte(template.Content), &doc); unmarshalErr != nil {
			t.Fatalf("unmarshal %s: %v", meta.Name, unmarshalErr)
		}

		if len(doc.Devices) == 0 {
			t.Errorf("template %s declares no devices", meta.Name)
			continue
		}

		for _, device := range doc.Devices {
			if device.Type == "" {
				t.Errorf(
					"template %s: device %q has no type, so it advertises LLDP station-only and CDP host",
					meta.Name, device.Name,
				)
			}
		}
	}
}
