package scenario

import "testing"

func TestEndpointMixAllocation(t *testing.T) {
	for _, profile := range []string{"", "enterprise", "hospital", "warehouse", "manufacturing", "retail", "service-provider"} {
		for slots := 0; slots <= maxSiteWorkstations; slots++ {
			checkEndpointAllocation(t, profile, slots)
		}
	}
}

// checkEndpointAllocation holds the mix to its contract at one site size:
// every slot filled, every floor met once there is room for all of them, a
// signature device never more than its floor, and the layout carrying exactly
// what was allocated.
func checkEndpointAllocation(t *testing.T, profile string, slots int) {
	t.Helper()

	mix := endpointMix(profile)
	counts := allocateEndpoints(mix, slots)
	laidOut := map[string]int{}
	for _, kind := range siteEndpointKinds(profile, slots) {
		laidOut[kind.role]++
	}
	allocated := map[string]int{}
	total, floors := 0, 0
	for index, share := range mix {
		total += counts[index]
		floors += share.floor(slots)
		allocated[share.kind.role] += counts[index]
	}
	if total != slots {
		t.Errorf("%q at %d slots: allocated %d", profile, slots, total)
	}
	for index, share := range mix {
		if slots >= floors && counts[index] < share.floor(slots) {
			t.Errorf("%q at %d slots: %s = %d, below its floor %d",
				profile, slots, share.kind.role, counts[index], share.floor(slots))
		}
		if share.weight == 0 && counts[index] > share.floor(slots) {
			t.Errorf("%q at %d slots: signature %s grew to %d", profile, slots, share.kind.role, counts[index])
		}
	}
	for role, count := range allocated {
		if laidOut[role] != count {
			t.Errorf("%q at %d slots: laid out %d %s, allocated %d", profile, slots, laidOut[role], role, count)
		}
	}
}

// A kind's slots spread across the site rather than filling its first access
// switches, so every closet shows the vertical's mix.
func TestEndpointMixInterleaves(t *testing.T) {
	kinds := siteEndpointKinds("hospital", 18)
	run, longest := 1, 1
	for index := 1; index < len(kinds); index++ {
		if kinds[index].role == kinds[index-1].role {
			run++
			longest = max(longest, run)
		} else {
			run = 1
		}
	}
	if longest > 2 {
		t.Errorf("longest run of one kind = %d slots, want at most 2: %v", longest, roles(kinds))
	}
}

func roles(kinds []endpointKind) []string {
	out := make([]string, 0, len(kinds))
	for _, kind := range kinds {
		out = append(out, kind.role)
	}
	return out
}

// The closet tier grows with the site: a rack PDU per 16 wired endpoints and a
// door controller per 32. A site too small for one keeps its own devices, so
// the six-slot warehouse still has three handhelds, while a 64-slot enterprise
// site carries four PDUs and two controllers.
func TestClosetTierScalesWithTheSite(t *testing.T) {
	for _, tc := range []struct {
		profile string
		slots   int
		want    map[string]int
	}{
		{profile: "warehouse", slots: 6, want: map[string]int{"pdu": 0, "badge-controller": 0, "rugged-handheld": 3}},
		{profile: "hospital", slots: 18, want: map[string]int{"pdu": 1, "badge-controller": 0, "mr-system": 1}},
		{profile: "enterprise", slots: 64, want: map[string]int{"pdu": 4, "badge-controller": 2, "ups": 1}},
	} {
		got := map[string]int{}
		for _, kind := range siteEndpointKinds(tc.profile, tc.slots) {
			got[kind.role]++
		}
		for role, want := range tc.want {
			if got[role] != want {
				t.Errorf("%s at %d slots: %s = %d, want %d", tc.profile, tc.slots, role, got[role], want)
			}
		}
	}
}
