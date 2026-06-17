package templates

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestSplitTags(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		raw  string
		want []string
	}{
		{"empty", "", nil},
		{"single", "router", []string{"router"}},
		{"multiple", "router,switch,snmp", []string{"router", "switch", "snmp"}},
		{"trims whitespace", " router , switch ", []string{"router", "switch"}},
		{"skips empties", "router,,switch,", []string{"router", "switch"}},
		{"only commas", ",,,", nil},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := splitTags(tc.raw)
			if !slices.Equal(got, tc.want) {
				t.Errorf("splitTags(%q) = %v, want %v", tc.raw, got, tc.want)
			}
		})
	}
}

func TestDetermineTemplateType(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		path     string
		fileName string
		want     string
	}{
		{"router by name", "/x/edge-router.yaml", "edge-router", typeRouter},
		{"switch by name", "/x/core-switch.yaml", "core-switch", typeSwitch},
		{"ap by name", "/x/office-ap.yaml", "office-ap", typeAccessPoint},
		{"wireless by name", "/x/wireless.yaml", "wireless", typeAccessPoint},
		{"firewall by name", "/x/asa.yaml", "asa", typeFirewall},
		{"server by name", "/x/server.yaml", "server", typeServer},
		{"complete by name", "/x/full.yaml", "full", typeComplete},
		{"services by path", "/templates/services/dns.yaml", "dns", typeServer},
		{"combinations by path", "/templates/combinations/x.yaml", "x", typeComplete},
		{"vendors by path", "/templates/vendors/x.yaml", "x", typeCustom},
		{"default basic", "/templates/misc/thing.yaml", "thing", typeBasic},
		{"name beats path", "/templates/services/edge-router.yaml", "edge-router", typeRouter},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := determineTemplateType(tc.path, tc.fileName); got != tc.want {
				t.Errorf("determineTemplateType(%q,%q) = %q, want %q", tc.path, tc.fileName, got, tc.want)
			}
		})
	}
}

func TestGenerateTags(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		path string
		want []string // order-independent membership check
	}{
		{"no matches", "/templates/plain/thing.yaml", nil},
		{"router path", "/templates/router/x.yaml", []string{"router"}},
		{"firewall maps to security", "/templates/firewall/x.yaml", []string{"security"}},
		{"vendor cisco", "/templates/cisco/x.yaml", []string{"cisco"}},
		{"path + vendor", "/templates/router/cisco-edge.yaml", []string{"router", "cisco"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := generateTags(tc.path)
			for _, want := range tc.want {
				if !slices.Contains(got, want) {
					t.Errorf("generateTags(%q) = %v, missing %q", tc.path, got, want)
				}
			}
			if len(tc.want) == 0 && len(got) != 0 {
				t.Errorf("generateTags(%q) = %v, want empty", tc.path, got)
			}
		})
	}
}

func TestSanitizeConfigName(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"clean", "my-config_1", "my-config_1"},
		{"spaces", "my config", "my-config"},
		{"slashes blocked", "../etc/passwd", "---etc-passwd"},
		{"special chars", "a@b#c", "a-b-c"},
		{"alnum preserved", "ABC123", "ABC123"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := SanitizeConfigName(tc.in); got != tc.want {
				t.Errorf("SanitizeConfigName(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestExtractFrontMatter(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "t.yaml")
	content := "# Display: Edge Router\n" +
		"# Description: A full router persona\n" +
		"# Vendor: Cisco\n" +
		"# Tags: router, snmp\n" +
		"devices:\n" +
		"# this comment is below the body and ignored\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	meta := extractFrontMatter(path)
	want := map[string]string{
		"display":     "Edge Router",
		"description": "A full router persona",
		"vendor":      "Cisco",
		"tags":        "router, snmp",
	}
	for k, v := range want {
		if meta[k] != v {
			t.Errorf("meta[%q] = %q, want %q", k, meta[k], v)
		}
	}
	if len(meta) != len(want) {
		t.Errorf("meta has %d keys, want %d: %v", len(meta), len(want), meta)
	}
}

func TestExtractFrontMatter_MissingFile(t *testing.T) {
	t.Parallel()
	if meta := extractFrontMatter(filepath.Join(t.TempDir(), "nope.yaml")); meta != nil {
		t.Errorf("expected nil for missing file, got %v", meta)
	}
}

func TestCountDevicesInFile(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		content string
		want    int
	}{
		{"two devices", "devices:\n  - name: r1\n  - name: r2\n", 2},
		{"no devices defaults to 1", "metadata:\n  foo: bar\n", 1},
		{"one device", "devices:\n  - name: only\n", 1},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			path := filepath.Join(t.TempDir(), "c.yaml")
			if err := os.WriteFile(path, []byte(tc.content), 0o600); err != nil {
				t.Fatal(err)
			}
			if got := CountDevicesInFile(path); got != tc.want {
				t.Errorf("CountDevicesInFile = %d, want %d", got, tc.want)
			}
		})
	}
}

func TestCountDevicesInFile_MissingFile(t *testing.T) {
	t.Parallel()
	if got := CountDevicesInFile(filepath.Join(t.TempDir(), "nope.yaml")); got != 1 {
		t.Errorf("missing file = %d, want 1", got)
	}
}

func TestExtractDescription(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{"first comment", "# A nice router\ndevices:\n", "A nice router"},
		{"skips yaml header", "# yaml-language-server: foo\n# Real desc\ndevices:\n", "Real desc"},
		{"stops at body", "devices:\n# too late\n", ""},
		{"no comment", "devices:\n", ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			path := filepath.Join(t.TempDir(), "d.yaml")
			if err := os.WriteFile(path, []byte(tc.content), 0o600); err != nil {
				t.Fatal(err)
			}
			if got := extractDescription(path); got != tc.want {
				t.Errorf("extractDescription = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestParseTemplateFile_FrontMatterWins(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "edge-router.yaml")
	content := "# Display: Enterprise Router\n" +
		"# Description: Full enterprise router\n" +
		"# Type: firewall\n" +
		"# Vendor: Juniper\n" +
		"# Tags: custom-tag\n" +
		"devices:\n  - name: r1\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	tmpl := parseTemplateFile(path)
	if tmpl.Name != "edge-router" {
		t.Errorf("Name = %q, want edge-router", tmpl.Name)
	}
	if tmpl.DisplayName != "Enterprise Router" {
		t.Errorf("DisplayName = %q", tmpl.DisplayName)
	}
	if tmpl.Description != "Full enterprise router" {
		t.Errorf("Description = %q", tmpl.Description)
	}
	if tmpl.Type != "firewall" { // front-matter type wins over name "router"
		t.Errorf("Type = %q, want firewall", tmpl.Type)
	}
	if tmpl.Vendor != "juniper" { // lowercased
		t.Errorf("Vendor = %q, want juniper", tmpl.Vendor)
	}
	if !slices.Equal(tmpl.Tags, []string{"custom-tag"}) {
		t.Errorf("Tags = %v, want [custom-tag]", tmpl.Tags)
	}
	if tmpl.DeviceCount != 1 {
		t.Errorf("DeviceCount = %d, want 1", tmpl.DeviceCount)
	}
}

func TestParseTemplateFile_InfersFromNameWhenNoFrontMatter(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "core-switch.yaml")
	if err := os.WriteFile(path, []byte("devices:\n  - name: s1\n  - name: s2\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	tmpl := parseTemplateFile(path)
	if tmpl.Type != typeSwitch {
		t.Errorf("Type = %q, want %q", tmpl.Type, typeSwitch)
	}
	if tmpl.Description != "core-switch configuration template" {
		t.Errorf("Description = %q (expected synthesized fallback)", tmpl.Description)
	}
	if tmpl.DeviceCount != 2 {
		t.Errorf("DeviceCount = %d, want 2", tmpl.DeviceCount)
	}
}

func TestScan(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "router.yaml"), "# A router\ndevices:\n  - name: r1\n")
	mustWrite(t, filepath.Join(dir, "switch.yml"), "# A switch\ndevices:\n  - name: s1\n")
	mustWrite(t, filepath.Join(dir, "notes.txt"), "ignored")
	// nested dir is walked recursively
	sub := filepath.Join(dir, "vendors")
	if err := os.Mkdir(sub, 0o750); err != nil {
		t.Fatal(err)
	}
	mustWrite(t, filepath.Join(sub, "cisco.yaml"), "# Cisco\ndevices:\n  - name: c1\n")

	got, err := Scan(dir)
	if err != nil {
		t.Fatalf("Scan error: %v", err)
	}
	if len(got) != 3 { // .txt skipped, 3 yaml found
		t.Fatalf("Scan found %d templates, want 3: %v", len(got), got)
	}
	names := make([]string, len(got))
	for i, tmpl := range got {
		names[i] = tmpl.Name
	}
	for _, want := range []string{"router", "switch", "cisco"} {
		if !slices.Contains(names, want) {
			t.Errorf("Scan missing %q in %v", want, names)
		}
	}
}

func TestScan_MissingDirIsNotFatal(t *testing.T) {
	t.Parallel()
	got, _ := Scan(filepath.Join(t.TempDir(), "does-not-exist"))
	if len(got) != 0 {
		t.Errorf("Scan of missing dir = %v, want empty", got)
	}
}

func TestFindAndLoad(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "myrouter.yaml"), "# router\ndevices:\n  - name: r1\n")
	t.Setenv("NIAC_TEMPLATES_DIR", dir)

	// case-insensitive match
	found := Find("MyRouter")
	if found == "" {
		t.Fatal("Find returned empty for existing template")
	}
	if filepath.Base(found) != "myrouter.yaml" {
		t.Errorf("Find = %q, want myrouter.yaml", found)
	}

	if Find("nonexistent") != "" {
		t.Error("Find returned non-empty for missing template")
	}

	content, path, err := Load("myrouter")
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}
	if !strings.Contains(string(content), "devices:") {
		t.Errorf("Load content unexpected: %q", content)
	}
	if path != found {
		t.Errorf("Load path = %q, want %q", path, found)
	}

	if _, _, loadErr := Load("nope"); loadErr == nil {
		t.Error("Load of missing template should error")
	}
}

func TestSaveConfig(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir) // SaveConfig writes under ./configs

	content := []byte("devices:\n  - name: r1\n")

	// derives name from templateName when newConfigName empty
	path, err := SaveConfig("router", "", content)
	if err != nil {
		t.Fatalf("SaveConfig error: %v", err)
	}
	if filepath.Base(path) != "router-config.yaml" {
		t.Errorf("derived name = %q, want router-config.yaml", filepath.Base(path))
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading written config: %v", err)
	}
	if string(got) != string(content) {
		t.Errorf("written content = %q, want %q", got, content)
	}

	// explicit name is sanitized
	path2, err := SaveConfig("router", "my custom/name", content)
	if err != nil {
		t.Fatalf("SaveConfig error: %v", err)
	}
	if filepath.Base(path2) != "my-custom-name.yaml" {
		t.Errorf("sanitized name = %q, want my-custom-name.yaml", filepath.Base(path2))
	}
}

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}
