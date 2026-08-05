package scenario

import "fmt"

const (
	unixTTL           = 64
	unixTCPWindowSize = 65535
)

type endpointKind struct {
	role       string
	prefix     string
	osType     string
	ttl        uint8
	windowSize uint16
}

func endpointKinds(profile string) []endpointKind {
	windows := func(role, prefix string) endpointKind {
		return endpointKind{
			role: role, prefix: prefix, osType: "windows", ttl: windowsTTL,
			windowSize: windowsTCPWindowSize,
		}
	}
	embedded := func(role, prefix string) endpointKind {
		return endpointKind{
			role: role, prefix: prefix, osType: "linux", ttl: unixTTL,
			windowSize: unixTCPWindowSize,
		}
	}

	switch profile {
	case "hospital":
		return []endpointKind{
			windows("nurse-station", "NURSE"),
			embedded("infusion-pump", "PUMP"),
			embedded("mr-system", "MRI"),
			embedded("philips-patient-monitor", "PHMX850"),
			embedded("ge-patient-monitor", "GEB850"),
		}
	case "warehouse":
		return []endpointKind{
			embedded("rugged-handheld", "SCAN"),
			embedded("barcode-printer", "LABEL"),
		}
	case "manufacturing":
		return []endpointKind{
			embedded("plc", "PLC"),
			embedded("hmi", "HMI"),
			embedded("robot-controller", "ROBOT"),
		}
	case "retail":
		return []endpointKind{
			windows("point-of-sale", "POS"),
			embedded("receipt-printer", "RCPT"),
			embedded("digital-signage", "SIGN"),
		}
	case "service-provider":
		return []endpointKind{windows("noc-workstation", "NOC")}
	case "enterprise":
		return []endpointKind{
			windows("workstation", "WS"),
			windows("windows-laptop", "LAP"),
			{
				role:       "macbook",
				prefix:     "MBP",
				osType:     "darwin",
				ttl:        unixTTL,
				windowSize: unixTCPWindowSize,
			},
		}
	default:
		return []endpointKind{windows("workstation", "WS")}
	}
}

func wiredEndpointKind(request Request, accessIndex, slot int) endpointKind {
	kinds := endpointKinds(request.EndpointProfile)
	index := ((accessIndex-1)*request.Counts.WorkstationsPerAccess + slot - 1) % len(kinds)
	return kinds[index]
}

func wiredEndpointName(request Request, site Site, accessIndex, slot int) string {
	kind := wiredEndpointKind(request, accessIndex, slot)
	building, floor := location(accessIndex)
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
