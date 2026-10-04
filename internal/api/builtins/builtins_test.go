package builtins

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
)

func TestShippedScenariosAreValid(t *testing.T) {
	scenarioRoot, err := filepath.Abs(filepath.Join("..", "..", "..", "cmd", "niac", "templates"))
	if err != nil {
		t.Fatalf("resolve shipped scenario path: %v", err)
	}
	t.Setenv("NIAC_TEMPLATES_DIR", scenarioRoot)

	validated, err := validateShippedScenarios(t, scenarioRoot)
	if err != nil {
		t.Fatalf("walk shipped scenarios: %v", err)
	}
	if validated == 0 {
		t.Fatal("no shipped scenarios found")
	}
}

func validateShippedScenarios(t *testing.T, scenarioRoot string) (int, error) {
	t.Helper()
	validated := 0
	err := filepath.WalkDir(scenarioRoot, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		extension := strings.ToLower(filepath.Ext(path))
		if entry.IsDir() || (extension != ".yaml" && extension != ".yml") {
			return nil
		}
		validated++
		name := strings.TrimSuffix(filepath.Base(path), extension)
		t.Run(name, func(t *testing.T) { validateShippedScenario(t, name, path) })
		return nil
	})
	return validated, err
}

func validateShippedScenario(t *testing.T, name, scenarioPath string) {
	t.Helper()
	if found := Find(name); found != scenarioPath {
		t.Fatalf("Find(%q) = %q, want %q", name, found, scenarioPath)
	}
	content, err := os.ReadFile(filepath.Clean(scenarioPath))
	if err != nil {
		t.Fatalf("read shipped scenario: %v", err)
	}
	cfg, err := config.LoadYAMLBytes(content)
	if err != nil {
		t.Fatalf("parse selected scenario: %v", err)
	}
	result := config.NewValidator(scenarioPath).Validate(cfg)
	if result.HasErrors() {
		t.Fatal(result.Format())
	}
	assertShippedScenarioBehavior(t, scenarioPath, cfg)
}

func assertShippedScenarioBehavior(t *testing.T, scenarioPath string, cfg *config.Config) {
	t.Helper()
	expectedOID := expectedVendorObjectID(filepath.Base(scenarioPath))
	if strings.Contains(scenarioPath, "vendor-templates") && expectedOID == "" {
		t.Fatalf("vendor scenario %q has no expected sysObjectID", scenarioPath)
	}
	for _, device := range cfg.Devices {
		if device.LLDPConfig != nil && device.LLDPConfig.PortDescription != "" {
			if len(device.Interfaces) == 0 || device.Interfaces[0].Name != device.LLDPConfig.PortDescription {
				t.Errorf("device %q does not preserve its LLDP interface identity", device.Name)
			}
		}
		if expectedOID != "" && device.Properties["sysObjectID"] != expectedOID {
			t.Errorf("vendor device %q sysObjectID = %q, want %q",
				device.Name, device.Properties["sysObjectID"], expectedOID)
		}
	}
}

func expectedVendorObjectID(name string) string {
	return map[string]string{
		"cx-6300m-48g.yaml":      "1.3.6.1.4.1.47196.4.1.1.3.123",
		"iap-505.yaml":           "1.3.6.1.4.1.14823.1.2.96",
		"aironet-iw9165e.yaml":   "1.3.6.1.4.1.9.1.3120",
		"asa-5525-x.yaml":        "1.3.6.1.4.1.9.1.1408",
		"catalyst-9300-48p.yaml": "1.3.6.1.4.1.9.1.2697",
		"isr-4451-x.yaml":        "1.3.6.1.4.1.9.1.1903",
		"nexus-9336c-fx2.yaml":   "1.3.6.1.4.1.9.12.3.1.3.1411",
		"x440-g2-48t.yaml":       "1.3.6.1.4.1.1916.2.140",
		"x670-g2-48x.yaml":       "1.3.6.1.4.1.1916.2.139",
		"procurve-2530-48.yaml":  "1.3.6.1.4.1.11.2.3.7.11.150",
		"ex4300-48t.yaml":        "1.3.6.1.4.1.2636.1.1.1.2.71",
		"mist-ap43.yaml":         "1.3.6.1.4.1.55538",
		"mx204.yaml":             "1.3.6.1.4.1.2636.1.1.1.2.57",
		"srx340.yaml":            "1.3.6.1.4.1.2636.1.1.1.2.92",
	}[name]
}

func TestFind(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "myrouter.yaml"), "# router\ndevices:\n  - name: r1\n")
	t.Setenv("NIAC_TEMPLATES_DIR", dir)

	// case-insensitive match
	found := Find("MyRouter")
	if found == "" {
		t.Fatal("Find returned empty for existing scenario")
	}
	if filepath.Base(found) != "myrouter.yaml" {
		t.Errorf("Find = %q, want myrouter.yaml", found)
	}

	if Find("nonexistent") != "" {
		t.Error("Find returned non-empty for missing scenario")
	}
}

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}
