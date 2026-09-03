package api

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
)

// The daemon half of the authored-device contract.
//
// testdata/authored_devices/*.yaml is written by the UI
// (ui/src/utils/authored-device-contract.test.ts) as exactly what the device
// editor POSTs in `rawYaml`, and is parsed here through the real save path.
// The editor's document model is the authored YAML itself, so its round trip
// is an identity only while both YAML implementations agree about the
// document; one set of files, asserted from both sides, is what makes that
// checkable rather than asserted.
const authoredFixtureDir = "testdata/authored_devices"

// The property that makes the editor's round trip safe: what the GET read-back
// serializes must load to the same device the author's own document loaded to.
// A block the read-back cannot express is a block the editor loses on save —
// `mdns` was exactly that until P1b-2, parsed by the loader and absent from
// deviceToYAML, so an authored mDNS advertisement disappeared the first time
// an operator touched any other field.
//
// Comparing loaded devices rather than documents keeps the assertion about
// meaning: `omitempty` dropping an authored `babble: false`, or a MAC coming
// back lower-cased, are the same device and must not fail.
func TestAuthoredDeviceReadBackIsLossless(t *testing.T) {
	for _, name := range authoredFixtureNames(t) {
		t.Run(name, func(t *testing.T) {
			authored := readAuthoredFixture(t, name)

			first, err := parseDeviceFromYAML(authored, name)
			if err != nil {
				t.Fatalf("the editor's YAML does not parse through the save path: %v", err)
			}

			readBack, err := config.MarshalDeviceYAML(first)
			if err != nil {
				t.Fatalf("serialize read-back: %v", err)
			}
			second, err := parseDeviceFromYAML(string(readBack), name)
			if err != nil {
				t.Fatalf("the read-back does not parse: %v\n%s", err, readBack)
			}

			if !reflect.DeepEqual(first, second) {
				t.Errorf(
					"load → serialize → load is not an identity; "+
						"the read-back loses or changes authoring\nauthored:\n%s\nread-back:\n%s",
					authored, readBack,
				)
			}
		})
	}
}

// A key the authored Device does not declare must be rejected, not dropped.
// The save path decoded leniently until P1b-2: a misnamed field parsed clean,
// the re-encode carried only what had decoded, and the strict loader never saw
// the key — so the editor silently discarded exactly what it was asked to save.
func TestAuthoredDeviceRejectsUnknownKey(t *testing.T) {
	for _, doc := range []string{
		"name: probe\ntype: switch\nmac: 00:11:22:33:44:55\nnot_a_field: 7\n",
		"name: probe\ntype: switch\nmac: 00:11:22:33:44:55\nsnmp_agent:\n  sysnaem: typo\n",
	} {
		if _, err := parseDeviceFromYAML(doc, "probe"); err == nil {
			t.Errorf("unknown key accepted, so the editor would drop it silently:\n%s", doc)
		}
	}
}

// Scalars whose YAML spelling two implementations need not agree on. The UI
// writes these unquoted; a digits-only MAC is base-60 under YAML 1.1, and
// `on` is a boolean there. Asserted as values, on the real fixture, so a
// future bump of either YAML library fails here instead of in the field.
func TestAuthoredDeviceScalarShapes(t *testing.T) {
	device, err := parseDeviceFromYAML(readAuthoredFixture(t, "shape-probe"), "shape-probe")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	if got, want := device.MACAddress.String(), "00:11:22:33:44:55"; got != want {
		t.Errorf("mac = %q, want %q", got, want)
	}
	if got := len(device.IPAddresses); got != 2 {
		t.Fatalf("ips = %d, want 2", got)
	}
	if got, want := device.IPAddresses[1].String(), "2001:db8::7"; got != want {
		t.Errorf("ips[1] = %q, want %q", got, want)
	}
	for key, want := range map[string]string{
		"sexagesimal":  "00:11:22",
		"booleanish":   "on",
		"leading_zero": "0700",
		"version":      "1.10",
		"multiline":    "line one\nline two",
		"quoted":       `has "quotes" and a: colon`,
	} {
		if got := device.Properties[key]; got != want {
			t.Errorf("properties[%s] = %q, want %q", key, got, want)
		}
	}
}

// The derived properties the loader writes from other authored fields must not
// come back as authored ones — the editor shows every property in its
// free-form map and saves what it was shown.
func TestAuthoredDeviceOmitsDerivedProperties(t *testing.T) {
	doc := "name: probe\ntype: switch\nmac: 00:11:22:33:44:55\nvlan: 10\nproperties:\n  site: clinic\n"

	device, err := parseDeviceFromYAML(doc, "probe")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	readBack, err := config.MarshalDeviceYAML(device)
	if err != nil {
		t.Fatalf("serialize read-back: %v", err)
	}

	var out struct {
		Properties map[string]string `yaml:"properties"`
	}
	if unmarshalErr := yaml.Unmarshal(readBack, &out); unmarshalErr != nil {
		t.Fatalf("unmarshal read-back: %v", unmarshalErr)
	}
	if !reflect.DeepEqual(out.Properties, map[string]string{"site": "clinic"}) {
		t.Errorf("properties = %v, want only the authored one", out.Properties)
	}
}

func authoredFixtureNames(t *testing.T) []string {
	t.Helper()

	entries, err := os.ReadDir(authoredFixtureDir)
	if err != nil {
		t.Fatalf("read fixture dir: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("no authored-device fixtures: the UI half never wrote them")
	}

	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		names = append(names, strings.TrimSuffix(entry.Name(), ".yaml"))
	}

	return names
}

func readAuthoredFixture(t *testing.T, name string) string {
	t.Helper()

	data, err := os.ReadFile(filepath.Join(authoredFixtureDir, name+".yaml"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	return string(data)
}
