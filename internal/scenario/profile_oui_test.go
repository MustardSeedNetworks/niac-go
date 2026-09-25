package scenario_test

import (
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/oui"
	"github.com/MustardSeedNetworks/niac-go/internal/scenario"
)

// A profile's vendor becomes its MAC prefix, and the registry resolves it by
// substring, so a short vendor name can take an unrelated organization's OUI:
// the packs' UPS once went out as Adapcom and the camera as Galaxis. A
// discovery tool reports that organization as the device's manufacturer.
// Adding a vendor means stating here whose prefix it should get.
func TestEveryProfileVendorAllocatesItsOwnOUI(t *testing.T) {
	want := map[string]string{
		"apc":                 "AMERICAN POWER CONVERSION CORP",
		"adtran":              "Adtran Inc",
		"apple":               "Apple, Inc.",
		"axis":                "Axis Communications AB",
		"baxter":              "Baxter International Inc",
		"calix":               "Calix Networks",
		"cisco":               "Cisco Systems, Inc",
		"dell":                "Dell EMC",
		"fanuc robotics":      "FANUC ROBOTICS NORTH AMERICA, Inc.",
		"ge healthcare":       "GE Healthcare",
		"hewlett packard":     "Hewlett Packard",
		"hid global":          "Crossmatch Technologies/HID Global",
		"palo alto":           "Palo Alto Networks",
		"philips healthcare":  "Philips Healthcare PCCI",
		"rockwell automation": "Rockwell Automation",
		"samsung":             "Samsung Electronics Co.,Ltd",
		"seiko epson":         "Seiko Epson Corporation",
		"siemens":             "Siemens AG",
		"zebra":               "Zebra Technologies Inc",
	}
	registry, err := oui.LoadEmbedded()
	if err != nil {
		t.Fatalf("LoadEmbedded() error = %v", err)
	}
	for _, profile := range scenario.Profiles() {
		organization, stated := want[profile.Vendor]
		if !stated {
			t.Errorf("profile %s: vendor %q has no expected OUI organization", profile.Role, profile.Vendor)
			continue
		}
		mac, allocateErr := registry.Allocate(profile.Vendor, 1)
		if allocateErr != nil {
			t.Errorf("profile %s: Allocate(%q) error = %v", profile.Role, profile.Vendor, allocateErr)
			continue
		}
		if got, _ := registry.Lookup(mac); got != organization {
			t.Errorf("profile %s: vendor %q allocates %s, registered to %q; want %q",
				profile.Role, profile.Vendor, mac, got, organization)
		}
	}
}
