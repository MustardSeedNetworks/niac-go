package devicecli

import (
	"strings"

	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
)

func (s *Session) executeVLAN(fields []string) Response {
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
	case "name":
		if len(fields) < singleArgumentCommandFields {
			return Response{Output: "% Invalid VLAN name"}
		}
		name := strings.Join(fields[1:], " ")
		if err := s.state.UpdateVLAN(s.vlanID, func(vlan devicestate.VLAN) devicestate.VLAN {
			vlan.Name = name
			return vlan
		}); err != nil {
			return Response{Output: "% Configuration failed"}
		}
	default:
		return Response{Output: invalidInput}
	}
	return Response{}
}
