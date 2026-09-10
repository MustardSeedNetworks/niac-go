package api

import (
	"net/http"
	"net/netip"
	"time"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
	"github.com/MustardSeedNetworks/niac-go/internal/library"
)

type draftBehaviorsReplaceRequest struct {
	Timelines []draftBehaviorTimeline `json:"timelines"`
}

type draftBehaviorTimeline struct {
	Name          string               `json:"name"`
	StartOffsetMS int                  `json:"startOffsetMs"`
	RepeatCount   int                  `json:"repeatCount"`
	Phases        []draftBehaviorPhase `json:"phases"`
}

type draftBehaviorPhase struct {
	Name          string                 `json:"name"`
	StartOffsetMS int                    `json:"startOffsetMs"`
	DurationMS    int                    `json:"durationMs"`
	Reset         bool                   `json:"reset"`
	Traffic       []draftBehaviorTraffic `json:"traffic"`
	Faults        []draftBehaviorFault   `json:"faults"`
	Actions       []draftBehaviorAction  `json:"actions"`
}

type draftBehaviorAction struct {
	Device string                       `json:"device"`
	Type   devicestate.DeviceActionType `json:"type"`
}

type draftBehaviorTraffic struct {
	Device      string `json:"device"`
	Interface   string `json:"interface"`
	Utilization int    `json:"utilization"`
}

type draftBehaviorFault struct {
	Device     string  `json:"device"`
	Interface  string  `json:"interface"`
	Type       string  `json:"type"`
	Value      *int    `json:"value"`
	Address    *string `json:"address"`
	PrefixBits *int    `json:"prefixBits"`
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
	if message := request.faultPayloadError(); message != "" {
		writeError(w, r, http.StatusBadRequest, "validation_failed", message, nil)
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
				Actions: behaviorActionsFromRequest(phase.Actions),
			}
		}
	}
	return result
}

func behaviorActionsFromRequest(authored []draftBehaviorAction) []config.BehaviorAction {
	result := make([]config.BehaviorAction, len(authored))
	for index, action := range authored {
		result[index] = config.BehaviorAction{Device: action.Device, Type: action.Type}
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
			Device: action.Device, Interface: action.Interface, Type: action.Type,
		}
		if action.Value != nil {
			result[index].Value = *action.Value
		}
		if action.Address != nil {
			result[index].Address = netip.MustParseAddr(*action.Address)
		}
		if action.PrefixBits != nil {
			result[index].PrefixBits = *action.PrefixBits
		}
	}
	return result
}

func (request draftBehaviorsReplaceRequest) faultPayloadError() string {
	for _, timeline := range request.Timelines {
		for _, phase := range timeline.Phases {
			for _, fault := range phase.Faults {
				if message := fault.payloadError(); message != "" {
					return message
				}
			}
		}
	}
	return ""
}

func (fault draftBehaviorFault) payloadError() string {
	if fault.Type == string(devicestate.FaultBadMask) {
		return validateMaskPayload(fault.PrefixBits, fault.Value, fault.Address)
	}
	if fault.PrefixBits != nil {
		return "prefixBits requires a mask fault"
	}
	return validateFaultPayload(
		fault.Type == string(devicestate.FaultDuplicateDHCPOffer) || fault.Type == string(devicestate.FaultDuplicateIP),
		fault.Value, fault.Address,
	)
}
