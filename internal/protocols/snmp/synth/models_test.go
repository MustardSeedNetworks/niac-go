package synth_test

import (
	"strings"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/protocols/snmp/synth"
)

func TestModelsReturnsOnlyRequestedVendor(t *testing.T) {
	got := synth.Models(synth.VendorCiscoIOS)
	if len(got) == 0 {
		t.Fatal("cisco-ios should have models defined")
	}
	for _, m := range got {
		if m.Vendor != synth.VendorCiscoIOS {
			t.Errorf("Models(cisco-ios) returned %s entry %q", m.Vendor, m.Model)
		}
	}
}

func TestModelsForTypeFilters(t *testing.T) {
	got := synth.ModelsForType(synth.VendorCiscoIOS, synth.TypeSwitch)
	if len(got) == 0 {
		t.Fatal("cisco-ios + switch should have model entries")
	}
	for _, m := range got {
		if m.Type != synth.TypeSwitch {
			t.Errorf("ModelsForType(switch) returned type=%s for %q", m.Type, m.Model)
		}
	}
}

func TestModelsSortedStable(t *testing.T) {
	// Models are sorted by (Type, Label). Two consecutive calls must
	// produce the same slice so the UI doesn't flicker on re-render.
	a := synth.Models(synth.VendorJunos)
	b := synth.Models(synth.VendorJunos)
	if len(a) != len(b) {
		t.Fatalf("length differs: %d vs %d", len(a), len(b))
	}
	for i := range a {
		if a[i].Model != b[i].Model {
			t.Errorf("order differs at %d: %s vs %s", i, a[i].Model, b[i].Model)
		}
	}
}

func TestPickModelCatalystSetsExpectedFields(t *testing.T) {
	p, err := synth.PickModel(synth.VendorCiscoIOS, "catalyst-9300-48p")
	if err != nil {
		t.Fatalf("PickModel: %v", err)
	}
	if p.SysObjectID != "1.3.6.1.4.1.9.1.2238" {
		t.Errorf("sysObjectID = %q, want Catalyst 9300 OID", p.SysObjectID)
	}
	if !strings.Contains(p.SysDescr, "CAT9K_IOSXE") {
		t.Errorf("sysDescr should mention CAT9K_IOSXE platform image, got %q", p.SysDescr)
	}
	if p.DefaultIfCount != 48 {
		t.Errorf("ifCount = %d, want 48", p.DefaultIfCount)
	}
	if p.Type != synth.TypeSwitch {
		t.Errorf("type = %q, want switch", p.Type)
	}
	// Switch flags inherited from the vendor's per-type profile.
	if !p.IncludeBridge {
		t.Error("switch should include BRIDGE-MIB")
	}
	if p.IncludeIPMIB {
		t.Error("switch should not include IP-MIB")
	}
}

func TestPickModelMistOverridesFlagsToMinimal(t *testing.T) {
	// Mist AP43 sets overrideFlags to a zero profileFlags struct so
	// LLDP / BRIDGE / IP-MIB are all off (API-first SNMP per the
	// design-doc resolution).
	p, err := synth.PickModel(synth.VendorJuniperMist, "mist-ap43")
	if err != nil {
		t.Fatalf("PickModel: %v", err)
	}
	if p.IncludeIPMIB || p.IncludeBridge || p.IncludeLLDP || p.IPForwarding {
		t.Errorf("Mist AP43 should have all include flags off, got %+v", p)
	}
}

func TestPickModelUnknownReturnsError(t *testing.T) {
	_, err := synth.PickModel(synth.VendorCiscoIOS, "catalyst-99999-fake")
	if err == nil {
		t.Fatal("unknown model should error")
	}
}

func TestAllModelsCoversEveryVendor(t *testing.T) {
	got := synth.AllModels()
	if len(got) == 0 {
		t.Fatal("AllModels returned empty")
	}
	seen := map[synth.Vendor]bool{}
	for _, m := range got {
		seen[m.Vendor] = true
	}
	// At least one model per top-tier vendor — sanity that we shipped
	// real coverage rather than just stubs.
	for _, want := range []synth.Vendor{
		synth.VendorCiscoIOS, synth.VendorJunos, synth.VendorAristaEOS,
	} {
		if !seen[want] {
			t.Errorf("AllModels missing entries for %s", want)
		}
	}
}

func TestBuildWithModelEmitsModelSpecificSysDescr(t *testing.T) {
	// End-to-end sanity: PickModel → Build produces a walk with the
	// model's sysDescr verbatim. This is the whole point of the
	// vendor+model expansion — the synthesised walk should look like
	// the real gear's snmpwalk output.
	p, _ := synth.PickModel(synth.VendorAristaEOS, "dcs-7280sr")
	walk := string(synth.Build(p, synth.DeviceInput{
		Hostname: "spine-01",
		IP:       "10.0.0.1",
	}, synth.BuildOptions{InterfaceCount: 4}))

	mustContain(t, walk, "DCS-7280SR-48C6")
	// And the platform's enterprise OID is in the walk too.
	mustContain(t, walk, "1.3.6.1.4.1.30065")
	// Spine = 100G — confirm ifSpeed gauge reflects that (100G *
	// 1_000_000 Mbps→bps = 100_000_000_000, won't fit Gauge32; the
	// emitter clamps via safeconv.Uint32 so verify the line exists).
	mustContain(t, walk, ".1.3.6.1.2.1.2.2.1.5.1 = Gauge32:")
}
