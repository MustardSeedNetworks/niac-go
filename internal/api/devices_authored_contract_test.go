package api

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
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

// The editor's whole loop, through the HTTP handlers: open a device, save it
// untouched, open it again. That has to be an identity — an operator who fixes
// one typo must not lose an unrelated block — and it is the loop that was
// broken twice over. The projection the editor used to POST carried a
// `hostname` DeviceUpdateRequest does not declare, so with the strict decoder
// every update of an existing device answered 400; and had it been accepted, it
// would have written back only the 56 fields the projection has properties for.
func TestDeviceEditorLoopPreservesTheDocument(t *testing.T) {
	for _, name := range authoredFixtureNames(t) {
		t.Run(name, func(t *testing.T) {
			// A device authoring `ssh.password_env` cannot be saved unless that
			// variable is set: the daemon refuses to persist a config it could
			// not run (ValidateDeviceManagementRequirements). The clinic server
			// authors one, so the loop needs it present to reach the save.
			t.Setenv("NIAC_SSH_PASSWORD", "contract-fixture")
			server := newAuthoredDeviceServer(t, name)

			opened := getDeviceRawYAML(t, server, name)
			if opened == "" {
				t.Fatal("GET returned no rawYaml, so the editor has nothing to load")
			}

			body, err := json.Marshal(DeviceUpdateRequest{RawYAML: opened})
			if err != nil {
				t.Fatalf("marshal update: %v", err)
			}
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPut, "/api/v1/config/devices/"+name, bytes.NewReader(body))
			server.handleDevicesV2(rec, req)
			if rec.Code != http.StatusOK {
				t.Fatalf("save answered %d: %s", rec.Code, rec.Body.String())
			}

			if reopened := getDeviceRawYAML(t, server, name); reopened != opened {
				t.Errorf("saving an untouched device changed it\nopened:\n%s\nreopened:\n%s", opened, reopened)
			}
		})
	}
}

// The create half of the same loop. It was covered only by an E2E that mocked
// a 201, which is exactly how the update 400 stayed invisible: the daemon has
// to accept a body of `{hostname, rawYaml}` with none of the scalar fields
// `buildDeviceFromRequest` applies before it reaches the document.
func TestDeviceEditorLoopCreatesFromDocument(t *testing.T) {
	// One MAC identity, one vendor identity — the two halves of the choice the
	// daemon rejects both of.
	for _, name := range []string{"exam-pc", "clinic-rtr-01"} {
		t.Run(name, func(t *testing.T) {
			t.Setenv("NIAC_SSH_PASSWORD", "contract-fixture")
			authored := readAuthoredFixture(t, name)
			server := newEmptyDeviceServer(t)

			body, err := json.Marshal(DeviceCreateRequest{Hostname: name, RawYAML: authored})
			if err != nil {
				t.Fatalf("marshal create: %v", err)
			}
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/api/v1/config/devices", bytes.NewReader(body))
			server.handleDevicesV2(rec, req)
			if rec.Code != http.StatusCreated {
				t.Fatalf("create answered %d: %s", rec.Code, rec.Body.String())
			}

			want, err := parseDeviceFromYAML(authored, name)
			if err != nil {
				t.Fatalf("parse fixture: %v", err)
			}
			wantYAML, err := config.MarshalDeviceYAML(want)
			if err != nil {
				t.Fatalf("serialize expectation: %v", err)
			}
			if got := getDeviceRawYAML(t, server, name); got != string(wantYAML) {
				t.Errorf("the created device is not the authored one\nwant:\n%s\ngot:\n%s", wantYAML, got)
			}
		})
	}
}

// One edit reaches the device and nothing else moves.
func TestDeviceEditorLoopAppliesOneEdit(t *testing.T) {
	const name = "clinic-rtr-01"
	server := newAuthoredDeviceServer(t, name)

	edited := strings.Replace(
		getDeviceRawYAML(t, server, name),
		"syslocation: Clinic branch, comms closet",
		"syslocation: Clinic branch, server room",
		1,
	)
	body, err := json.Marshal(DeviceUpdateRequest{RawYAML: edited})
	if err != nil {
		t.Fatalf("marshal update: %v", err)
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/config/devices/"+name, bytes.NewReader(body))
	server.handleDevicesV2(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("save answered %d: %s", rec.Code, rec.Body.String())
	}

	reopened := getDeviceRawYAML(t, server, name)
	if !strings.Contains(reopened, "syslocation: Clinic branch, server room") {
		t.Errorf("the edit did not land:\n%s", reopened)
	}
	// The DHCP block is untouched by the edit and is one of the 167 fields the
	// old camelCase projection had no property for.
	if !strings.Contains(reopened, "pool_start: 10.20.0.100") {
		t.Errorf("an unrelated block was lost by the save:\n%s", reopened)
	}
}

func newAuthoredDeviceServer(t *testing.T, name string) *Server {
	t.Helper()

	// Loaded the same way the daemon loads it, so the server holds the device
	// an author's file produces rather than one a test built by hand.
	device, err := parseDeviceFromYAML(readAuthoredFixture(t, name), name)
	if err != nil {
		t.Fatalf("load fixture: %v", err)
	}
	cfg := &config.Config{Devices: []config.Device{*device}}

	configPath := filepath.Join(t.TempDir(), "config.yaml")
	configYAML, err := config.MarshalConfigYAML(cfg)
	if err != nil {
		t.Fatalf("marshal config: %v", err)
	}
	if writeErr := os.WriteFile(configPath, configYAML, 0o600); writeErr != nil {
		t.Fatalf("write config: %v", writeErr)
	}

	return &Server{
		cfg:    ServerConfig{Config: cfg, ConfigPath: configPath, Version: "test"},
		logger: slog.Default(),
	}
}

func newEmptyDeviceServer(t *testing.T) *Server {
	t.Helper()

	cfg := &config.Config{}
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	configYAML, err := config.MarshalConfigYAML(cfg)
	if err != nil {
		t.Fatalf("marshal config: %v", err)
	}
	if writeErr := os.WriteFile(configPath, configYAML, 0o600); writeErr != nil {
		t.Fatalf("write config: %v", writeErr)
	}

	return &Server{
		cfg:    ServerConfig{Config: cfg, ConfigPath: configPath, Version: "test"},
		logger: slog.Default(),
	}
}

func getDeviceRawYAML(t *testing.T, server *Server, name string) string {
	t.Helper()

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/config/devices/"+name, nil)
	server.handleDevicesV2(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET answered %d: %s", rec.Code, rec.Body.String())
	}

	var resp DeviceResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode device: %v", err)
	}

	return resp.RawYAML
}
