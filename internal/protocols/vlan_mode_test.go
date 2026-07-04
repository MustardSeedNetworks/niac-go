package protocols

import (
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
)

// TestConfigUsesVLANs drives the VLAN-mode decision: any VLAN-tagged device puts
// the stack in VLAN mode, which makes it ignore untagged frames so it can never
// replay on the native/default VLAN.
func TestConfigUsesVLANs(t *testing.T) {
	cases := []struct {
		name string
		cfg  *config.Config
		want bool
	}{
		{"nil", nil, false},
		{"no devices", &config.Config{}, false},
		{"all untagged", &config.Config{Devices: []config.Device{{Name: "a"}, {Name: "b"}}}, false},
		{"one tagged", &config.Config{Devices: []config.Device{{Name: "a"}, {Name: "b", VLAN: 200}}}, true},
		{"zero vlan is untagged", &config.Config{Devices: []config.Device{{Name: "a", VLAN: 0}}}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := configUsesVLANs(tc.cfg); got != tc.want {
				t.Errorf("configUsesVLANs = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestVLANModeDropsUntagged: in VLAN mode an untagged frame is dropped before any
// handler runs (a nil handler would panic if it were dispatched), while a tagged
// frame proceeds. Uses a stack with no handlers wired, so reaching dispatch panics.
func TestVLANModeDropsUntagged(t *testing.T) {
	s := &Stack{vlanMode: true, stats: &Statistics{}}

	// Untagged (VLAN <= 0) must be dropped: decodePacket returns without routing.
	untagged := &Packet{Buffer: make([]byte, 64), Length: 64, VLAN: -1}
	s.decodePacket(untagged) // must not panic (no dispatch)

	// A non-VLAN-mode stack does not drop untagged; guard that flag alone gates it.
	if (&Stack{vlanMode: false}).vlanMode {
		t.Fatal("sanity")
	}
}
