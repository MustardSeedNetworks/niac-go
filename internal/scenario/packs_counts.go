package scenario

const (
	hospitalAccessSwitches        = 8
	hospitalAccessPointsPerAccess = 3
	hospitalWorkstationsPerAccess = 9
	// A warehouse covers a large open floor, so it fans more long-range
	// radios off each closet than the other packs do and its endpoints are
	// mostly rugged handhelds. That density is what makes its map read
	// differently from the other single-site verticals at the same size.
	warehouseAccessSwitches            = 15
	warehouseAccessPointsPerAccess     = 5
	warehouseWorkstationsPerAccess     = 10
	campusAccessSwitches               = 4
	campusAccessPointsPerAccess        = 2
	campusWorkstationsPerAccess        = 2
	retailAccessSwitches               = 4
	retailAccessPointsPerAccess        = 3
	retailWorkstationsPerAccess        = 3
	manufacturingAccessSwitches        = 18
	manufacturingAccessPointsPerAccess = 3
	manufacturingWorkstationsPerAccess = 9
	providerAccessSwitches             = 4
	providerAccessPointsPerAccess      = 2
	providerWorkstationsPerAccess      = 2
)

func packCounts(access, accessPoints, workstations int) Counts {
	counts := Counts{
		SiteWANRouters: maxRedundantPeers, Firewalls: maxRedundantPeers, CoreSwitches: maxRedundantPeers,
		DistributionSwitches: maxRedundantPeers, AccessSwitches: access, ServerSwitches: maxRedundantPeers,
		AccessPointsPerAccess: accessPoints, WorkstationsPerAccess: workstations,
	}
	if accessPoints > 0 {
		counts.WirelessControllers = maxRedundantPeers
	}
	return counts
}

// campusCounts drops the distribution tier: a wide, shallow campus lands its
// closets straight on a collapsed core, so generating distribution switches
// would put a tier on the map that nobody deployed.
func campusCounts() Counts {
	counts := packCounts(campusAccessSwitches, campusAccessPointsPerAccess, campusWorkstationsPerAccess)
	counts.DistributionSwitches = 0

	return counts
}
