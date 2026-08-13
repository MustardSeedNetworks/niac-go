package cliclient

import "context"

// Runtime is the selected simulation's state, as served by /api/v1/runtime.
// It carries the uptime and config identity that /api/v1/stats does not.
type Runtime struct {
	Running     bool    `json:"running"`
	Interface   string  `json:"interface"`
	ConfigName  string  `json:"config_name"`
	ConfigPath  string  `json:"config_path"`
	DeviceCount int     `json:"device_count"`
	Uptime      float64 `json:"uptime_seconds"`
	Version     string  `json:"version"`
	PacketsRX   uint64  `json:"packets_received"`
	PacketsTX   uint64  `json:"packets_sent"`
}

// Runtime reads the selected simulation's state.
func (c *Client) Runtime(ctx context.Context) (*Runtime, error) {
	var runtime Runtime
	if err := c.get(ctx, "/api/v1/runtime", &runtime); err != nil {
		return nil, err
	}

	return &runtime, nil
}

// Session is one running scenario as the daemon reports it.
type Session struct {
	SessionID    string `json:"sessionId"`
	Selected     bool   `json:"selected"`
	Running      bool   `json:"running"`
	PhysicalVLAN int    `json:"physicalVlan"`
	DeviceCount  int    `json:"deviceCount"`
}

type simulationState struct {
	Sessions []Session `json:"sessions"`
}

// Sessions lists the scenarios the daemon is running.
func (c *Client) Sessions(ctx context.Context) ([]Session, error) {
	var state simulationState
	if err := c.get(ctx, "/api/v1/simulation", &state); err != nil {
		return nil, err
	}

	return state.Sessions, nil
}
