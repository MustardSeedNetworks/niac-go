package scenario

import "fmt"

const (
	unixTTL           = 64
	unixTCPWindowSize = 65535
)

type endpointKind struct {
	role   string
	prefix string
	osType string
	ttl    uint8
	// personalComputer marks a machine someone sits in front of. Those ship
	// without an SNMP agent, so a discovery tool files them as hosts; the
	// clinical, industrial and retail appliances alongside them are
	// SNMP-managed in the real world and are meant to appear that way.
	personalComputer bool
	windowSize       uint16
}

func wiredEndpointKind(request Request, accessIndex, slot int) endpointKind {
	kinds := siteEndpointKinds(
		request.EndpointProfile,
		request.Counts.AccessSwitches*request.Counts.WorkstationsPerAccess,
	)
	return kinds[(accessIndex-1)*request.Counts.WorkstationsPerAccess+slot-1]
}

// wiredEndpointName labels one endpoint.
//
// Appliances get the readable asset form. Personal computers get a compact one,
// because a NetBIOS name is capped at 15 characters and - now that they carry no
// SNMP agent - NetBIOS is where a discovery tool reads a Windows machine's name
// from. A longer name would arrive truncated and disagree with authored truth.
// Real fleets name Windows hosts tersely for exactly this reason.
//
// The compact tail packs building, floor and slot with no separators. Read it
// from the right: two digits of slot, one of floor (always 1-4, see location),
// and whatever precedes them is the building.
func wiredEndpointName(request Request, site Site, accessIndex, slot int) string {
	kind := wiredEndpointKind(request, accessIndex, slot)
	building, floor := location(accessIndex)
	if kind.personalComputer {
		return fmt.Sprintf("%s-%s-%d%d%02d", site.Code, kind.prefix, building, floor, slot)
	}

	return fmt.Sprintf("%s-%s-B%02d-F%02d-%02d", site.Code, kind.prefix, building, floor, slot)
}

func isWiredEndpointRole(role string) bool {
	for _, profile := range []string{"enterprise", "hospital", "warehouse", "manufacturing", "retail", "service-provider"} {
		for _, share := range endpointMix(profile) {
			if share.kind.role == role {
				return true
			}
		}
	}
	return false
}

// isMEDEndpointRole reports whether role is one of the appended LLDP-MED
// endpoints. They are deliberately not in endpointMix -- being there would
// take wired slots from the vertical's own devices -- so the identity
// map has to know about them separately.
func isMEDEndpointRole(role string) bool {
	return role == "voip-phone" || role == "ip-camera"
}
