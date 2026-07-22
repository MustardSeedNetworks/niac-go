// Package devicecli implements stateful vendor-like command sessions.
package devicecli

import "github.com/MustardSeedNetworks/niac-go/internal/devicestate"

// Mode identifies the active command hierarchy.
type Mode string

const (
	ModeUser            Mode = "user"
	ModePrivileged      Mode = "privileged"
	ModeGlobalConfig    Mode = "global-config"
	ModeInterfaceConfig Mode = "interface-config"
	ModeVLANConfig      Mode = "vlan-config"
	ModeRouterConfig    Mode = "router-config"
)

// Response is the result of one command execution.
type Response struct {
	Output string
	Close  bool
}

// Session holds command mode and context for one connection.
type Session struct {
	state           *devicestate.Store
	mode            Mode
	interfaceName   string
	vlanID          int
	routerProtocol  string
	routerProcessID string
}
