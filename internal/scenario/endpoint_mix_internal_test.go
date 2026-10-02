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

// The common tier keeps one NAS per site and grows the rest with it: a rack
// PDU per 16 wired endpoints, a conference room per 24 and a door controller
// per 32. A site too small for the closet devices keeps its own, so a six-slot
// warehouse gives one handheld's slot to its NAS and nothing else, while a
// 64-slot enterprise site carries four PDUs, two rooms and two controllers.
func TestCommonTierScalesWithTheSite(t *testing.T) {
	for _, tc := range []struct {
		profile string
		slots   int
		want    map[string]int
	}{
		{profile: "warehouse", slots: 6, want: map[string]int{
			"nas": 1, "pdu": 0, "conference-room": 0, "badge-controller": 0, "rugged-handheld": 2,
		}},
		{profile: "hospital", slots: 18, want: map[string]int{
			"nas": 1, "pdu": 1, "conference-room": 0, "badge-controller": 0, "mr-system": 1,
		}},
		{profile: "enterprise", slots: 64, want: map[string]int{
			"nas": 1, "pdu": 4, "conference-room": 2, "badge-controller": 2, "ups": 1,
		}},
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

// A service provider's POP is where its access network terminates: the OLT
// that lights the PON, and the reference ONT and CPE router its technicians
// turn subscribers up against. Each is an appliance, so it answers SNMP as
// itself, and each is a signature device, so a POP carries one of each
// however many NOC workstations it has.
func TestServiceProviderPOPCarriesItsAccessTier(t *testing.T) {
	accessTier := map[string]bool{"olt": true, "ont": true, "cpe-router": true}
	for _, slots := range []int{8, 64} {
		got := map[string]int{}
		for _, kind := range siteEndpointKinds("service-provider", slots) {
			got[kind.role]++
			if kind.personalComputer && accessTier[kind.role] {
				t.Errorf("%d slots: %s is laid out as a personal computer, so it answers no SNMP", slots, kind.role)
			}
		}
		for _, role := range []string{"olt", "ont", "cpe-router"} {
			if got[role] != 1 {
				t.Errorf("%d slots: %s = %d, want 1", slots, role, got[role])
			}
		}
		if got["noc-workstation"] == 0 {
			t.Errorf("%d slots: no NOC workstation", slots)
		}
	}
}

// The client tier is more than one model per vertical. A hospital's shared
// terminals are thin clients beside its nurse stations, a store's back office
// has a desktop beside its tills, and a POP's staff carry laptops and sit at
// office desktops as well as NOC consoles. With the NOC console the only
// weighted kind, a 64-slot POP was nine parts NOC workstation to every other
// device put together.
func TestClientTierCarriesMoreThanOneModel(t *testing.T) {
	for _, tc := range []struct {
		profile string
		slots   int
		want    []string
	}{
		{profile: "hospital", slots: 64, want: []string{"nurse-station", "thin-client"}},
		{profile: "retail", slots: 64, want: []string{"point-of-sale", "workstation"}},
		{profile: "service-provider", slots: 64, want: []string{"noc-workstation", "windows-laptop", "workstation"}},
	} {
		got := map[string]int{}
		for _, kind := range siteEndpointKinds(tc.profile, tc.slots) {
			got[kind.role]++
		}
		for _, role := range tc.want {
			if got[role] == 0 {
				t.Errorf("%s at %d slots: no %s; mix %v", tc.profile, tc.slots, role, got)
			}
		}
		if noc := got["noc-workstation"]; noc*2 > tc.slots {
			t.Errorf("%s at %d slots: %d NOC workstations, more than half the site", tc.profile, tc.slots, noc)
		}
	}
}
