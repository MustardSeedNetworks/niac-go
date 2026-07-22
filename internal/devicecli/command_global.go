package devicecli

import (
	"net/netip"
	"strconv"

	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
)

func (s *Session) executeGlobal(fields []string) Response {
	if len(fields) == 5 && fields[0] == "ip" && fields[1] == "route" {
		return s.setStaticRoute(fields)
	}
	switch fields[0] {
	case "end":
		if len(fields) != 1 {
			return Response{Output: invalidInput}
		}
		s.mode = ModePrivileged
		return Response{}
	case commandExit:
		if len(fields) != 1 {
			return Response{Output: invalidInput}
		}
		s.mode = ModePrivileged
		return Response{}
	case "hostname":
		return s.setHostname(fields)
	case "interface":
		return s.selectInterface(fields)
	case "vlan":
		return s.selectVLAN(fields)
	case "router":
		return s.selectRouter(fields)
	default:
		return Response{Output: invalidInput}
	}
}

func (s *Session) setStaticRoute(fields []string) Response {
	destination, destinationErr := netip.ParsePrefix(fields[2])
	nextHop, nextHopErr := netip.ParseAddr(fields[3])
	if destinationErr != nil || !destination.Addr().Is4() || nextHopErr != nil || !nextHop.Is4() ||
		!hasInterface(s.state.Snapshot().Network.Interfaces, fields[4]) {
		return Response{Output: "% Invalid static route"}
	}
	s.state.UpsertRoute(devicestate.Route{
		Destination: destination, Via: fields[4], NextHop: nextHop,
	})
	return Response{}
}

func (s *Session) selectVLAN(fields []string) Response {
	if len(fields) != singleArgumentCommandFields {
		return Response{Output: invalidInput}
	}
	id, err := strconv.Atoi(fields[1])
	if err != nil || id < 1 || id > 4094 {
		return Response{Output: "% Invalid VLAN"}
	}
	s.state.EnsureVLAN(id)
	s.vlanID = id
	s.mode = ModeVLANConfig
	return Response{}
}

func (s *Session) selectRouter(fields []string) Response {
	if len(fields) != 3 || fields[1] != "ospf" {
		return Response{Output: invalidInput}
	}
	processID, err := strconv.Atoi(fields[2])
	if err != nil || processID < 1 {
		return Response{Output: "% Invalid process ID"}
	}
	canonicalID := strconv.Itoa(processID)
	s.state.EnsureRouter(fields[1], canonicalID)
	s.routerProtocol = fields[1]
	s.routerProcessID = canonicalID
	s.mode = ModeRouterConfig
	return Response{}
}

func (s *Session) setHostname(fields []string) Response {
	if len(fields) != 2 || !validHostname(fields[1]) {
		return Response{Output: "% Invalid hostname"}
	}
	err := s.state.UpdateIdentity(func(identity devicestate.Identity) (devicestate.Identity, error) {
		identity.Hostname = fields[1]
		return identity, nil
	})
	if err != nil {
		return Response{Output: "% Configuration failed"}
	}
	return Response{}
}

func (s *Session) selectInterface(fields []string) Response {
	if len(fields) != 2 || !hasInterface(s.state.Snapshot().Network.Interfaces, fields[1]) {
		return Response{Output: "% Interface not found"}
	}
	s.interfaceName = fields[1]
	s.mode = ModeInterfaceConfig
	return Response{}
}

func hasInterface(interfaces []devicestate.Interface, name string) bool {
	for _, iface := range interfaces {
		if iface.Name == name {
			return true
		}
	}
	return false
}

func validHostname(hostname string) bool {
	if len(hostname) == 0 || len(hostname) > 63 || hostname[0] == '-' || hostname[len(hostname)-1] == '-' {
		return false
	}
	for _, char := range hostname {
		if !isHostnameChar(char) {
			return false
		}
	}
	return true
}

func isHostnameChar(char rune) bool {
	return char >= 'a' && char <= 'z' || char >= 'A' && char <= 'Z' || char >= '0' && char <= '9' || char == '-'
}
