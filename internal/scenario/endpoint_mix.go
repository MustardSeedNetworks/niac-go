package scenario

import "sort"

// endpointShare is one kind's place in a vertical's wired endpoint mix.
//
// atLeast is how many every site gets before anything is shared out: one MRI,
// one UPS, one printer, however large or small the site. weight is the kind's
// share of the slots left after that. A signature device carries atLeast and
// no weight, so it stays at one per site when a site grows; a common device
// carries weight, so it grows with the site. That is the realism doc's "many
// of a few things and one of several others", and it holds at P-PACK-1's
// sizes as well as today's, where a repeating rotation did not.
type endpointShare struct {
	kind    endpointKind
	atLeast int
	weight  int
}

func personalComputer(role, prefix, osType string) endpointKind {
	kind := endpointKind{
		role: role, prefix: prefix, osType: osType, personalComputer: true,
		ttl: unixTTL, windowSize: unixTCPWindowSize,
	}
	if osType == "windows" {
		kind.ttl, kind.windowSize = windowsTTL, windowsTCPWindowSize
	}

	return kind
}

func appliance(role, prefix string) endpointKind {
	return endpointKind{
		role: role, prefix: prefix, osType: "linux", ttl: unixTTL,
		windowSize: unixTCPWindowSize,
	}
}

// Weights are relative within one vertical's mix, so they read as how common a
// kind is next to its neighbours rather than as counts.
const (
	weightRare = 1 + iota
	weightSome
	weightMany
	weightMost
	weightDominant
)

// siteUPS is the common tier's power device, one per site in every vertical,
// and the pack device that answers UPS-MIB.
func siteUPS() endpointShare {
	return endpointShare{kind: appliance("ups", "UPS"), atLeast: 1}
}

func endpointMix(profile string) []endpointShare {
	switch profile {
	case "hospital":
		return []endpointShare{
			{kind: appliance("mr-system", "MRI"), atLeast: 1},
			siteUPS(),
			{kind: appliance("label-printer", "LABEL"), atLeast: 1, weight: weightRare},
			{kind: appliance("infusion-pump", "PUMP"), weight: weightDominant},
			{kind: appliance("philips-patient-monitor", "PHMX850"), weight: weightMany},
			{kind: personalComputer("nurse-station", "NURSE", "windows"), weight: weightMany},
			{kind: appliance("ge-patient-monitor", "GEB850"), weight: weightSome},
		}
	case "warehouse":
		return []endpointShare{
			siteUPS(),
			{kind: appliance("barcode-printer", "LABEL"), atLeast: 1, weight: weightRare},
			{kind: appliance("rugged-handheld", "SCAN"), weight: weightMany},
		}
	case "manufacturing":
		return []endpointShare{
			siteUPS(),
			{kind: appliance("barcode-printer", "LABEL"), atLeast: 1},
			{kind: appliance("robot-controller", "ROBOT"), atLeast: 1, weight: weightRare},
			{kind: appliance("plc", "PLC"), weight: weightMost},
			{kind: appliance("hmi", "HMI"), weight: weightSome},
		}
	case "retail":
		return []endpointShare{
			siteUPS(),
			{kind: personalComputer("point-of-sale", "POS", "windows"), weight: weightSome},
			{kind: appliance("receipt-printer", "RCPT"), weight: weightSome},
			{kind: appliance("digital-signage", "SIGN"), weight: weightRare},
		}
	case "service-provider":
		return []endpointShare{
			siteUPS(),
			{kind: appliance("office-printer", "PRN"), atLeast: 1},
			{kind: personalComputer("noc-workstation", "NOC", "windows"), weight: weightRare},
		}
	case "enterprise":
		return []endpointShare{
			siteUPS(),
			{kind: appliance("office-printer", "PRN"), atLeast: 1, weight: weightRare},
			{kind: personalComputer("workstation", "WS", "windows"), weight: weightDominant},
			{kind: personalComputer("windows-laptop", "LAP", "windows"), weight: weightMany},
			{kind: personalComputer("macbook", "MBP", "darwin"), weight: weightSome},
		}
	default:
		return []endpointShare{
			siteUPS(),
			{kind: appliance("office-printer", "PRN"), atLeast: 1, weight: weightRare},
			{kind: personalComputer("workstation", "WS", "windows"), weight: weightDominant},
		}
	}
}

// siteEndpointKinds lays out one site's wired endpoint slots, in slot order.
func siteEndpointKinds(profile string, slots int) []endpointKind {
	mix := endpointMix(profile)
	counts := allocateEndpoints(mix, slots)

	return interleaveEndpoints(mix, counts, slots)
}

// allocateEndpoints gives each kind its floor, in table order while slots
// last, then shares the rest by weight with the largest-remainder method, so
// the counts always sum to slots and never depend on anything but the table.
func allocateEndpoints(mix []endpointShare, slots int) []int {
	counts := make([]int, len(mix))
	remaining := slots
	totalWeight := 0
	for index, share := range mix {
		counts[index] = min(share.atLeast, remaining)
		remaining -= counts[index]
		totalWeight += share.weight
	}
	if remaining == 0 {
		return counts
	}

	remainders := make([]int, len(mix))
	order := make([]int, len(mix))
	left := remaining
	for index, share := range mix {
		quota := remaining * share.weight
		counts[index] += quota / totalWeight
		left -= quota / totalWeight
		remainders[index] = quota % totalWeight
		order[index] = index
	}
	sort.SliceStable(order, func(a, b int) bool { return remainders[order[a]] > remainders[order[b]] })
	for _, index := range order[:left] {
		counts[index]++
	}

	return counts
}

// interleaveEndpoints spreads the counts across the slots with smooth weighted
// round-robin, so a common kind appears under every access switch rather than
// filling the first few, and a site's one MRI sits among the pumps.
func interleaveEndpoints(mix []endpointShare, counts []int, slots int) []endpointKind {
	kinds := make([]endpointKind, 0, slots)
	current := make([]int, len(mix))
	for range slots {
		best := -1
		for index, count := range counts {
			if count == 0 {
				continue
			}
			current[index] += count
			if best < 0 || current[index] > current[best] {
				best = index
			}
		}
		current[best] -= slots
		kinds = append(kinds, mix[best].kind)
	}

	return kinds
}
