package templates_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	tmpl "github.com/MustardSeedNetworks/niac-go/internal/templates"
)

// TestList verifies that all expected templates are listed.
func TestList(t *testing.T) {
	templates := tmpl.List()

	// The named templates below are the assertion. A bare count only tells
	// whoever adds the next template to bump the number.
	if len(templates) == 0 {
		t.Fatal("no templates found in the embedded tree")
	}

	// Verify each template has required fields
	for _, tpl := range templates {
		if tpl.Name == "" {
			t.Error("Template has empty name")
		}

		if tpl.Description == "" {
			t.Errorf("Template %s has empty description", tpl.Name)
		}

		if tpl.UseCase == "" {
			t.Errorf("Template %s has empty use case", tpl.Name)
		}
	}

	// Verify expected template names exist
	expectedNames := []string{
		"basic-network",
		"captured-switch",
		"captured-pair",
		"resource-pressure",
	}

	// The shipped set is a deliberate four: the rest of the shipped scenarios
	// are scenario packs, generated on demand. A fifth hand-written template
	// should be a decision, not drift.
	if len(templates) != len(expectedNames) {
		t.Errorf("embedded templates = %d, want exactly %v", len(templates), expectedNames)
	}

	for _, expected := range expectedNames {
		found := false

		for _, tpl := range templates {
			if tpl.Name == expected {
				found = true

				break
			}
		}

		if !found {
			t.Errorf("Expected template %s not found in list", expected)
		}
	}
}

// TestListNames verifies that ListNames returns correct template names.
func TestListNames(t *testing.T) {
	names := tmpl.ListNames()

	// Deliberately not a count: adding a template is routine, and a test that
	// fails on the number rather than on a property teaches people to bump the
	// number. List() reads the same embedded FS, so a name in one and not the
	// other is the real defect.
	if len(names) != len(tmpl.List()) {
		t.Errorf("ListNames returned %d names, List returned %d templates", len(names), len(tmpl.List()))
	}
	if len(names) == 0 {
		t.Fatal("no templates found in the embedded tree")
	}

	// Verify names are sorted
	for i := 1; i < len(names); i++ {
		if names[i-1] > names[i] {
			t.Errorf("Names are not sorted: %s comes after %s", names[i-1], names[i])
		}
	}
}

// TestGet verifies that templates can be loaded.
func TestGet(t *testing.T) {
	testCases := []string{
		"basic-network",
	}

	for _, name := range testCases {
		t.Run(name, func(t *testing.T) {
			template, err := tmpl.Get(name)
			if err != nil {
				t.Fatalf("Failed to get template %s: %v", name, err)
			}

			// Verify template fields
			if template.Name != name {
				t.Errorf("Expected name %s, got %s", name, template.Name)
			}

			if template.Description == "" {
				t.Error("Template has empty description")
			}

			if template.UseCase == "" {
				t.Error("Template has empty use case")
			}

			if template.Content == "" {
				t.Error("Template has empty content")
			}

			// Verify content is valid YAML with devices section
			if !strings.Contains(template.Content, "devices:") {
				t.Error("Template content does not contain 'devices:' section")
			}
		})
	}
}

// TestGetWithYamlExtension verifies that .yaml extension is handled.
func TestGetWithYamlExtension(t *testing.T) {
	// Should work with or without .yaml extension
	tmpl1, err1 := tmpl.Get("basic-network")
	tmpl2, err2 := tmpl.Get("basic-network.yaml")

	if err1 != nil || err2 != nil {
		t.Fatalf("Failed to get template: %v, %v", err1, err2)
	}

	if tmpl1.Content != tmpl2.Content {
		t.Error("Template content differs with/without .yaml extension")
	}
}

// TestGetNonExistent verifies error handling for missing templates.
func TestGetNonExistent(t *testing.T) {
	_, err := tmpl.Get("non-existent-template")
	if err == nil {
		t.Error("Expected error for non-existent template, got nil")
	}

	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("Expected 'not found' in error message, got: %v", err)
	}
}

// TestExists verifies template existence checking.
func TestExists(t *testing.T) {
	testCases := []struct {
		name   string
		exists bool
	}{
		{"basic-network", true},
		{"captured-switch", true},
		{"captured-pair", true},
		{"resource-pressure", true},
		{"small-office", false},
		{"non-existent", false},
		{"invalid-name", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			exists := tmpl.Exists(tc.name)
			if exists != tc.exists {
				t.Errorf("Exists(%s) = %v, want %v", tc.name, exists, tc.exists)
			}
		})
	}
}

// TestValidate verifies template content validation.
func TestValidate(t *testing.T) {
	testCases := []struct {
		name    string
		content string
		wantErr bool
	}{
		{
			name:    "valid template",
			content: "devices:\n  - name: test\n    type: router",
			wantErr: false,
		},
		{
			name:    "empty content",
			content: "",
			wantErr: true,
		},
		{
			name:    "missing devices section",
			content: "config:\n  name: test",
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tmpl.Validate(tc.content)
			if (err != nil) != tc.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}

// TestTemplateConfigValidity verifies all templates are valid YAML configs.
func TestTemplateConfigValidity(t *testing.T) {
	templates := tmpl.List()

	for _, tpl := range templates {
		t.Run(tpl.Name, func(t *testing.T) {
			cfg := loadTemplateAsConfig(t, tpl.Name)
			assertConfigHasDevices(t, cfg, tpl.Name)
			assertDevicesHaveRequiredFields(t, cfg.Devices, tpl.Name)
		})
	}
}

// loadTemplateAsConfig loads a template and parses it as a config.
func loadTemplateAsConfig(t *testing.T, templateName string) *config.Config {
	t.Helper()

	fullTmpl, err := tmpl.Get(templateName)
	if err != nil {
		t.Fatalf("Failed to get template: %v", err)
	}

	tmpFile := createTempConfigFile(t, fullTmpl.Content)

	cfg, err := config.Load(tmpFile)
	if err != nil {
		t.Fatalf("Template %s failed to load as valid config: %v", templateName, err)
	}

	return cfg
}

// createTempConfigFile writes content into the layout a template is installed
// into -- networks/ beside walks/ -- and returns the path.
//
// The layout is the point, not scaffolding: a template that replays a capture
// names it relative to its own file with `include_path: ../walks`, so a
// template written to a bare temp directory cannot resolve its walk and looks
// broken while working perfectly in the product.
func createTempConfigFile(t *testing.T, content string) string {
	t.Helper()

	root := t.TempDir()
	networks := filepath.Join(root, "networks")
	walks := filepath.Join(root, "walks")
	for _, dir := range []string{networks, walks} {
		if err := os.MkdirAll(dir, 0o750); err != nil {
			t.Fatalf("create %s: %v", dir, err)
		}
	}

	starter := filepath.Join("..", "library", "starter", "walks")
	entries, err := os.ReadDir(starter)
	if err != nil {
		t.Fatalf("read starter walks: %v", err)
	}
	for _, entry := range entries {
		capture, readErr := os.ReadFile(filepath.Join(starter, entry.Name()))
		if readErr != nil {
			t.Fatalf("read %s: %v", entry.Name(), readErr)
		}
		if writeErr := os.WriteFile(filepath.Join(walks, entry.Name()), capture, 0o600); writeErr != nil {
			t.Fatalf("write %s: %v", entry.Name(), writeErr)
		}
	}

	path := filepath.Join(networks, "niac-test.yaml")
	if writeErr := os.WriteFile(path, []byte(content), 0o600); writeErr != nil {
		t.Fatalf("Failed to write temp file: %v", writeErr)
	}

	return path
}

// assertConfigHasDevices verifies a config has at least one device.
func assertConfigHasDevices(t *testing.T, cfg *config.Config, templateName string) {
	t.Helper()

	if len(cfg.Devices) == 0 {
		t.Errorf("Template %s has no devices", templateName)
	}
}

// assertDevicesHaveRequiredFields verifies all devices have required fields.
func assertDevicesHaveRequiredFields(t *testing.T, devices []config.Device, templateName string) {
	t.Helper()

	for i, device := range devices {
		assertDeviceValid(t, &device, templateName, i)
	}
}

// assertDeviceValid verifies a single device has required fields.
func assertDeviceValid(t *testing.T, device *config.Device, templateName string, index int) {
	t.Helper()

	if device.Name == "" {
		t.Errorf("Template %s device %d has empty name", templateName, index)
	}

	if device.Type == "" {
		t.Errorf("Template %s device %s has empty type", templateName, device.Name)
	}

	if len(device.MACAddress) == 0 {
		t.Errorf("Template %s device %s has empty MAC address", templateName, device.Name)
	}
}

// TestBasicNetworkTemplate verifies basic-network template specifics.
func TestBasicNetworkTemplate(t *testing.T) {
	template, err := tmpl.Get("basic-network")
	if err != nil {
		t.Fatalf("Failed to get basic-network template: %v", err)
	}

	// Create temp file and load config
	tmpFile, err := os.CreateTemp(t.TempDir(), "niac-test-*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	defer func() { _ = os.Remove(tmpFile.Name()) }()
	defer func() { _ = tmpFile.Close() }()

	_, _ = tmpFile.WriteString(template.Content)
	_ = tmpFile.Close()

	cfg, err := config.Load(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Basic network should have 2 devices (router + switch)
	if len(cfg.Devices) != 2 {
		t.Errorf("Expected 2 devices, got %d", len(cfg.Devices))
	}

	// Verify both devices have basic protocols enabled
	lldpCount := 0

	for _, device := range cfg.Devices {
		if device.LLDPConfig != nil && device.LLDPConfig.Enabled {
			lldpCount++
		}
	}

	if lldpCount < 2 {
		t.Errorf("Expected at least 2 devices with LLDP, got %d", lldpCount)
	}
}

// BenchmarkGet benchmarks template loading performance.
func BenchmarkGet(b *testing.B) {
	for b.Loop() {
		_, err := tmpl.Get("basic-network")
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkList benchmarks template listing performance.
func BenchmarkList(b *testing.B) {
	for b.Loop() {
		_ = tmpl.List()
	}
}
