package scenario_test

import (
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/MustardSeedNetworks/niac-go/internal/deviceclass"
	"github.com/MustardSeedNetworks/niac-go/internal/scenario"
	tmpl "github.com/MustardSeedNetworks/niac-go/internal/templates"
)

// The shape this check reads. Parsed straight from the authored YAML rather
// than loaded through config: loading resolves walk files, and two templates
// reference captures that live outside a test's temp directory.
type discoveryDoc struct {
	Devices []discoveryDevice `yaml:"devices"`
}

type discoveryDevice struct {
	Name       string          `yaml:"name"`
	Type       string          `yaml:"type"`
	Lldp       *enabledBlock   `yaml:"lldp"`
	Cdp        *enabledBlock   `yaml:"cdp"`
	Edp        *enabledBlock   `yaml:"edp"`
	Fdp        *enabledBlock   `yaml:"fdp"`
	TrunkPorts []discoveryPort `yaml:"trunk_ports"`
}

type enabledBlock struct {
	Enabled bool `yaml:"enabled"`
}

type discoveryPort struct {
	RemoteDevice string `yaml:"remote_device"`
	FDBOnly      bool   `yaml:"fdb_only"`
}

// TestDeclaredLinksAreDiscoverable is the parity gate, in the cheapest form
// that needs no capture.
//
// A scenario declares its topology in trunk_ports, and an analyzer only ever
// learns that topology by hearing both ends announce themselves. If a declared
// link has an end that speaks no discovery protocol, the link exists in the
// config and can never exist on the wire — NIAC's own map draws it from the
// config and Seed or Link-Live never will, and the two disagree for a reason
// nobody can see by looking at either one.
//
// This is the static half of the check. The runtime half — that the link is
// actually discovered when the scenario runs — needs the wire harness.
func TestDeclaredLinksAreDiscoverable(t *testing.T) {
	for _, pack := range scenario.Packs() {
		result, err := scenario.Generate(pack.Request)
		if err != nil {
			t.Fatalf("generate %s: %v", pack.ID, err)
		}
		assertDiscoverable(t, "pack:"+pack.ID, parseDoc(t, result.YAML))
	}

	for _, meta := range tmpl.List() {
		template, err := tmpl.Get(meta.Name)
		if err != nil {
			t.Fatalf("get %s: %v", meta.Name, err)
		}
		assertDiscoverable(t, "template:"+meta.Name, parseDoc(t, []byte(template.Content)))
	}
}

func parseDoc(t *testing.T, raw []byte) discoveryDoc {
	t.Helper()
	var doc discoveryDoc
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	return doc
}

// speaksDiscovery reports whether a device announces itself over any protocol
// an analyzer reads. The rules are the emitters': the vendor protocols opt in
// explicitly, LLDP is on by default for the types that ship it that way.
func speaksDiscovery(device discoveryDevice) bool {
	if device.Lldp != nil {
		if device.Lldp.Enabled {
			return true
		}
	} else if deviceclass.RunsLLDPByDefault(deviceclass.Parse(device.Type)) {
		return true
	}
	for _, block := range []*enabledBlock{device.Cdp, device.Edp, device.Fdp} {
		if block != nil && block.Enabled {
			return true
		}
	}
	return false
}

func assertDiscoverable(t *testing.T, source string, doc discoveryDoc) {
	t.Helper()

	byName := make(map[string]discoveryDevice, len(doc.Devices))
	for _, device := range doc.Devices {
		byName[device.Name] = device
	}

	for _, local := range doc.Devices {
		for _, trunk := range local.TrunkPorts {
			// An FDB-only entry is a device seen but deliberately not linked,
			// and a stub port names no peer at all.
			if trunk.FDBOnly || trunk.RemoteDevice == "" {
				continue
			}
			remote, ok := byName[trunk.RemoteDevice]
			if !ok {
				continue
			}
			if !speaksDiscovery(local) {
				t.Errorf("%s: %s declares a link to %s but announces itself over no discovery protocol",
					source, local.Name, trunk.RemoteDevice)
			}
			if !speaksDiscovery(remote) {
				t.Errorf("%s: %s declares a link to %s, which announces itself over no discovery protocol",
					source, local.Name, trunk.RemoteDevice)
			}
		}
	}
}

// TestSpeaksDiscovery pins the rule the gate above applies, so a green run
// there means the scenarios are clean rather than the check being vacuous.
func TestSpeaksDiscovery(t *testing.T) {
	tests := []struct {
		name   string
		device discoveryDevice
		want   bool
	}{
		{"a switch with no blocks runs LLDP", discoveryDevice{Type: "switch"}, true},
		{"a router with no blocks runs LLDP", discoveryDevice{Type: "router"}, true},
		{"a firewall with no blocks runs LLDP", discoveryDevice{Type: "firewall"}, true},
		{"a phone with no blocks runs LLDP", discoveryDevice{Type: "voip-phone"}, true},

		// The case the gate exists to catch: something declares a link and
		// announces itself over nothing.
		{"a workstation with no blocks is silent", discoveryDevice{Type: "workstation"}, false},
		{"a server with no blocks is silent", discoveryDevice{Type: "server"}, false},
		{"a printer with no blocks is silent", discoveryDevice{Type: "printer"}, false},
		{"an untyped device is silent", discoveryDevice{}, false},

		{
			"a switch with LLDP turned off is silent",
			discoveryDevice{Type: "switch", Lldp: &enabledBlock{Enabled: false}},
			false,
		},
		{
			"a server told to run LLDP speaks",
			discoveryDevice{Type: "server", Lldp: &enabledBlock{Enabled: true}},
			true,
		},
		{
			"a silent type speaks once CDP is on",
			discoveryDevice{Type: "workstation", Cdp: &enabledBlock{Enabled: true}},
			true,
		},
		{
			"a present but disabled CDP block is not consent",
			discoveryDevice{Type: "workstation", Cdp: &enabledBlock{Enabled: false}},
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := speaksDiscovery(tt.device); got != tt.want {
				t.Errorf("speaksDiscovery = %v, want %v", got, tt.want)
			}
		})
	}
}

// A link whose far end is silent is unverifiable by any analyzer, and the gate
// has to say so rather than pass quietly.
func TestGateCatchesAnUndiscoverableLink(t *testing.T) {
	doc := discoveryDoc{Devices: []discoveryDevice{
		{
			Name: "sw-01", Type: "switch",
			TrunkPorts: []discoveryPort{{RemoteDevice: "srv-01"}},
		},
		{Name: "srv-01", Type: "server"},
	}}

	fake := &testing.T{}
	assertDiscoverable(fake, "synthetic", doc)
	if !fake.Failed() {
		t.Error("a link to a device that announces itself over nothing was accepted")
	}
}
