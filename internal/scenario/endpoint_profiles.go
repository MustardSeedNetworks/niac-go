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

func endpointKinds(profile string) []endpointKind {
	pc := func(role, prefix, osType string) endpointKind {
		kind := endpointKind{
			role: role, prefix: prefix, osType: osType, personalComputer: true,
			ttl: unixTTL, windowSize: unixTCPWindowSize,
		}
		if osType == "windows" {
			kind.ttl, kind.windowSize = windowsTTL, windowsTCPWindowSize
		}

		return kind
	}
	appliance := func(role, prefix string) endpointKind {
		return endpointKind{
			role: role, prefix: prefix, osType: "linux", ttl: unixTTL,
			windowSize: unixTCPWindowSize,
		}
	}

	switch profile {
	case "hospital":
		return []endpointKind{
			pc("nurse-station", "NURSE", "windows"),
			appliance("infusion-pump", "PUMP"),
			appliance("mr-system", "MRI"),
			appliance("philips-patient-monitor", "PHMX850"),
			appliance("ge-patient-monitor", "GEB850"),
		}
	case "warehouse":
		return []endpointKind{
			appliance("rugged-handheld", "SCAN"),
			appliance("barcode-printer", "LABEL"),
		}
	case "manufacturing":
		return []endpointKind{
			appliance("plc", "PLC"),
			appliance("hmi", "HMI"),
			appliance("robot-controller", "ROBOT"),
		}
	case "retail":
		return []endpointKind{
			pc("point-of-sale", "POS", "windows"),
			appliance("receipt-printer", "RCPT"),
			appliance("digital-signage", "SIGN"),
		}
	case "service-provider":
		return []endpointKind{pc("noc-workstation", "NOC", "windows")}
	case "enterprise":
		return []endpointKind{
			pc("workstation", "WS", "windows"),
			pc("windows-laptop", "LAP", "windows"),
			pc("macbook", "MBP", "darwin"),
		}
	default:
		return []endpointKind{pc("workstation", "WS", "windows")}
	}
}

func wiredEndpointKind(request Request, accessIndex, slot int) endpointKind {
	kinds := endpointKinds(request.EndpointProfile)
	index := ((accessIndex-1)*request.Counts.WorkstationsPerAccess + slot - 1) % len(kinds)
	return kinds[index]
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
		for _, kind := range endpointKinds(profile) {
			if kind.role == role {
				return true
			}
		}
	}
	return false
}

// isMEDEndpointRole reports whether role is one of the appended LLDP-MED
// endpoints. They are deliberately not in endpointKinds -- being there would
// shift the wired round-robin and rename existing devices -- so the identity
// map has to know about them separately.
func isMEDEndpointRole(role string) bool {
	return role == "voip-phone" || role == "ip-camera"
}
