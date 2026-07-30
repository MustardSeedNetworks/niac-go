package api

import (
	"net/http"
	"time"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/library"
)

type draftBehaviorsReplaceRequest struct {
	Timelines []draftBehaviorTimeline `json:"timelines"`
}

type draftBehaviorTimeline struct {
	Name          string               `json:"name"`
	StartOffsetMS int                  `json:"start_offset_ms"`
	RepeatCount   int                  `json:"repeat_count"`
	Phases        []draftBehaviorPhase `json:"phases"`
}

type draftBehaviorPhase struct {
	Name          string                 `json:"name"`
	StartOffsetMS int                    `json:"start_offset_ms"`
	DurationMS    int                    `json:"duration_ms"`
	Reset         bool                   `json:"reset"`
	Traffic       []draftBehaviorTraffic `json:"traffic"`
	Faults        []draftBehaviorFault   `json:"faults"`
}

type draftBehaviorTraffic struct {
	Device      string `json:"device"`
	Interface   string `json:"interface"`
	Utilization int    `json:"utilization"`
}

type draftBehaviorFault struct {
	Device    string `json:"device"`
	Interface string `json:"interface"`
	Type      string `json:"type"`
	Value     int    `json:"value"`
}

func (s *Server) handleLibraryDraftBehaviorsReplace(
	w http.ResponseWriter,
	r *http.Request,
	name string,
) {
	revision, ok := requireDraftIfMatch(w, r)
	if !ok {
		return
	}
	var request draftBehaviorsReplaceRequest
	if !decodeJSONStrict(w, r, &request, MaxRequestBodySize) {
		return
	}
	draft, err := s.library.ReadDraft(name)
	if err != nil {
		s.writeDraftStoreError(w, r, "read behaviors", name, err)
		return
	}
	if draft.Revision != revision {
		s.writeDraftStoreError(w, r, "replace behaviors", name, library.ErrRevisionMismatch)
		return
	}
	cfg, err := config.LoadYAMLBytes([]byte(draft.Content))
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "config_invalid", "Draft configuration is invalid", nil)
		return
	}
	cfg.BehaviorTimelines = behaviorTimelinesFromRequest(request.Timelines)
	content, err := config.MarshalConfigYAML(cfg)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, "draft_serialization_failed",
			"Failed to serialize draft configuration", nil)
		return
	}
	if !s.validateDraftContent(w, r, string(content)) {
		return
	}
	updated, err := s.library.ReplaceDraft(name, revision, string(content))
	if err != nil {
		s.writeDraftStoreError(w, r, "replace behaviors", name, err)
		return
	}
	s.writeDraft(w, http.StatusOK, updated)
}

func behaviorTimelinesFromRequest(authored []draftBehaviorTimeline) []config.BehaviorTimeline {
	result := make([]config.BehaviorTimeline, len(authored))
	for timelineIndex, timeline := range authored {
		result[timelineIndex] = config.BehaviorTimeline{
			Name: timeline.Name, StartOffset: time.Duration(timeline.StartOffsetMS) * time.Millisecond,
			RepeatCount: timeline.RepeatCount, Phases: make([]config.BehaviorPhase, len(timeline.Phases)),
		}
		for phaseIndex, phase := range timeline.Phases {
			result[timelineIndex].Phases[phaseIndex] = config.BehaviorPhase{
				Name: phase.Name, StartOffset: time.Duration(phase.StartOffsetMS) * time.Millisecond,
				Duration: time.Duration(phase.DurationMS) * time.Millisecond, Reset: phase.Reset,
				Traffic: behaviorTrafficFromRequest(phase.Traffic),
				Faults:  behaviorFaultsFromRequest(phase.Faults),
			}
		}
	}
	return result
}

func behaviorTrafficFromRequest(authored []draftBehaviorTraffic) []config.BehaviorTraffic {
	result := make([]config.BehaviorTraffic, len(authored))
	for index, action := range authored {
		result[index] = config.BehaviorTraffic{
			Device: action.Device, Interface: action.Interface, Utilization: action.Utilization,
		}
	}
	return result
}

func behaviorFaultsFromRequest(authored []draftBehaviorFault) []config.BehaviorFault {
	result := make([]config.BehaviorFault, len(authored))
	for index, action := range authored {
		result[index] = config.BehaviorFault{
			Device: action.Device, Interface: action.Interface, Type: action.Type, Value: action.Value,
		}
	}
	return result
}
