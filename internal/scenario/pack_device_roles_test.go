package scenario_test

import (
	"fmt"
	"net/netip"
	"os"
	"sort"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/MustardSeedNetworks/niac-go/internal/scenario"
)

// Plan row F3(b): the EtherScope capture records what a discovery instrument
// asks each pack device for, and the only identifier on the wire is the target
// address. Turning that into a per-role demand matrix needs address -> role,
// which is exactly what Generate already decides. Writing the map here rather
// than transcribing it keeps it correct when the packs change: the pinned
// counts below fail the moment the generator's role mix moves.

type packDeviceRole struct {
	pack    string
	address string
	role    string
	device  string
}

func packDeviceRoles(t *testing.T) []packDeviceRole {
	t.Helper()

	var rows []packDeviceRole
	for _, pack := range scenario.Packs() {
		result, genErr := scenario.Generate(pack.Request)
		if genErr != nil {
			t.Fatalf("generate %s: %v", pack.ID, genErr)
		}
		var doc struct {
			Devices []struct {
				Name       string `yaml:"name"`
				Type       string `yaml:"type"`
				Interfaces []struct {
					Address string `yaml:"address"`
				} `yaml:"interfaces"`
			} `yaml:"devices"`
		}
		if unmarshalErr := yaml.Unmarshal(result.YAML, &doc); unmarshalErr != nil {
			t.Fatalf("unmarshal %s: %v", pack.ID, unmarshalErr)
		}
		for _, device := range doc.Devices {
			for _, iface := range device.Interfaces {
				address, ok := interfaceAddress(iface.Address)
				if !ok {
					continue
				}
				rows = append(rows, packDeviceRole{
					pack: pack.ID, address: address, role: device.Type, device: device.Name,
				})
			}
		}
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].pack != rows[j].pack {
			return rows[i].pack < rows[j].pack
		}
		return rows[i].address < rows[j].address
	})

	return rows
}

// interfaceAddress drops the prefix length. An interface with no address is not
// a target a manager can poll, so it is not in the map at all.
func interfaceAddress(authored string) (string, bool) {
	if authored == "" {
		return "", false
	}
	prefix, err := netip.ParsePrefix(authored)
	if err != nil {
		return "", false
	}

	return prefix.Addr().String(), true
}

// TestPackDeviceRoleMix pins the role mix the demand matrix in
// docs/design/consumer-oid-demand.tsv was cut against. A change here means that
// file is stale, not that this test is wrong.
func TestPackDeviceRoleMix(t *testing.T) {
	want := map[string]map[string]int{
		"campus": {
			"access-point":  32,
			"firewall":      8,
			"host":          28,
			"iot":           4,
			"layer3-switch": 8,
			"printer":       4,
			"router":        11,
			"server":        32,
			"switch":        24,
			"voip-phone":    8,
		},
		"enterprise-scale": {
			"access-point":  128,
			"firewall":      8,
			"host":          232,
			"iot":           4,
			"layer3-switch": 8,
			"printer":       24,
			"router":        11,
			"server":        32,
			"switch":        88,
			"voip-phone":    8,
		},
		"hospital": {
			"access-point":  30,
			"firewall":      2,
			"host":          4,
			"iot":           13,
			"layer3-switch": 2,
			"printer":       2,
			"router":        5,
			"server":        8,
			"switch":        10,
			"voip-phone":    2,
		},
		"manufacturing": {
			"access-point":  30,
			"firewall":      2,
			"iot":           12,
			"layer3-switch": 2,
			"printer":       1,
			"router":        5,
			"server":        8,
			"switch":        10,
			"voip-phone":    2,
		},
		"retail": {
			"access-point":  24,
			"firewall":      4,
			"host":          8,
			"iot":           10,
			"layer3-switch": 4,
			"printer":       8,
			"router":        7,
			"server":        16,
			"switch":        16,
			"voip-phone":    4,
		},
		"service-provider": {
			"access-point":  24,
			"firewall":      6,
			"host":          21,
			"iot":           3,
			"layer3-switch": 6,
			"printer":       3,
			"router":        9,
			"server":        24,
			"switch":        24,
			"voip-phone":    6,
		},
		"warehouse": {
			"access-point":  27,
			"firewall":      2,
			"host":          3,
			"iot":           1,
			"layer3-switch": 2,
			"printer":       3,
			"router":        5,
			"server":        8,
			"switch":        7,
			"voip-phone":    2,
		},
	}

	got := map[string]map[string]int{}
	seen := map[string]map[string]bool{}
	for _, row := range packDeviceRoles(t) {
		if seen[row.pack] == nil {
			seen[row.pack] = map[string]bool{}
			got[row.pack] = map[string]int{}
		}
		if seen[row.pack][row.device] {
			continue
		}
		seen[row.pack][row.device] = true
		got[row.pack][row.role]++
	}

	if len(want) != len(got) {
		t.Errorf("pack count: got %d, want %d", len(got), len(want))
	}
	for pack, counts := range want {
		for role, count := range counts {
			if got[pack][role] != count {
				t.Errorf("%s %s: got %d devices, want %d", pack, role, got[pack][role], count)
			}
		}
		for role, count := range got[pack] {
			if _, expected := counts[role]; !expected {
				t.Errorf("%s: unexpected role %s with %d devices", pack, role, count)
			}
		}
	}
}

// TestWritePackDeviceRoles is the map generator, not an assertion. It writes the
// address -> role map scripts/extract-oid-demand.py consumes, and is skipped
// unless an operator asks for it by setting the destination.
func TestWritePackDeviceRoles(t *testing.T) {
	destination := os.Getenv("NIAC_ROLE_MAP_OUT")
	if destination == "" {
		t.Skip("set NIAC_ROLE_MAP_OUT to regenerate the pack device role map")
	}

	var out strings.Builder
	out.WriteString("# pack\taddress\trole\n")
	out.WriteString(
		"# Generated by: NIAC_ROLE_MAP_OUT=<path> go test ./internal/scenario -run TestWritePackDeviceRoles\n",
	)
	for _, row := range packDeviceRoles(t) {
		fmt.Fprintf(&out, "%s\t%s\t%s\n", row.pack, row.address, row.role)
	}
	if err := os.WriteFile(destination, []byte(out.String()), 0o600); err != nil {
		t.Fatalf("write %s: %v", destination, err)
	}
}
