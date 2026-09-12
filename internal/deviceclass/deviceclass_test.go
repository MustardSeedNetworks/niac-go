package deviceclass_test

import (
	"os"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/deviceclass"
)

// TestVocabularyMatchesSchema is the whole point of this package.
//
// Twelve files used to hand-roll their own list of device types, and each
// drifted from the schema in its own way: eight templates shipped untyped and
// announced their core routers as end stations (#2096), `firewall` had no case
// at all and announced itself as a host, and `voip-phone` — the schema's own
// spelling — fell through a switch that listed `voip_phone`, so a hand-authored
// phone silently stopped advertising LLDP.
//
// The schema's `oneof` is the source of truth. This reads it rather than
// restating it, so a type added there fails here until this package knows it.
func TestVocabularyMatchesSchema(t *testing.T) {
	source, err := os.ReadFile("../converter/types.go")
	if err != nil {
		t.Fatalf("read converter types: %v", err)
	}

	pattern := regexp.MustCompile(`Type string \x60yaml:"type,omitempty" validate:"omitempty,oneof=([^"]+)"`)
	match := pattern.FindSubmatch(source)
	if match == nil {
		t.Fatal("could not find the device type oneof in converter/types.go")
	}
	schema := strings.Fields(string(match[1]))

	known := deviceclass.All()
	if len(known) != len(schema) {
		t.Errorf("deviceclass knows %d types, schema declares %d:\n  deviceclass: %v\n  schema:      %v",
			len(known), len(schema), known, schema)
	}

	for _, want := range schema {
		if !deviceclass.IsKnown(deviceclass.Type(want)) {
			t.Errorf("schema declares %q and deviceclass does not know it", want)
		}
	}
	for _, have := range known {
		if !slices.Contains(schema, string(have)) {
			t.Errorf("deviceclass knows %q and the schema does not declare it", have)
		}
	}
}

// A forwarding device is one that moves other devices' traffic. The
// distinction drives what it announces on the wire, and getting it wrong is
// what made a firewall look like a host to every discovery tool.
func TestForwarding(t *testing.T) {
	forwarding := map[deviceclass.Type]bool{
		deviceclass.Router: true, deviceclass.Switch: true,
		deviceclass.Layer3Switch: true, deviceclass.AP: true,
		deviceclass.AccessPoint: true, deviceclass.Firewall: true,
		deviceclass.Server: false, deviceclass.Host: false,
		deviceclass.Workstation: false, deviceclass.IoT: false,
		deviceclass.Printer: false, deviceclass.VoipPhone: false,
		deviceclass.Unknown: false,
	}
	for typ, want := range forwarding {
		if got := deviceclass.Forwards(typ); got != want {
			t.Errorf("Forwards(%q) = %v, want %v", typ, got, want)
		}
	}
}

// Real hardware in these roles runs LLDP out of the box; an end station does
// not unless someone turns it on. A VoIP phone is in the first group — that is
// how it learns its voice VLAN — and it was in the second by accident.
func TestRunsLLDPByDefault(t *testing.T) {
	for _, typ := range []deviceclass.Type{
		deviceclass.Router, deviceclass.Switch, deviceclass.Layer3Switch,
		deviceclass.AP, deviceclass.AccessPoint, deviceclass.Firewall,
		deviceclass.VoipPhone,
	} {
		if !deviceclass.RunsLLDPByDefault(typ) {
			t.Errorf("RunsLLDPByDefault(%q) = false, want true", typ)
		}
	}
	for _, typ := range []deviceclass.Type{
		deviceclass.Server, deviceclass.Host, deviceclass.Workstation,
		deviceclass.IoT, deviceclass.Printer,
	} {
		if deviceclass.RunsLLDPByDefault(typ) {
			t.Errorf("RunsLLDPByDefault(%q) = true, want false", typ)
		}
	}
}

// Spellings the runtime has always tolerated, and the absent case. Parse is
// the one place that decides what an authored string means.
func TestParse(t *testing.T) {
	for raw, want := range map[string]deviceclass.Type{
		"router": deviceclass.Router, "ROUTER": deviceclass.Router,
		"access_point": deviceclass.AccessPoint, "access-point": deviceclass.AccessPoint,
		"wireless-ap": deviceclass.AccessPoint, "wireless_ap": deviceclass.AccessPoint,
		"ap":         deviceclass.AP,
		"voip_phone": deviceclass.VoipPhone, "voip-phone": deviceclass.VoipPhone,
		"phone": deviceclass.VoipPhone,
		"":      deviceclass.Unknown, "toaster": deviceclass.Unknown,
	} {
		if got := deviceclass.Parse(raw); got != want {
			t.Errorf("Parse(%q) = %q, want %q", raw, got, want)
		}
	}
}
