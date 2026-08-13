package cliclient

import (
	"context"
	"time"

	"github.com/MustardSeedNetworks/niac-go/internal/topology"
)

// Neighbor is one discovery adjacency the daemon has recorded, as served by
// /api/v1/neighbors. The shape matches what the stack records, so a caller sees
// exactly what the simulation believes.
type Neighbor struct {
	Protocol          string    `json:"protocol"`
	LocalDevice       string    `json:"local_device"`
	RemoteDevice      string    `json:"remote_device"`
	RemotePort        string    `json:"remote_port"`
	RemoteChassisID   string    `json:"remote_chassis_id"`
	Description       string    `json:"description"`
	Capabilities      []string  `json:"capabilities"`
	ManagementAddress string    `json:"management_address"`
	LastSeen          time.Time `json:"last_seen"`
	ExpireAt          time.Time `json:"expire_at"`
}

// Neighbors reads the discovery adjacencies of the selected session.
func (c *Client) Neighbors(ctx context.Context) ([]Neighbor, error) {
	var neighbors []Neighbor
	if err := c.get(ctx, "/api/v1/neighbors", &neighbors); err != nil {
		return nil, err
	}

	return neighbors, nil
}

// Topology reads the node and link graph of the selected session, in the same
// shape the simulation records it, so callers keep the graph's own exporters
// rather than describing it a second time here.
func (c *Client) Topology(ctx context.Context) (*topology.Graph, error) {
	var graph topology.Graph
	if err := c.get(ctx, "/api/v1/topology", &graph); err != nil {
		return nil, err
	}

	return &graph, nil
}
