package api

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/MustardSeedNetworks/niac-go/internal/api/builtins"
	"github.com/MustardSeedNetworks/niac-go/internal/config"
)

// The built-in scenarios are the hand-authored small end of the scenario
// catalogue (2-9 devices); /api/v1/scenario/packs serves the generated end.

const builtinScenariosPath = "/api/v1/scenario/builtins/"

// CopyBuiltinScenarioRequest copies a built-in scenario into the saved configs.
type CopyBuiltinScenarioRequest struct {
	ScenarioName  string `json:"scenarioName"`
	NewConfigName string `json:"newConfigName,omitempty"`
}

// CopyBuiltinScenarioResponse names the saved copy.
type CopyBuiltinScenarioResponse struct {
	Success    bool   `json:"success"`
	Message    string `json:"message"`
	ConfigPath string `json:"configPath"`
}

// handleBuiltinScenarios handles GET /api/v1/scenario/builtins.
func (s *Server) handleBuiltinScenarios(w http.ResponseWriter, r *http.Request) {
	list := []builtins.Scenario{}

	for _, dir := range builtins.Dirs() {
		found, err := builtins.Scan(dir)
		if err != nil {
			s.logger.WarnContext(r.Context(), "Failed to scan built-in scenario dir", "dir", dir, "error", err)
			continue
		}
		list = append(list, found...)
	}

	s.writeJSON(w, list)
}

// handleBuiltinScenarioByName handles GET /api/v1/scenario/builtins/{name}.
func (s *Server) handleBuiltinScenarioByName(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimPrefix(r.URL.Path, builtinScenariosPath)
	if name == "" {
		writeError(w, r, http.StatusBadRequest, "invalid_request", "Scenario name required", nil)
		return
	}
	if strings.Contains(name, "..") || strings.Contains(name, "/") || strings.Contains(name, "\\") {
		writeError(w, r, http.StatusBadRequest, "invalid_name", "Invalid scenario name", nil)
		return
	}

	content, _, err := builtins.Load(name)
	if err != nil {
		if errors.Is(err, builtins.ErrNotFound) {
			writeError(w, r, http.StatusNotFound, "not_found", "Scenario not found: "+name, nil)
			return
		}
		writeError(w, r, http.StatusInternalServerError, "read_error", "Failed to read scenario", nil)
		return
	}

	s.writeJSON(w, builtins.Content{
		Name:    name,
		Content: string(content),
		Format:  "yaml",
	})
}

// handleBuiltinScenarioCopy handles POST /api/v1/scenario/builtins/copy.
func (s *Server) handleBuiltinScenarioCopy(w http.ResponseWriter, r *http.Request) {
	var req CopyBuiltinScenarioRequest
	if !decodeJSONStrict(w, r, &req, MaxRequestBodySize) {
		return
	}
	if req.ScenarioName == "" {
		writeError(w, r, http.StatusBadRequest, "invalid_request", "scenarioName is required", nil)
		return
	}

	content, _, err := builtins.Load(req.ScenarioName)
	if err != nil {
		writeError(w, r, http.StatusNotFound, "not_found", err.Error(), nil)
		return
	}

	cfg, err := config.LoadYAMLBytes(content)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, "invalid_scenario",
			"Scenario contains an invalid configuration", nil)
		return
	}
	if !s.authorizeConfigEntitlements(w, r, cfg) {
		return
	}

	configPath, err := builtins.SaveConfig(req.ScenarioName, req.NewConfigName, content)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, "write_error", err.Error(), nil)
		return
	}

	s.writeJSON(w, CopyBuiltinScenarioResponse{
		Success:    true,
		Message:    fmt.Sprintf("Configuration created from scenario '%s'", req.ScenarioName),
		ConfigPath: configPath,
	})
}
