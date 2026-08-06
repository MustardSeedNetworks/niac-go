package scenario

const (
	hospitalAccessSwitches        = 6
	hospitalAccessPointsPerAccess = 5
	hospitalWorkstationsPerAccess = 3
	// A warehouse covers a large open floor from a few wiring closets, so it
	// has far fewer access switches than a hospital or a plant and fans a
	// dense set of long-range radios off each one. Its endpoints are mostly
	// wireless handhelds rather than wired. This is what makes its map read
	// differently from the other single-site verticals.
	warehouseAccessSwitches            = 3
	warehouseAccessPointsPerAccess     = maxAccessPointsPerAccess
	warehouseWorkstationsPerAccess     = 2
	campusAccessSwitches               = 4
	campusAccessPointsPerAccess        = 2
	campusWorkstationsPerAccess        = 2
	retailAccessSwitches               = 4
	retailAccessPointsPerAccess        = 3
	retailWorkstationsPerAccess        = 3
	manufacturingAccessSwitches        = 6
	manufacturingAccessPointsPerAccess = 5
	manufacturingWorkstationsPerAccess = 2
	providerAccessSwitches             = 4
	providerAccessPointsPerAccess      = 0
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
