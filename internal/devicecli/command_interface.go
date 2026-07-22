package devicecli

import (
	"net/netip"
	"strings"

	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
)

func (s *Session) executeInterface(fields []string) Response {
	command := strings.Join(fields, " ")
	if fields[0] == "description" && len(fields) >= singleArgumentCommandFields {
		return s.updateInterface(func(iface devicestate.Interface) devicestate.Interface {
			iface.Description = strings.Join(fields[1:], " ")
			return iface
		})
	}
	if len(fields) == 3 && fields[0] == "ip" && fields[1] == "address" {
		address, err := netip.ParsePrefix(fields[2])
		if err != nil || !address.Addr().Is4() {
			return Response{Output: "% Invalid IPv4 prefix"}
		}
		return s.updateInterface(func(iface devicestate.Interface) devicestate.Interface {
			iface.Address = address
			return iface
		})
	}
	if command == "no ip address" {
		return s.updateInterface(func(iface devicestate.Interface) devicestate.Interface {
			iface.Address = netip.Prefix{}
			return iface
		})
	}
	switch command {
	case "end":
		s.mode = ModePrivileged
	case commandExit:
		s.mode = ModeGlobalConfig
	case "shutdown":
		return s.setInterfaceStatus(false)
	case "no shutdown":
		return s.setInterfaceStatus(true)
	default:
		return Response{Output: invalidInput}
	}
	return Response{}
}

func (s *Session) setInterfaceStatus(up bool) Response {
	return s.updateInterface(func(iface devicestate.Interface) devicestate.Interface {
		iface.AdminUp = up
		iface.OperUp = up && iface.CarrierUp
		return iface
	})
}

func (s *Session) updateInterface(update func(devicestate.Interface) devicestate.Interface) Response {
	err := s.state.UpdateInterface(s.interfaceName, func(iface devicestate.Interface) (devicestate.Interface, error) {
		return update(iface), nil
	})
	if err != nil {
		return Response{Output: "% Interface update failed"}
	}
	return Response{}
}
