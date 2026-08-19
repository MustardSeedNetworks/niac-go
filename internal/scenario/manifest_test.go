package scenario_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/scenario"
)

func generateHospital(t *testing.T) scenario.Manifest {
	t.Helper()
	result, err := scenario.Generate(hospitalPack(t).Request)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	return result.Manifest
}

func TestManifestDeclaresItsSchemaVersion(t *testing.T) {
	if got := generateHospital(t).SchemaVersion; got != 4 {
		t.Errorf("schema version = %d, want 4", got)
	}
}

// Generation is deterministic, so the request digest is the seed: a consumer
// holding it can regenerate the same scenario byte for byte.
func TestManifestIdentityIsTheReproducibleInput(t *testing.T) {
	first := generateHospital(t)
	second := generateHospital(t)
	if first.Identity.RequestSHA256 != second.Identity.RequestSHA256 {
		t.Fatal("same request produced two identities")
	}
	if first.Identity.RequestSHA256 == "" {
		t.Fatal("identity carries no request digest")
	}

	changed := hospitalPack(t).Request
	changed.Domain = "somewhere.else"
	other, err := scenario.Generate(changed)
	if err != nil {
		t.Fatal(err)
	}
	if other.Manifest.Identity.RequestSHA256 == first.Identity.RequestSHA256 {
		t.Error("a different request kept the same identity")
	}
	if first.Identity.Domain == "" {
		t.Error("identity omits the authored domain")
	}
}

func TestManifestRecordsInterfaceTruth(t *testing.T) {
	manifest := generateHospital(t)
	if manifest.Interfaces.Count == 0 {
		t.Fatal("interface truth carries no interfaces")
	}
	if manifest.Interfaces.SHA256 == "" {
		t.Error("interface truth carries no digest")
	}
	// The guided hospital story saturates both uplinks of MED-ACC-SW02 and the
	// matching distribution ends, deliberately above the 80% warning line.
	if len(manifest.Interfaces.Congested) != 4 {
		t.Errorf("congested interfaces = %d, want 4: %+v",
			len(manifest.Interfaces.Congested), manifest.Interfaces.Congested)
	}
}

func TestManifestExpectedObservationsFollowTheAuthoredConfig(t *testing.T) {
	observations := generateHospital(t).Observations
	for _, collector := range []string{"sys_info", "if_table", "lldp", "cdp"} {
		observation, ok := observations[collector]
		if !ok {
			t.Errorf("no expectation for %s", collector)
			continue
		}
		if observation.Devices == 0 {
			t.Errorf("%s expects no devices", collector)
		}
	}
	if observations["if_table"].Rows == 0 {
		t.Error("if_table expectation carries no interface rows")
	}
	if observations["sys_info"].Devices > generateHospital(t).DeviceCount {
		t.Error("more devices answer sys_info than exist")
	}
}

// An absent collector means the scenario authors nothing that collector reads.
// That is not the same claim as a count of zero, and a consumer that treats
// them alike will assert emptiness the scenario never promised.
func TestManifestOmitsObservationsNothingAuthors(t *testing.T) {
	observations := generateHospital(t).Observations
	for _, collector := range []string{"bgp4_mib", "host_resources", "fdp"} {
		if _, ok := observations[collector]; ok {
			t.Errorf("%s expectation exists but the hospital pack authors none", collector)
		}
	}
}

// Neighbour tables cannot be complete before every advertiser has transmitted
// once, so the tolerance follows the slowest authored advertisement interval
// rather than a number someone picked.
func TestManifestTimingFollowsProtocolAdvertisementIntervals(t *testing.T) {
	timing := generateHospital(t).Timing
	if timing.LLDPIntervalSeconds != 15 || timing.CDPIntervalSeconds != 15 {
		t.Errorf("intervals = %+v, want LLDP and CDP at 15s", timing)
	}
	if timing.FDPIntervalSeconds != 0 {
		t.Error("hospital authors no FDP but the manifest states an FDP interval")
	}
	if timing.NeighborsStableAfterSeconds != 30 {
		t.Errorf("stable-after = %ds, want 30 (two 15s cycles)",
			timing.NeighborsStableAfterSeconds)
	}
}

// Physical interface and VLAN are deployment identity. A pack is portable and
// must stay silent about where it happens to be attached.
func TestManifestNeverCarriesPhysicalBinding(t *testing.T) {
	encoded, err := json.Marshal(generateHospital(t))
	if err != nil {
		t.Fatal(err)
	}
	// Logical interface names are pack content and appear in congestion. What
	// must never appear is where the pack is attached: a physical VLAN, an
	// access VLAN, an attachment mode, or a host NIC.
	for _, forbidden := range []string{
		"physicalVlan", "accessVlan", "attachmentMode", "eth0",
	} {
		if strings.Contains(string(encoded), forbidden) {
			t.Errorf("manifest leaks physical binding %q:\n%s", forbidden, encoded)
		}
	}
}

// The version 4 additions are derived, not frozen, so a pack still pins only
// the parity subset. This asserts the packs endpoint's shape is unchanged by
// the bump — consumers reading pack.manifest.deviceCount are unaffected.
func TestPackManifestKeepsItsPublishedShape(t *testing.T) {
	encoded, err := json.Marshal(hospitalPack(t))
	if err != nil {
		t.Fatal(err)
	}
	var decoded struct {
		ManifestVersion int                        `json:"manifestVersion"`
		Manifest        map[string]json.RawMessage `json:"manifest"`
	}
	if err = json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.ManifestVersion != 4 {
		t.Errorf("manifestVersion = %d, want 4", decoded.ManifestVersion)
	}
	want := []string{
		"deviceCount", "networkCount", "linkCount",
		"deviceNamesSha256", "networksSha256", "linksSha256",
	}
	if len(decoded.Manifest) != len(want) {
		t.Fatalf("pack manifest keys = %v, want exactly %v", decoded.Manifest, want)
	}
	for _, key := range want {
		if _, ok := decoded.Manifest[key]; !ok {
			t.Errorf("pack manifest lost %s", key)
		}
	}
}
