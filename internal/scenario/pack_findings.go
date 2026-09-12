package scenario

// Every pack but the stress bed ships exactly one finding, chosen so a
// cold-start demo has something to find and so each one has a seed rule that
// sees it (the cross-product rule, 2026-09-06). A map that renders uniformly
// healthy proves nothing about a discovery tool.
//
// The type names match the authored fault catalog rather than restating it;
// a name that drifts fails validation at generation time.
const (
	faultHighUtilization = "high_utilization"
	faultPacketDiscards  = "packet_discards"
	faultFCSErrors       = "fcs_errors"
	faultPoELoss         = "poe_loss"
	faultDHCPNoOffer     = "dhcp_no_offer"
	faultLatency         = "latency"
)

// dockAccessPointPoELoss is the warehouse finding: the switch port feeding a
// dock radio stops supplying power, so the AP is simply gone rather than
// reachable-but-degraded. PoE has no magnitude, hence no value.
func dockAccessPointPoELoss() []PackFault {
	return []PackFault{{
		Device: "FUL-ACC-SW01", Interface: "TenGigabitEthernet1/0/1",
		Type: faultPoELoss,
	}}
}

// ringSegmentFCSErrors is the manufacturing finding: a plant floor ring segment
// corrupting frames, which is what a bad run of cable or a failing SFP looks
// like to a collector.
func ringSegmentFCSErrors() []PackFault {
	const errorRate = 12

	return []PackFault{{
		Device: "PLT-ACC-SW01", Interface: "HundredGigabitEthernet1/0/49",
		Type: faultFCSErrors, Value: faultValue(errorRate),
	}}
}

// storeDHCPNoOffer is the retail finding: the store's own DHCP server stops
// offering, so tills that renew keep working and anything new gets nothing.
// The fault is device-scoped because a service outage has no interface.
func storeDHCPNoOffer() []PackFault {
	const refusalRate = 100

	return []PackFault{{
		Device: "STR-DHCP01",
		Type:   faultDHCPNoOffer, Value: faultValue(refusalRate),
	}}
}

// closetUplinkDiscards is the campus finding: a wiring closet uplink dropping
// frames. Discards rather than errors, because the wire is fine and the queue
// is not.
func closetUplinkDiscards() []PackFault {
	const discardRate = 9

	return []PackFault{{
		Device: "NTH-ACC-SW01", Interface: "HundredGigabitEthernet1/0/49",
		Type: faultPacketDiscards, Value: faultValue(discardRate),
	}}
}

// popUplinkLatency is the service-provider finding: one POP answering slowly.
// Latency is device-scoped and its value is milliseconds, not a percentage.
func popUplinkLatency() []PackFault {
	const delayMilliseconds = 180

	return []PackFault{{
		Device: "NYC-WAN-R01",
		Type:   faultLatency, Value: faultValue(delayMilliseconds),
	}}
}
