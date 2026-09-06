package scenario

import (
	"github.com/MustardSeedNetworks/niac-go/internal/converter"
	"github.com/MustardSeedNetworks/niac-go/internal/safeconv"
)

func identitySuffix(site *Site, role string, index int) uint32 {
	octet := 0
	if site != nil {
		octet = site.Octet
	}
	return safeconv.Uint32(octet<<16 | roleCode(role)<<8 | index)
}

func roleCode(role string) int {
	switch role {
	case "lab":
		return roleLabCode
	case "wan":
		return roleWANCode
	case "firewall":
		return roleFirewallCode
	case "core":
		return roleCoreCode
	case "distribution":
		return roleDistributionCode
	case "access":
		return roleAccessCode
	case "server-switch":
		return roleServerSwitchCode
	case "ap":
		return roleAccessPointCode
	case "workstation":
		return roleWorkstationCode
	case "server":
		return roleServerCode
	case "controller":
		return roleControllerCode
	default:
		if isWiredEndpointRole(role) || isMEDEndpointRole(role) {
			return roleWorkstationCode
		}
		panic("unknown scenario role: " + role)
	}
}

func route(destination, via, nextHop string) converter.Route {
	return converter.Route{Destination: destination, Via: via, NextHop: nextHop}
}
