package api

import (
	"net/http"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
)

// segmentResponse is one multi-VLAN segment (ADR 0008): a VLAN tag and the
// device set isolated on it. A flat (non-segmented) config reports a single
// untagged segment wrapping all devices.
type segmentResponse struct {
	VLANTag  int              `json:"vlanTag"`
	Untagged bool             `json:"untagged,omitempty"`
	Devices  []map[string]any `json:"devices"`
}

// handleSegments enumerates the multi-VLAN segments the current config
// describes, grouping devices by VLAN tag. It reads config, not the stack: the
// runtime segmentTables demux is unexported and Stack.AllDevices() flattens the
// grouping away, so config.NormalizedSegments() is the only accurate,
// already-grouped, cross-package source. A config with no `segments:` still
// reports one untagged segment, so the wire shape is uniform.
// Route: GET /api/v1/segments.
func (s *Server) handleSegments(w http.ResponseWriter, _ *http.Request) {
	cfg := s.currentConfig()
	if cfg == nil {
		s.writeJSON(w, []segmentResponse{})

		return
	}
	s.writeJSON(w, buildSegmentResponses(cfg))
}

// buildSegmentResponses projects a config's normalized segments. Shared with
// the session-scoped segments endpoint so both render segments identically.
func buildSegmentResponses(cfg *config.Config) []segmentResponse {
	segments := cfg.NormalizedSegments()
	out := make([]segmentResponse, 0, len(segments))
	for i := range segments {
		seg := &segments[i]
		devices := make([]map[string]any, 0, len(seg.Devices))
		for j := range seg.Devices {
			devices = append(devices, deviceSummary(&seg.Devices[j]))
		}
		out = append(out, segmentResponse{
			VLANTag:  seg.Tag,
			Untagged: seg.Tag == config.UntaggedTag,
			Devices:  devices,
		})
	}

	return out
}
