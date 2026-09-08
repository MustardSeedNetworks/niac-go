package api

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/gosnmp/gosnmp"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/library"
	"github.com/MustardSeedNetworks/niac-go/internal/protocols/snmp"
)

// F8: one authored device, three authoring surfaces, one replay.
//
// The daemon accepts the same device three ways, and each resolves the
// document's relative paths -- snmp_agent.walk_file above all -- against its
// own base directory:
//
//	file    a config file on disk          config.LoadYAML, base = the config's
//	        resolved include_path, else the file's own directory
//	editor  rawYaml on the device routes   parseDeviceFromYAML (this package)
//	wizard  configData on POST /simulation config.LoadYAMLBytesManaged, base =
//	        inlineConfigDir() (internal/daemon/daemon.go, loadSimulationConfig)
//
// The editor's input is deliberately not the authored text. The UI posts back
// the document the daemon serialized for it (DeviceDetailResponse.rawYaml, see
// ui/src/api/client.ts), which carries the *resolved* walk path, so feeding
// this the authored spelling would test an input the editor never sends -- and
// would miss that a base too narrow for the read-back refuses the daemon's own
// document.
//
// Owner decision 2026-09-02 is that all three must be able to author everything
// the daemon can run. check-authoring-parity.py asserts that at the level of
// which *fields* each surface exposes. This asserts the half that gate cannot
// see: that the same authored bytes replay the same MIB on the wire.
//
// The measurement is the F1a harness (internal/protocols/snmp/fidelity.go), so
// "identical" here means identical in the same terms F1a reports.
//
// Before this landed the editor was the odd one out in both directions:
// `walk_file: walks/x.walk` -- the spelling every config file and template uses
// -- was refused as "walk file not found", and an absolute path outside the
// config directory, which the other two surfaces reject, was accepted.

const f8WalkName = "brocade-icx6610-24f-01.walk"

// f8Device is the authored document the device editor sends as rawYaml. The
// same text, indented under `devices:`, is the config file and the wizard's
// configData -- one authored device, not three that happen to agree.
const f8Device = `name: f8-edge-sw1
type: switch
mac: 02:00:00:00:0f:08
ips:
  - 192.0.2.80
snmp_agent:
  community: public
  walk_file: walks/` + f8WalkName + `
  add_mibs:
    - oid: 1.3.6.1.4.1.9.9.9999.1.0
      type: STRING
      value: fixed(f8-authoring-parity)
`

// f8AddedOID is the add_mibs entry above: an OID the capture does not carry, so
// a surface that dropped add_mibs would still serve every walk row and only
// this would go missing.
const f8AddedOID = "1.3.6.1.4.1.9.9.9999.1.0"

// f8ConfigDocument wraps the authored device as a whole configuration.
func f8ConfigDocument() string {
	var out strings.Builder
	out.WriteString("devices:\n")
	for index, line := range strings.Split(strings.TrimRight(f8Device, "\n"), "\n") {
		if index == 0 {
			out.WriteString("  - " + line + "\n")

			continue
		}
		out.WriteString("    " + line + "\n")
	}
	return out.String()
}

// f8Base lays out a directory the way all three surfaces expect to find one:
// the document at the top, its walks beside it.
func f8Base(t *testing.T) string {
	t.Helper()

	base := t.TempDir()
	walks := filepath.Join(base, "walks")
	if err := os.MkdirAll(walks, 0o750); err != nil {
		t.Fatalf("create walks dir: %v", err)
	}
	source := filepath.Join("..", "library", "starter", "walks", f8WalkName)
	capture, err := os.ReadFile(source)
	if err != nil {
		t.Fatalf("read starter walk: %v", err)
	}
	if writeErr := os.WriteFile(filepath.Join(walks, f8WalkName), capture, 0o600); writeErr != nil {
		t.Fatalf("write walk: %v", writeErr)
	}
	return base
}

// f8Report replays one resolved device and reports what reached the wire.
//
// The walk-then-add_mibs order mirrors Stack.initSNMPAgent
// (internal/protocols/stack_snmp.go); this test cannot call it because
// internal/protocols imports this package's dependencies, not the other way
// round, and the agent it builds is unexported.
func f8Report(t *testing.T, surface string, device *config.Device) snmp.FidelityReport {
	t.Helper()

	agent := snmp.NewAgentWithCommunity(device, device.SNMPConfig.Community, 0)
	for _, walk := range device.SNMPConfig.WalkFiles {
		if err := agent.LoadWalkFile(walk); err != nil {
			t.Fatalf("%s: load %s: %v", surface, walk, err)
		}
	}
	for _, mib := range device.SNMPConfig.AddMibs {
		if err := agent.AddMib(mib.OID, mib.Type, mib.Value); err != nil {
			t.Fatalf("%s: add_mibs %s: %v", surface, mib.OID, err)
		}
	}
	agent.Reindex()

	source, err := snmp.ParseWalkFile(device.SNMPConfig.WalkFile)
	if err != nil {
		t.Fatalf("%s: parse source walk: %v", surface, err)
	}
	wire, orderingBreak := snmp.SweepGetNext(agent, snmp.SweepBudget(len(source)))
	if orderingBreak != "" {
		t.Fatalf("%s: sweep ordering: %s", surface, orderingBreak)
	}

	report := snmp.BuildFidelityReport(f8WalkName, source, wire, agent.WalkContract())
	if !f8Served(wire, f8AddedOID) {
		t.Errorf("%s: authored add_mibs entry %s never reached the wire", surface, f8AddedOID)
	}
	return report
}

func f8Served(wire []gosnmp.SnmpPDU, oid string) bool {
	for _, binding := range wire {
		if strings.TrimPrefix(binding.Name, ".") == oid {
			return true
		}
	}
	return false
}

func TestAuthoringSurfacesReplayIdentically(t *testing.T) {
	base := f8Base(t)

	configPath := filepath.Join(base, "network.yaml")
	if err := os.WriteFile(configPath, []byte(f8ConfigDocument()), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	fileCfg, err := config.LoadYAML(configPath)
	if err != nil {
		t.Fatalf("file surface: %v", err)
	}

	readBack, err := serializeDeviceToYAML(&fileCfg.Devices[0])
	if err != nil {
		t.Fatalf("serialize for the editor: %v", err)
	}
	editorDevice, err := parseDeviceFromYAML(string(readBack), "f8-edge-sw1", base)
	if err != nil {
		t.Fatalf("editor surface: %v", err)
	}

	// The wizard's own call, with the base the daemon gives it.
	wizardCfg, err := config.LoadYAMLBytesManaged([]byte(f8ConfigDocument()), base, nil)
	if err != nil {
		t.Fatalf("wizard surface: %v", err)
	}

	surfaces := []struct {
		name   string
		device *config.Device
	}{
		{"file", &fileCfg.Devices[0]},
		{"editor", editorDevice},
		{"wizard", &wizardCfg.Devices[0]},
	}

	reports := make([]snmp.FidelityReport, 0, len(surfaces))
	for _, surface := range surfaces {
		reports = append(reports, f8Report(t, surface.name, surface.device))
	}

	// A device that loaded nothing produces three identical empty reports, so
	// the equality below has to be anchored to what the capture actually
	// carries before it means anything.
	first := reports[0]
	if first.SourceOIDs == 0 || first.WireOIDs == 0 {
		t.Fatalf("nothing replayed: %d source OIDs, %d on the wire", first.SourceOIDs, first.WireOIDs)
	}
	if first.Rejected != 0 {
		t.Errorf("%d source rows were refused before the comparison could see them", first.Rejected)
	}
	if first.Incomplete {
		t.Fatal("the sweep stopped before end-of-MIB, so the report says nothing")
	}
	// F1a's measured baseline for this capture. Equal reports at the wrong
	// number would mean all three surfaces regressed together.
	const brocadeBaseline = 117
	if first.Unclassified != brocadeBaseline {
		t.Errorf("%d unclassified rows, F1a baseline for this capture is %d\n%s",
			first.Unclassified, brocadeBaseline, strings.Join(first.Samples, "\n"))
	}

	for index := 1; index < len(reports); index++ {
		if !reflect.DeepEqual(first, reports[index]) {
			t.Errorf("%s replays differently from %s:\n %s: %+v\n %s: %+v",
				surfaces[index].name, surfaces[0].name,
				surfaces[0].name, first, surfaces[index].name, reports[index])
		}
	}
}

// The other half of the same fix. A config file confines snmp_agent.walk_file
// to the directory the config lives in; before the editor's save path was given
// that directory it had no base to confine against, so an authenticated
// operator could point a device at any parseable file on the host and have the
// daemon serve its contents over SNMP.
func TestTheEditorRefusesAWalkOutsideTheConfigDirectory(t *testing.T) {
	base := f8Base(t)

	elsewhere := filepath.Join(t.TempDir(), "outside.walk")
	if err := os.WriteFile(elsewhere, []byte(precedenceWalkLine), 0o600); err != nil {
		t.Fatalf("write walk: %v", err)
	}

	document := "name: f8-escape\ntype: switch\nmac: 02:00:00:00:0f:0a\nips:\n  - 192.0.2.81\n" +
		"snmp_agent:\n  community: public\n  walk_file: " + elsewhere + "\n"

	_, err := parseDeviceFromYAML(document, "f8-escape", base)
	if err == nil {
		t.Fatal("the editor accepted a walk file outside the config directory")
	}
	if !strings.Contains(err.Error(), "outside base directory") {
		t.Errorf("error = %v, want it to name the containment rule", err)
	}
}

const precedenceWalkLine = ".1.3.6.1.2.1.1.1.0 = STRING: \"outside\"\n"

// A device the daemon can read, the daemon must be able to save. The shipped
// walk-backed templates reach their captures through `include_path: ../walks`,
// so the read-back document names a walk one directory *up* from the config --
// which a base of the config's own directory refuses. This is the case that
// broke when the editor was first given a base at all.
func TestATemplateDeviceSurvivesAnEditorRoundTrip(t *testing.T) {
	root := t.TempDir()
	if _, err := library.Open(root); err != nil {
		t.Fatalf("bootstrap library: %v", err)
	}
	networks := filepath.Join(root, string(library.KindNetworks))

	for _, name := range []string{"captured-switch", "captured-pair"} {
		t.Run(name, func(t *testing.T) {
			cfg, err := config.LoadYAML(filepath.Join(networks, name+".yaml"))
			if err != nil {
				t.Fatalf("load template: %v", err)
			}
			server := &Server{cfg: ServerConfig{
				ConfigPath: filepath.Join(networks, name+".yaml"),
				Config:     cfg,
			}}

			for index := range cfg.Devices {
				assertEditorRoundTrip(t, &cfg.Devices[index], server.authoredIncludeDir())
			}
		})
	}
}

func assertEditorRoundTrip(t *testing.T, device *config.Device, includeDir string) {
	t.Helper()

	readBack, err := serializeDeviceToYAML(device)
	if err != nil {
		t.Fatalf("%s: serialize: %v", device.Name, err)
	}
	saved, err := parseDeviceFromYAML(string(readBack), device.Name, includeDir)
	if err != nil {
		t.Fatalf("%s: the daemon cannot save the device it just served: %v", device.Name, err)
	}
	if saved.SNMPConfig.WalkFile != device.SNMPConfig.WalkFile {
		t.Errorf("%s: walk file changed on round trip: %q -> %q",
			device.Name, device.SNMPConfig.WalkFile, saved.SNMPConfig.WalkFile)
	}
	if len(saved.SNMPConfig.AddMibs) != len(device.SNMPConfig.AddMibs) {
		t.Errorf("%s: %d add_mibs became %d",
			device.Name, len(device.SNMPConfig.AddMibs), len(saved.SNMPConfig.AddMibs))
	}
}
