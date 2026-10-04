package fabric_test

import (
	"encoding/json"
	"net/netip"
	"strings"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/fabric"
)

func dualStackConfig() *config.Config {
	cfg := referenceConfig()
	cfg.Networks[0].SubnetV6 = "2001:db8:200::/64"
	cfg.Networks[1].SubnetV6 = "fd00:20:210::/64"
	cfg.Devices[0].Interfaces[0].AddressV6 = "2001:db8:200::1/64"
	cfg.Devices[0].Interfaces[1].AddressV6 = "fd00:20:210::1/64"
	return cfg
}

func TestCompileCarriesIPv6PrefixesBesideIPv4(t *testing.T) {
	report := fabric.Compile(dualStackConfig(), accessBinding())
	if !report.Safe {
		t.Fatalf("Compile() diagnostics = %#v", report.Diagnostics)
	}

	networks := make(map[string]fabric.Network)
	for _, network := range report.Topology.Networks {
		networks[network.Name] = network
	}
	if got, want := networks["lab-access"].PrefixV6, netip.MustParsePrefix("2001:db8:200::/64"); got != want {
		t.Fatalf("lab-access PrefixV6 = %s, want %s", got, want)
	}
	if got, want := networks["lab-access"].Prefix, netip.MustParsePrefix("10.10.200.0/24"); got != want {
		t.Fatalf("lab-access Prefix = %s, want %s", got, want)
	}

	addresses := make(map[string]netip.Prefix)
	for _, iface := range report.Topology.Interfaces {
		addresses[iface.Device+"/"+iface.Name] = iface.AddressV6
	}
	if got, want := addresses["LAB-EDGE-R1/outside"], netip.MustParsePrefix("2001:db8:200::1/64"); got != want {
		t.Fatalf("outside AddressV6 = %s, want %s", got, want)
	}
	if got := addresses["LAB-DHCP01/eth0"]; got.IsValid() {
		t.Fatalf("IPv4-only interface AddressV6 = %s, want none", got)
	}
}

func TestCompileRejectsInvalidIPv6Semantics(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*config.Config)
		code   fabric.DiagnosticCode
		field  string
	}{
		{
			name:   "subnet is IPv4",
			mutate: func(cfg *config.Config) { cfg.Networks[0].SubnetV6 = "10.99.0.0/24" },
			code:   fabric.CodeInvalidNetwork,
			field:  "networks[0].subnet_v6",
		},
		{
			name:   "subnet is IPv4-mapped",
			mutate: func(cfg *config.Config) { cfg.Networks[0].SubnetV6 = "::ffff:10.99.0.0/120" },
			code:   fabric.CodeInvalidNetwork,
			field:  "networks[0].subnet_v6",
		},
		{
			name:   "subnet not at its network address",
			mutate: func(cfg *config.Config) { cfg.Networks[0].SubnetV6 = "2001:db8:200::1/64" },
			code:   fabric.CodeInvalidNetwork,
			field:  "networks[0].subnet_v6",
		},
		{
			name:   "subnet is link-local",
			mutate: func(cfg *config.Config) { cfg.Networks[0].SubnetV6 = "fe80::/64" },
			code:   fabric.CodeInvalidNetwork,
			field:  "networks[0].subnet_v6",
		},
		{
			name:   "subnet is multicast",
			mutate: func(cfg *config.Config) { cfg.Networks[0].SubnetV6 = "ff05::/64" },
			code:   fabric.CodeInvalidNetwork,
			field:  "networks[0].subnet_v6",
		},
		{
			name:   "subnet is a single address",
			mutate: func(cfg *config.Config) { cfg.Networks[0].SubnetV6 = "2001:db8:200::1/128" },
			code:   fabric.CodeInvalidNetwork,
			field:  "networks[0].subnet_v6",
		},
		{
			name:   "subnets overlap",
			mutate: func(cfg *config.Config) { cfg.Networks[1].SubnetV6 = "2001:db8:200::/56" },
			code:   fabric.CodeOverlappingNetworks,
			field:  "networks[1].subnet_v6",
		},
		{
			name: "network has no IPv6 subnet",
			mutate: func(cfg *config.Config) {
				cfg.Networks[0].SubnetV6 = ""
			},
			code:  fabric.CodeAddressOutsideNetwork,
			field: "devices[LAB-EDGE-R1].interfaces[0].address_v6",
		},
		{
			name: "address is IPv4",
			mutate: func(cfg *config.Config) {
				cfg.Devices[0].Interfaces[0].AddressV6 = "10.10.200.9/24"
			},
			code:  fabric.CodeInvalidInterfaceAddress,
			field: "devices[LAB-EDGE-R1].interfaces[0].address_v6",
		},
		{
			name: "address is bare",
			mutate: func(cfg *config.Config) {
				cfg.Devices[0].Interfaces[0].AddressV6 = "2001:db8:200::1"
			},
			code:  fabric.CodeInvalidInterfaceAddress,
			field: "devices[LAB-EDGE-R1].interfaces[0].address_v6",
		},
		{
			name: "address outside network",
			mutate: func(cfg *config.Config) {
				cfg.Devices[0].Interfaces[0].AddressV6 = "2001:db8:999::1/64"
			},
			code:  fabric.CodeAddressOutsideNetwork,
			field: "devices[LAB-EDGE-R1].interfaces[0].address_v6",
		},
		{
			name: "address prefix differs from network",
			mutate: func(cfg *config.Config) {
				cfg.Devices[0].Interfaces[0].AddressV6 = "2001:db8:200::1/80"
			},
			code:  fabric.CodeInterfacePrefixMismatch,
			field: "devices[LAB-EDGE-R1].interfaces[0].address_v6",
		},
		{
			name: "address is the subnet-router anycast address",
			mutate: func(cfg *config.Config) {
				cfg.Devices[0].Interfaces[0].AddressV6 = "2001:db8:200::/64"
			},
			code:  fabric.CodeReservedInterfaceAddr,
			field: "devices[LAB-EDGE-R1].interfaces[0].address_v6",
		},
		{
			name: "address assigned twice",
			mutate: func(cfg *config.Config) {
				cfg.Devices[1].Interfaces[0].AddressV6 = "2001:db8:200::1/64"
			},
			code:  fabric.CodeDuplicateInterfaceAddr,
			field: "devices[LAB-DHCP01].interfaces[0].address_v6",
		},
		{
			name: "address without a network",
			mutate: func(cfg *config.Config) {
				cfg.Devices[2].Interfaces = append(cfg.Devices[2].Interfaces, config.Interface{
					Name: "lo6", AddressV6: "2001:db8:200::9/64",
				})
			},
			code:  fabric.CodeUnknownNetwork,
			field: "devices[EVT-EDGE-R1].interfaces[1].network",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := dualStackConfig()
			tt.mutate(cfg)
			report := fabric.Compile(cfg, accessBinding())
			for _, diagnostic := range report.Diagnostics {
				if diagnostic.Code == tt.code && diagnostic.Field == tt.field {
					return
				}
			}
			t.Fatalf("diagnostics = %#v, want %q at %s", report.Diagnostics, tt.code, tt.field)
		})
	}
}

func TestCompileAllowsAnycastIdentifierOnSlash127(t *testing.T) {
	cfg := dualStackConfig()
	cfg.Networks[1].SubnetV6 = "fd00:20:210::/127"
	cfg.Devices[0].Interfaces[1].AddressV6 = "fd00:20:210::/127"
	cfg.Devices[2].Interfaces[0].AddressV6 = "fd00:20:210::1/127"

	report := fabric.Compile(cfg, accessBinding())
	if !report.Safe {
		t.Fatalf("Compile() diagnostics = %#v", report.Diagnostics)
	}
}

// An IPv4-only scenario must marshal exactly as it did before the IPv6 fields
// existed, so the topology JSON gains a key only where an author wrote one.
func TestTopologyJSONOmitsAbsentIPv6(t *testing.T) {
	for _, tt := range []struct {
		name string
		cfg  *config.Config
		want []string
		deny []string
	}{
		{
			name: "IPv4 only",
			cfg:  referenceConfig(),
			deny: []string{"prefixV6", "addressV6"},
		},
		{
			name: "dual-stack",
			cfg:  dualStackConfig(),
			want: []string{`"prefixV6":"2001:db8:200::/64"`, `"addressV6":"2001:db8:200::1/64"`},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			report := fabric.Compile(tt.cfg, accessBinding())
			raw, err := json.Marshal(report.Topology)
			if err != nil {
				t.Fatal(err)
			}
			for _, want := range tt.want {
				if !strings.Contains(string(raw), want) {
					t.Errorf("topology JSON lacks %s:\n%s", want, raw)
				}
			}
			for _, deny := range tt.deny {
				if strings.Contains(string(raw), deny) {
					t.Errorf("topology JSON carries %s:\n%s", deny, raw)
				}
			}
		})
	}
}
