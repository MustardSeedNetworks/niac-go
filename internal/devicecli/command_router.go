package devicecli

import (
	"net/netip"
	"strconv"

	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
)

func (s *Session) executeRouter(fields []string) Response {
	switch fields[0] {
	case "end":
		if len(fields) != 1 {
			return Response{Output: invalidInput}
		}
		s.mode = ModePrivileged
	case commandExit:
		if len(fields) != 1 {
			return Response{Output: invalidInput}
		}
		s.mode = ModeGlobalConfig
	case "network":
		if len(fields) != 5 || fields[3] != "area" || !validIPv4(fields[1]) || !validIPv4(fields[2]) {
			return Response{Output: invalidInput}
		}
		area, ok := normalizeOSPFArea(fields[4])
		if !ok {
			return Response{Output: invalidInput}
		}
		s.state.AddRouterNetwork(s.routerProtocol, s.routerProcessID, devicestate.RouterNetwork{
			Address: fields[1], Wildcard: fields[2], Area: area,
		})
	default:
		return Response{Output: invalidInput}
	}
	return Response{}
}

func normalizeOSPFArea(value string) (string, bool) {
	if address, err := netip.ParseAddr(value); err == nil && address.Is4() {
		return address.String(), true
	}
	area, err := strconv.ParseUint(value, 10, 32)
	if err != nil {
		return "", false
	}
	return strconv.FormatUint(area, 10), true
}

func validIPv4(value string) bool {
	address, err := netip.ParseAddr(value)
	return err == nil && address.Is4()
}
