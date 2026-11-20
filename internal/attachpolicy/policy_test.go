// SPDX-License-Identifier: BUSL-1.1

package attachpolicy_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/attachpolicy"
	"github.com/MustardSeedNetworks/niac-go/internal/fabric"
)

func TestParseAttachmentPolicies(t *testing.T) {
	got, err := attachpolicy.ParseAll([]string{
		"eth0=access:200",
		"eth1=direct",
		"eth2=trunk:200,201,299",
	})
	if err != nil {
		t.Fatalf("attachpolicy.ParseAll() error = %v", err)
	}
	want := []attachpolicy.Policy{
		{Interface: "eth0", Mode: fabric.ModeAccess, AccessVLAN: 200},
		{Interface: "eth1", Mode: fabric.ModeDirect},
		{Interface: "eth2", Mode: fabric.ModeTrunk, AllowedVLANs: []uint16{200, 201, 299}},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("attachpolicy.ParseAll() = %#v, want %#v", got, want)
	}
}

func TestParseAttachmentPoliciesRejectsInvalidValues(t *testing.T) {
	tests := []struct {
		name  string
		value []string
		match string
	}{
		{name: "missing interface", value: []string{"=direct"}, match: "expected INTERFACE"},
		{name: "missing mode", value: []string{"eth0="}, match: "expected INTERFACE"},
		{name: "direct VLAN", value: []string{"eth0=direct:200"}, match: "expected INTERFACE"},
		{name: "zero VLAN", value: []string{"eth0=access:0"}, match: "between 1 and 4094"},
		{name: "reserved VLAN", value: []string{"eth0=access:4095"}, match: "between 1 and 4094"},
		{name: "non-numeric VLAN", value: []string{"eth0=access:lab"}, match: "between 1 and 4094"},
		{name: "empty trunk", value: []string{"eth0=trunk:"}, match: "at least one VLAN"},
		{name: "trunk zero VLAN", value: []string{"eth0=trunk:0"}, match: "between 1 and 4094"},
		{
			name:  "trunk reserved VLAN",
			value: []string{"eth0=trunk:4095"},
			match: "between 1 and 4094",
		},
		{
			name:  "trunk non-numeric VLAN",
			value: []string{"eth0=trunk:lab"},
			match: "between 1 and 4094",
		},
		{
			name:  "trunk duplicate VLAN",
			value: []string{"eth0=trunk:200,200"},
			match: "duplicate VLAN",
		},
		{
			name:  "trunk unordered VLANs",
			value: []string{"eth0=trunk:201,200"},
			match: "ascending order",
		},
		{name: "whitespace", value: []string{" eth0=direct"}, match: "expected INTERFACE"},
		{name: "duplicate", value: []string{"eth0=direct", "eth0=direct"}, match: "duplicate"},
		{
			// Still rejected, but as a direct-exclusivity violation rather than a
			// bare duplicate interface: one interface may now hold a trunk and an
			// access policy together (#1463), just never direct with anything.
			name:  "direct combined with trunk",
			value: []string{"eth0=direct", "eth0=trunk:200"},
			match: "cannot combine direct",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := attachpolicy.ParseAll(tt.value)
			if err == nil || !strings.Contains(err.Error(), tt.match) {
				t.Fatalf("attachpolicy.ParseAll() error = %v, want containing %q", err, tt.match)
			}
		})
	}
}

// TestParseAttachmentPoliciesAllowsNativeAlongsideTrunk guards #1463.
//
// #1426 taught the session registry that one interface carries N tagged
// sessions plus at most one native session — a trunk port with a native VLAN.
// The policy parser still rejected any repeat of an interface, so the operator
// could not approve both bindings and the capability was unreachable: CT304
// runs six tagged scenarios and refuses a seventh in access mode on policy
// grounds, not registry grounds.
func TestParseAttachmentPoliciesAllowsNativeAlongsideTrunk(t *testing.T) {
	got, err := attachpolicy.ParseAll([]string{
		"eth0=trunk:200,201",
		"eth0=access:210",
	})
	if err != nil {
		t.Fatalf("native alongside trunk on one interface = %v, want accepted", err)
	}

	want := []attachpolicy.Policy{
		{Interface: "eth0", Mode: fabric.ModeTrunk, AllowedVLANs: []uint16{200, 201}},
		{Interface: "eth0", Mode: fabric.ModeAccess, AccessVLAN: 210},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("attachpolicy.ParseAll() = %#v, want %#v", got, want)
	}
}

// Both bindings must actually be approved by the resulting policy set.
func TestNativeAndTaggedBindingsAreBothApproved(t *testing.T) {
	policies, err := attachpolicy.ParseAll([]string{"eth0=trunk:200,201", "eth0=access:210"})
	if err != nil {
		t.Fatalf("attachpolicy.ParseAll() error = %v", err)
	}

	approves := func(binding fabric.Binding) bool {
		for _, policy := range policies {
			if policy.Approves(binding) {
				return true
			}
		}
		return false
	}

	tagged := fabric.Binding{Interface: "eth0", Mode: fabric.ModeTrunk, AccessVLAN: 201}
	native := fabric.Binding{Interface: "eth0", Mode: fabric.ModeAccess, AccessVLAN: 210}
	if !approves(tagged) {
		t.Error("tagged binding on eth0 VLAN 201 was not approved")
	}
	if !approves(native) {
		t.Error("native binding on eth0 access VLAN 210 was not approved")
	}
}

// Direct means unisolated ownership of the whole interface, so it still cannot
// share one with anything else.
func TestParseAttachmentPoliciesKeepsDirectExclusive(t *testing.T) {
	for _, values := range [][]string{
		{"eth0=direct", "eth0=trunk:200"},
		{"eth0=trunk:200", "eth0=direct"},
		{"eth0=direct", "eth0=access:210"},
	} {
		if _, err := attachpolicy.ParseAll(values); err == nil {
			t.Errorf("attachpolicy.ParseAll(%v) = nil error, want direct to stay exclusive", values)
		}
	}
}

// A repeated mode on one interface is still an operator mistake.
func TestParseAttachmentPoliciesRejectsRepeatedMode(t *testing.T) {
	if _, err := attachpolicy.ParseAll([]string{"eth0=access:210", "eth0=access:211"}); err == nil {
		t.Error("two access policies on one interface = nil error, want a duplicate error")
	}
	if _, err := attachpolicy.ParseAll([]string{"eth0=trunk:200", "eth0=trunk:201"}); err == nil {
		t.Error("two trunk policies on one interface = nil error, want a duplicate error")
	}
}

func TestFormatRoundTripsParse(t *testing.T) {
	for _, value := range []string{"eth0=direct", "eth0=access:200", "eth0=trunk:200,201,299"} {
		policy, err := attachpolicy.Parse(value)
		if err != nil {
			t.Fatalf("attachpolicy.Parse(%q) error = %v", value, err)
		}
		if got := attachpolicy.Format(policy); got != value {
			t.Errorf("attachpolicy.Format(attachpolicy.Parse(%q)) = %q", value, got)
		}
	}
}

func TestMerge(t *testing.T) {
	parse := func(values ...string) []attachpolicy.Policy {
		t.Helper()
		policies, err := attachpolicy.ParseAll(values)
		if err != nil {
			t.Fatalf("attachpolicy.ParseAll(%v) error = %v", values, err)
		}
		return policies
	}
	tests := []struct {
		name        string
		flags, file []string
		want        []string
		conflict    string
	}{
		{name: "file only", file: []string{"eth0=direct"}, want: []string{"eth0=direct"}},
		{name: "flags only", flags: []string{"eth0=access:200"}, want: []string{"eth0=access:200"}},
		{
			name:  "disjoint, flags first",
			flags: []string{"eth0=access:200"}, file: []string{"eth1=direct"},
			want: []string{"eth0=access:200", "eth1=direct"},
		},
		{
			name:  "same policy in both counts once",
			flags: []string{"eth0=trunk:200,201"}, file: []string{"eth0=trunk:200,201", "eth1=direct"},
			want: []string{"eth0=trunk:200,201", "eth1=direct"},
		},
		{
			name:  "native beside trunk across sources",
			flags: []string{"eth0=trunk:200,201"}, file: []string{"eth0=access:210"},
			want: []string{"eth0=trunk:200,201", "eth0=access:210"},
		},
		{
			name:  "direct against flag access",
			flags: []string{"eth0=access:200"}, file: []string{"eth0=direct"},
			conflict: "cannot combine direct",
		},
		{
			name:  "different VLAN, same mode",
			flags: []string{"eth0=access:200"}, file: []string{"eth0=access:201"},
			conflict: "duplicate access policy",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			merged := attachpolicy.Merge(parse(tt.flags...), parse(tt.file...))
			err := attachpolicy.Validate(merged)
			if tt.conflict != "" {
				if err == nil || !strings.Contains(err.Error(), tt.conflict) {
					t.Fatalf("attachpolicy.Validate(merged) error = %v, want containing %q", err, tt.conflict)
				}
				return
			}
			if err != nil {
				t.Fatalf("attachpolicy.Validate(merged) error = %v", err)
			}
			got := make([]string, len(merged))
			for i, policy := range merged {
				got[i] = attachpolicy.Format(policy)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("merged = %v, want %v", got, tt.want)
			}
		})
	}
}
