package api

import (
	"net/http"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/drafttopology"
	"github.com/MustardSeedNetworks/niac-go/internal/library"
)

func (s *Server) handleLibraryDraftTopologyMutation(
	w http.ResponseWriter,
	r *http.Request,
	name string,
) {
	revision, ok := requireDraftIfMatch(w, r)
	if !ok {
		return
	}
	draft, err := s.library.ReadDraft(name)
	if err != nil {
		s.writeDraftStoreError(w, r, "read topology", name, err)
		return
	}
	draft.Content, err = s.rebaseCapturedDraftContent(draft.Content)
	if err != nil {
		s.writeDraftStoreError(w, r, "rebase topology", name, err)
		return
	}
	if draft.Revision != revision {
		s.writeDraftStoreError(w, r, "mutate topology", name, library.ErrRevisionMismatch)
		return
	}

	var mutation drafttopology.Mutation
	if !decodeJSONStrict(w, r, &mutation, MaxRequestBodySize) {
		return
	}
	if sourceErr := drafttopology.ValidateSource(draft.Content); sourceErr != nil {
		writeDraftTopologyError(w, r, sourceErr)
		return
	}
	cfg, err := config.LoadYAMLBytes([]byte(draft.Content))
	if err != nil {
		s.logger.ErrorContext(
			r.Context(),
			"Draft topology source is invalid",
			"name",
			name,
			"error",
			err,
		)
		writeError(
			w,
			r,
			http.StatusBadRequest,
			"config_invalid",
			"Draft configuration is invalid",
			nil,
		)
		return
	}
	if profileErr := s.enrichCapturedDraftProfile(cfg, &mutation); profileErr != nil {
		writeError(w, r, http.StatusBadRequest, "captured_profile_invalid", profileErr.Error(), nil)
		return
	}
	if mutationErr := drafttopology.Apply(cfg, mutation); mutationErr != nil {
		writeDraftTopologyError(w, r, mutationErr)
		return
	}
	content, err := config.MarshalConfigYAML(cfg)
	if err != nil {
		s.logger.ErrorContext(
			r.Context(),
			"Draft topology serialization failed",
			"name",
			name,
			"error",
			err,
		)
		writeError(w, r, http.StatusInternalServerError, "draft_serialization_failed",
			"Failed to serialize draft configuration", nil)
		return
	}
	if !s.validateDraftContent(w, r, string(content)) {
		return
	}
	updated, err := s.library.ReplaceDraft(name, revision, string(content))
	if err != nil {
		s.writeDraftStoreError(w, r, "mutate topology", name, err)
		return
	}
	s.writeDraft(w, http.StatusOK, updated)
}

func writeDraftTopologyError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case drafttopology.IsInvalid(err):
		writeError(
			w,
			r,
			http.StatusBadRequest,
			"topology_mutation_invalid",
			"Topology mutation is invalid",
			nil,
		)
	case drafttopology.IsNotFound(err):
		writeError(
			w,
			r,
			http.StatusNotFound,
			"topology_resource_not_found",
			"Topology resource not found",
			nil,
		)
	case drafttopology.IsConflict(err):
		writeError(
			w,
			r,
			http.StatusConflict,
			"topology_conflict",
			"Topology mutation conflicts with the draft",
			nil,
		)
	default:
		writeError(
			w,
			r,
			http.StatusInternalServerError,
			"topology_mutation_failed",
			"Topology mutation failed",
			nil,
		)
	}
}
