package scenario_test

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/scenario"
)

func TestGenerateEnterpriseReferenceMatchesAcceptedTopology(t *testing.T) {
	request := scenario.EnterpriseReferenceRequest()
	first, err := scenario.Generate(request)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	second, err := scenario.Generate(request)
	if err != nil {
		t.Fatalf("second Generate() error = %v", err)
	}
	if !bytes.Equal(first.YAML, second.YAML) {
		t.Fatal("same request produced different YAML")
	}

	want := scenario.Parity{
		DeviceCount:       543,
		NetworkCount:      39,
		LinkCount:         634,
		DeviceNamesSHA256: "8514a6d423b598a11d6ebc6edfc399c978883b831106e8c187e681619229346f",
		NetworksSHA256:    "e879b7ba38e40f925809edc3bf98d2044959df5d2f76d492e6f2019cbcba5555",
		// Routed WAN edges carry no VLAN metadata; switched links retain their trunks.
		LinksSHA256: "4c1acbf07eccc6464a4a86d8a53f867fdaa1cc7374d18b881bf30487a98713e6",
	}
	if first.Manifest.Parity() != want {
		t.Fatalf("manifest = %#v, want %#v", first.Manifest.Parity(), want)
	}

	cfg, err := config.LoadYAMLBytes(first.YAML)
	if err != nil {
		t.Fatalf("generated YAML does not load: %v", err)
	}
	if result := config.NewValidator("generated-enterprise.yaml").Validate(cfg); !result.Valid ||
		result.HasWarnings() {
		t.Fatalf("generated config is not strict-clean: %s", result.Format())
	}
	assertEnterpriseDeviceMix(t, cfg)
	assertAPDiscoveryIdentity(t, cfg)
	assertAuthoredInterfacesAndLinks(t, cfg)
	assertServiceDNS(t, cfg)
	assertLabEdgeDNS(t, cfg)
	assertUniqueIdentityAndRoutes(t, cfg)
}

func TestGenerateHonorsFleetRepeatControls(t *testing.T) {
	request := scenario.EnterpriseReferenceRequest()
	request.Sites = request.Sites[:2]
	request.Counts = scenario.Counts{
		SiteWANRouters:        2,
		Firewalls:             2,
		CoreSwitches:          2,
		DistributionSwitches:  2,
		AccessSwitches:        4,
		ServerSwitches:        2,
		AccessPointsPerAccess: 3,
		WorkstationsPerAccess: 5,
		WirelessControllers:   2,
	}

	result, err := scenario.Generate(request)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	cfg, err := config.LoadYAMLBytes(result.YAML)
	if err != nil {
		t.Fatalf("generated YAML does not load: %v", err)
	}
	// Three global devices plus 57 at each site: 54, and the three appended
	// LLDP-MED endpoints (two phones, one camera).
	if got, want := cfg.DeviceCount(), 117; got != want {
		t.Fatalf("device count = %d, want %d", got, want)
	}
	for _, site := range request.Sites {
		if got := countNamed(cfg, site.Code+"-WAP-"); got != 12 {
			t.Fatalf("%s access points = %d, want 12", site.Code, got)
		}
		got := countNamed(cfg, site.Code+"-WS-") + countNamed(cfg, site.Code+"-LAP-") +
			countNamed(cfg, site.Code+"-MBP-")
		if got != 20 {
			t.Fatalf("%s wired clients = %d, want 20", site.Code, got)
		}
	}
	assertUniqueIdentityAndRoutes(t, cfg)
}

func TestGenerateSingleSiteRouterDoesNotRouteThroughMissingPeer(t *testing.T) {
	request := scenario.EnterpriseReferenceRequest()
	request.Sites = request.Sites[:1]
	request.Counts.SiteWANRouters = 1
	request.Counts.Firewalls = 1
	request.Counts.CoreSwitches = 1

	result, err := scenario.Generate(request)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	cfg, err := config.LoadYAMLBytes(result.YAML)
	if err != nil {
		t.Fatalf("generated YAML does not load: %v", err)
	}
	assertUniqueIdentityAndRoutes(t, cfg)
}

func TestProfileCatalogUsesUniqueRoles(t *testing.T) {
	seen := make(map[string]bool)
	for _, profile := range scenario.Profiles() {
		if profile.Role == "" || profile.DeviceType == "" || profile.Vendor == "" ||
			profile.Model == "" {
			t.Errorf("incomplete profile: %+v", profile)
		}
		if seen[profile.Role] {
			t.Errorf("duplicate role profile %q", profile.Role)
		}
		if profile.Vendor == "cisco" &&
			!strings.HasPrefix(profile.SysObjectID, "1.3.6.1.4.1.9.1.") {
			t.Errorf(
				"%s Cisco profile has non-Cisco sysObjectID %q",
				profile.Role,
				profile.SysObjectID,
			)
		}
		seen[profile.Role] = true
	}
	for _, role := range []string{
		"lab", "wan", "firewall", "core", "distribution", "access",
		"server-switch", "ap", "workstation", "windows-laptop", "macbook",
		"nurse-station", "infusion-pump", "mr-system", "rugged-handheld", "barcode-printer",
		"plc", "hmi", "robot-controller", "point-of-sale", "receipt-printer", "digital-signage",
		"noc-workstation", "server", "controller",
	} {
		if !seen[role] {
			t.Errorf("missing profile for role %q", role)
		}
	}
	for _, profile := range scenario.Profiles() {
		if profile.Role == "ap" && profile.SysObjectID != "1.3.6.1.4.1.9.1.525" {
			t.Errorf(
				"AP sysObjectID = %q, want Link-Live-recognized Cisco AP identity",
				profile.SysObjectID,
			)
		}
	}
}

func TestVerticalPacksGenerateDistinctEndpointProfiles(t *testing.T) {
	expected := map[string][]string{
		"hospital":      {"nurse-station", "infusion-pump", "mr-system"},
		"warehouse":     {"rugged-handheld", "barcode-printer"},
		"manufacturing": {"plc", "hmi", "robot-controller"},
	}
	for _, pack := range scenario.Packs() {
		roles, found := expected[pack.ID]
		if !found {
			continue
		}
		result, err := scenario.Generate(pack.Request)
		if err != nil {
			t.Fatalf("Generate(%s): %v", pack.ID, err)
		}
		cfg, err := config.LoadYAMLBytes(result.YAML)
		if err != nil {
			t.Fatalf("LoadYAMLBytes(%s): %v", pack.ID, err)
		}
		seen := make(map[string]bool)
		for _, device := range cfg.Devices {
			seen[device.Properties["role"]] = true
		}
		for _, role := range roles {
			if !seen[role] {
				t.Errorf("%s has no %s endpoint", pack.ID, role)
			}
		}
	}
}

func ExampleGenerate() {
	result, _ := scenario.Generate(scenario.EnterpriseReferenceRequest())
	fmt.Println(result.Manifest.DeviceCount)
	// Output: 543
}
