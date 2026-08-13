package cliclient

import (
	"context"
	"net/url"
)

// StackStats is the daemon's per-protocol packet accounting, as served by
// /api/v1/stats. These are the counters `niac monitor` reports; it used to
// print zeros for all of them because the transport it read from carried no
// protocol breakdown.
type StackStats struct {
	PacketsSent     uint64 `json:"packetsSent"`
	PacketsReceived uint64 `json:"packetsReceived"`
	ARPRequests     uint64 `json:"arpRequests"`
	ARPReplies      uint64 `json:"arpReplies"`
	ICMPRequests    uint64 `json:"icmpRequests"`
	ICMPReplies     uint64 `json:"icmpReplies"`
	DNSQueries      uint64 `json:"dnsQueries"`
	DHCPRequests    uint64 `json:"dhcpRequests"`
	SNMPQueries     uint64 `json:"snmpQueries"`
	Errors          uint64 `json:"errors"`
}

// Stats is one sample of the daemon's runtime counters.
type Stats struct {
	Interface   string     `json:"interface"`
	Version     string     `json:"version"`
	DeviceCount int        `json:"deviceCount"`
	Goroutines  int        `json:"goroutines"`
	Stack       StackStats `json:"stack"`
}

// ARP is request and reply traffic together, which is what a single "ARP"
// column in the monitor output means.
func (s StackStats) ARP() uint64 { return s.ARPRequests + s.ARPReplies }

// ICMP is request and reply traffic together, for the same reason as ARP.
func (s StackStats) ICMP() uint64 { return s.ICMPRequests + s.ICMPReplies }

// Stats reads one sample of the counters for whichever session is selected.
//
// Several scenarios run at once, each with its own stack and its own counters,
// so "the stats" is ambiguous unless a session is named: a caller watching the
// selected session sees zeros while another session carries the traffic. Prefer
// SessionStats.
func (c *Client) Stats(ctx context.Context) (*Stats, error) {
	var stats Stats
	if err := c.get(ctx, "/api/v1/stats", &stats); err != nil {
		return nil, err
	}

	return &stats, nil
}

// SessionStats reads one sample of a named session's counters.
func (c *Client) SessionStats(ctx context.Context, session string) (*Stats, error) {
	var stats Stats
	if err := c.get(ctx, "/api/v1/sessions/"+url.PathEscape(session)+"/stats", &stats); err != nil {
		return nil, err
	}

	return &stats, nil
}
