package api

import (
	"net/http"
	"slices"

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

// handleSegments enumerates the VLAN segments the current config describes.
// Route: GET /api/v1/segments.
func (s *Server) handleSegments(w http.ResponseWriter, _ *http.Request) {
	cfg := s.currentConfig()
	if cfg == nil {
		s.writeJSON(w, []segmentResponse{})

		return
	}
	s.writeJSON(w, buildSegmentResponses(cfg))
}

// buildSegmentResponses groups the config's devices by VLAN for display.
//
// An explicit `segments:` block is authoritative — that is the engine's own
// multi-VLAN binding (ADR 0008) and NormalizedSegments() is the right source
// for it. None of the shipped scenario packs use one, though: they carry VLAN
// membership per device instead. Falling straight through to
// NormalizedSegments() therefore took its "bare devices = one untagged segment"
// compatibility branch 100% of the time, and a config the daemon knows is split
// {200:40, 210:7, 240:6, None:3} rendered as a single "Untagged" bucket (D12).
//
// Deliberately NOT fixed by changing NormalizedSegments(): that drives
// Stack.initializeSegments, which builds a VLAN-tagged DeviceTable per segment.
// Re-grouping there would change which frames are emitted tagged — a traffic
// change, not a display fix. This groups for the view only; the engine's
// segment model is untouched.
//
// Tag 0 (config.UntaggedTag) stays a real bucket for genuinely untagged
// devices rather than a catch-all, so a mixed config reports both.
func buildSegmentResponses(cfg *config.Config) []segmentResponse {
	if len(cfg.Segments) > 0 {
		return segmentResponsesFrom(cfg.NormalizedSegments())
	}

	return segmentResponsesFrom(segmentsByDeviceVLAN(cfg.Devices))
}

// segmentsByDeviceVLAN groups a flat device list by each device's VLAN
// membership, ordered untagged-first then ascending by tag so the view is
// stable across requests.
func segmentsByDeviceVLAN(devices []config.Device) []config.Segment {
	if len(devices) == 0 {
		return nil
	}

	byTag := make(map[int][]config.Device)
	for i := range devices {
		tag := devices[i].VLAN
		byTag[tag] = append(byTag[tag], devices[i])
	}

	tags := make([]int, 0, len(byTag))
	for tag := range byTag {
		tags = append(tags, tag)
	}
	slices.Sort(tags)

	segments := make([]config.Segment, 0, len(tags))
	for _, tag := range tags {
		segments = append(segments, config.Segment{Tag: tag, Devices: byTag[tag]})
	}

	return segments
}

// segmentResponsesFrom projects segments onto the wire shape. Shared with the
// session-scoped segments endpoint so both render identically.
func segmentResponsesFrom(segments []config.Segment) []segmentResponse {
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
