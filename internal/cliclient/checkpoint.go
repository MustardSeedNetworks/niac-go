package cliclient

import (
	"context"
	"net/http"
	"net/url"
)

// Checkpoint and fault control, the two mutations an acceptance run needs on
// top of start and stop: save a healthy scenario, inject a fault, assert what
// the consumer sees, restore.

// DeviceFaultRequest arms or clears one device-scoped fault. A zero Value
// clears the fault, which is how the daemon's own surface spells it.
type DeviceFaultRequest struct {
	Device string `json:"device"`
	Type   string `json:"errorType"`
	Value  int    `json:"value"`
}

type checkpointRequest struct {
	Name string `json:"name"`
}

type checkpointSaveResponse struct {
	Devices int `json:"devices"`
}

// SaveCheckpoint captures one session's scenario and reports how many devices
// the checkpoint covers.
func (c *Client) SaveCheckpoint(ctx context.Context, sessionID, name string) (int, error) {
	var response checkpointSaveResponse
	path := sessionCheckpointPath(sessionID, "")
	if err := c.mutate(ctx, http.MethodPost, path, checkpointRequest{Name: name}, &response); err != nil {
		return 0, err
	}

	return response.Devices, nil
}

// RestoreCheckpoint returns one session to a named checkpoint.
func (c *Client) RestoreCheckpoint(ctx context.Context, sessionID, name string) error {
	path := sessionCheckpointPath(sessionID, "/restore")
	return c.mutate(ctx, http.MethodPost, path, checkpointRequest{Name: name}, nil)
}

// SetDeviceFault arms or clears one device-scoped fault on the selected
// simulation.
func (c *Client) SetDeviceFault(ctx context.Context, request DeviceFaultRequest) error {
	return c.mutate(ctx, http.MethodPost, "/api/v1/errors", request, nil)
}

func sessionCheckpointPath(sessionID, suffix string) string {
	return "/api/v1/sessions/" + url.PathEscape(sessionID) + "/checkpoints" + suffix
}
