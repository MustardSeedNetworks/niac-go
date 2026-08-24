package config

import (
	"net"
	"reflect"
	"testing"
)

// snmpConfigFieldsCoveredByUnset is the number of fields unset() inspects.
//
// Keep it equal to the number of fields on SNMPConfig. If you added one, add it
// to unset() as well and bump this number — see the test below for why.
const snmpConfigFieldsCoveredByUnset = 15

// TestUnsetCoversEverySNMPConfigField pins #1460's fix against silent rot.
//
// snmpToYAML omits an SNMPConfig that unset() reports as empty. unset() is a
// hand-written conjunction over every field, so adding a field to SNMPConfig
// without extending unset() makes a device that configures only that field
// marshal as if it had no SNMP block at all — quietly reintroducing #1460 for
// the new field, with no test failing.
//
// A field count is a blunt instrument, but it is the one signal that cannot be
// forgotten: the struct cannot grow without this test going red.
//
// reflect.Value.IsZero() is deliberately not used in unset() instead. Its
// semantics differ on an empty-but-non-nil slice — `walk_files: []` is "unset"
// under the len()==0 check and "set" under IsZero — and the len() reading is
// the one that keeps the round trip faithful.
func TestUnsetCoversEverySNMPConfigField(t *testing.T) {
	got := reflect.TypeFor[SNMPConfig]().NumField()
	if got != snmpConfigFieldsCoveredByUnset {
		t.Fatalf(
			"SNMPConfig has %d fields but unset() is documented to cover %d.\n"+
				"Add the new field to unset() in yaml_marshal_protocols.go and update "+
				"snmpConfigFieldsCoveredByUnset, or a device configuring only that field "+
				"will marshal with no snmp_agent block at all (#1460).",
			got, snmpConfigFieldsCoveredByUnset,
		)
	}
}

// TestUnsetIsTrueOnlyForTheZeroValue is the behavioural half: every single
// field, set on its own, must make the config count as configured.
func TestUnsetIsTrueOnlyForTheZeroValue(t *testing.T) {
	if !(&SNMPConfig{}).unset() {
		t.Fatal("the zero SNMPConfig should be unset")
	}

	enabled := true
	cases := map[string]SNMPConfig{
		"Enabled":           {Enabled: &enabled},
		"Community":         {Community: "public"},
		"SysName":           {SysName: "sw-1"},
		"SysDescr":          {SysDescr: "a switch"},
		"SysContact":        {SysContact: "noc"},
		"SysLocation":       {SysLocation: "rack 1"},
		"WalkFile":          {WalkFile: "a.walk"},
		"WalkFiles":         {WalkFiles: []string{"a.walk"}},
		"AddMibs":           {AddMibs: []AddMib{{OID: "1.3.6.1"}}},
		"CommunityIncludes": {CommunityIncludes: []CommunityInclude{{Community: "public"}}},
		"AccessList":        {AccessList: parseTestIPs(t, "10.0.0.1")},
		"SnmpAddr":          {SnmpAddr: parseTestIP(t, "10.0.0.2")},
		"Dot1DFdbTable":     {Dot1DFdbTable: &FdbTableConfig{}},
		"Dot1QFdbTable":     {Dot1QFdbTable: &FdbTableConfig{}},
		"Traps":             {Traps: &TrapConfig{}},
	}

	if len(cases) != snmpConfigFieldsCoveredByUnset {
		t.Fatalf("this table covers %d fields, want %d", len(cases), snmpConfigFieldsCoveredByUnset)
	}
	for name, cfg := range cases {
		t.Run(name, func(t *testing.T) {
			if cfg.unset() {
				t.Errorf("a config with only %s set reports unset, so its block would be dropped", name)
			}
		})
	}
}

func parseTestIP(t *testing.T, s string) net.IP {
	t.Helper()
	ip := net.ParseIP(s)
	if ip == nil {
		t.Fatalf("parse %q", s)
	}
	return ip
}

func parseTestIPs(t *testing.T, s string) []net.IP {
	t.Helper()
	return []net.IP{parseTestIP(t, s)}
}
