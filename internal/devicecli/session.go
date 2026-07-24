package devicecli

import (
	"strings"

	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
)

const (
	invalidInput                = "% Invalid input detected"
	commandExit                 = "exit"
	singleArgumentCommandFields = 2
)

// NewSession creates an isolated user-mode command session.
func NewSession(state *devicestate.Store, validateRoute RouteValidator) *Session {
	return &Session{state: state, validateRoute: validateRoute, mode: ModeUser}
}

// Mode returns the current command mode.
func (s *Session) Mode() Mode {
	return s.mode
}

// Prompt renders the current hostname and command mode.
func (s *Session) Prompt() string {
	hostname := s.state.Snapshot().Identity.Hostname
	switch s.mode {
	case ModeUser:
		return hostname + ">"
	case ModePrivileged:
		return hostname + "#"
	case ModeGlobalConfig:
		return hostname + "(config)#"
	case ModeInterfaceConfig:
		return hostname + "(config-if)#"
	case ModeVLANConfig:
		return hostname + "(config-vlan)#"
	case ModeRouterConfig:
		return hostname + "(config-router)#"
	default:
		return hostname + ">"
	}
}

// Execute parses and runs one command in the session's current mode.
func (s *Session) Execute(line string) Response {
	trimmed := strings.TrimSpace(line)
	if prefix, found := strings.CutSuffix(trimmed, "?"); found {
		return Response{Output: s.help(strings.TrimSpace(prefix))}
	}
	fields := strings.Fields(line)
	if len(fields) == 0 {
		return Response{}
	}
	switch s.mode {
	case ModeUser:
		return s.executeUser(fields)
	case ModePrivileged:
		return s.executePrivileged(fields)
	case ModeGlobalConfig:
		return s.executeGlobal(fields)
	case ModeInterfaceConfig:
		return s.executeInterface(fields)
	case ModeVLANConfig:
		return s.executeVLAN(fields)
	case ModeRouterConfig:
		return s.executeRouter(fields)
	default:
		return Response{Output: invalidInput}
	}
}
