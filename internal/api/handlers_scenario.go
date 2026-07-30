package api

import (
	"errors"
	"net/http"

	"github.com/MustardSeedNetworks/niac-go/internal/scenario"
)

type scenarioGenerateResponse struct {
	Content  string            `json:"content"`
	Manifest scenario.Manifest `json:"manifest"`
}

func (s *Server) handleScenarioPacks(w http.ResponseWriter, _ *http.Request) {
	s.writeJSON(w, scenario.Packs())
}

func (s *Server) handleScenarioProfiles(w http.ResponseWriter, r *http.Request) {
	profiles := scenario.Profiles()
	if !s.libraryReady() {
		s.writeJSON(w, profiles)
		return
	}
	custom, err := scenario.CustomProfiles(s.library.Root())
	if err != nil {
		s.logger.ErrorContext(r.Context(), "[API] load captured scenario profiles", "error", err)
		writeError(
			w,
			r,
			http.StatusInternalServerError,
			"profile_catalog_failed",
			"Device profile catalog is unavailable",
			nil,
		)
		return
	}
	s.writeJSON(w, append(profiles, custom...))
}

func (s *Server) handleScenarioGenerate(w http.ResponseWriter, r *http.Request) {
	var request scenario.Request
	if !decodeJSONStrict(w, r, &request, MaxRequestBodySize) {
		return
	}

	result, err := scenario.Generate(request)
	if err != nil {
		status := scenarioGenerationErrorStatus(err)
		message := err.Error()
		if status == http.StatusInternalServerError {
			message = "Scenario generation failed"
			if s.logger != nil {
				s.logger.ErrorContext(r.Context(), "Scenario generation failed", "error", err)
			}
		}
		writeError(w, r, status, "scenario_generation_failed", message, nil)
		return
	}
	s.writeJSON(
		w,
		scenarioGenerateResponse{Content: string(result.YAML), Manifest: result.Manifest},
	)
}

func scenarioGenerationErrorStatus(err error) int {
	if errors.Is(err, scenario.ErrInvalidRequest) {
		return http.StatusBadRequest
	}
	return http.StatusInternalServerError
}
