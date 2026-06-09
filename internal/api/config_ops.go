package api

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
)

// MergeConfigsRequest is the body for POST /api/v1/config/merge.
//
// Both fields hold raw YAML text. The merge follows the same semantics as
// `niac config merge` on the CLI:
//
//   - Devices in `Overlay` with the same name as a device in `Base` REPLACE
//     the base entry entirely (whole-record replacement, no field-level merge).
//   - Devices that exist only in `Base` are kept.
//   - Devices that exist only in `Overlay` are appended.
type MergeConfigsRequest struct {
	Base    string `json:"base"`
	Overlay string `json:"overlay"`
}

// MergeConfigsResponse is the body for POST /api/v1/config/merge.
type MergeConfigsResponse struct {
	Merged      string `json:"merged"`
	BaseDevices int    `json:"base_devices"`
	OverlayDevs int    `json:"overlay_devices"`
	MergedDevs  int    `json:"merged_devices"`
}

// ConfigImportRequest is the body for POST /api/v1/config/import.
//
// Format is one of "yaml" (passthrough validation only) or "java-dsl"
// (legacy key-value format → YAML, equivalent to `niac config export`).
type ConfigImportRequest struct {
	Format  string `json:"format"`
	Content string `json:"content"`
}

// ConfigImportResponse is the body for POST /api/v1/config/import.
type ConfigImportResponse struct {
	YAML    string `json:"yaml"`
	Devices int    `json:"devices"`
}

// ErrConfigImportFormat is returned when an unsupported `format` is provided.
var ErrConfigImportFormat = errors.New("unsupported import format (use yaml or java-dsl)")

// handleConfigMerge merges two configs and returns the merged YAML. POST only.
func (s *Server) handleConfigMerge(w http.ResponseWriter, r *http.Request) {
	// Method gating (POST-only) is enforced declaratively by the route registry.
	var req MergeConfigsRequest
	if !decodeJSONStrict(w, r, &req, MaxRequestBodySize) {
		return
	}
	if strings.TrimSpace(req.Base) == "" {
		writeError(w, r, http.StatusBadRequest, "missing_base", "base YAML is required", nil)
		return
	}
	if strings.TrimSpace(req.Overlay) == "" {
		writeError(w, r, http.StatusBadRequest, "missing_overlay", "overlay YAML is required", nil)
		return
	}

	base, err := config.LoadYAMLBytes([]byte(req.Base))
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_base",
			fmt.Sprintf("Failed to parse base YAML: %v", err), nil)
		return
	}
	overlay, err := config.LoadYAMLBytes([]byte(req.Overlay))
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_overlay",
			fmt.Sprintf("Failed to parse overlay YAML: %v", err), nil)
		return
	}

	merged := mergeConfigsByName(base, overlay)
	yamlBytes, err := config.MarshalConfigYAML(merged)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, "marshal_failed",
			"Failed to render merged YAML", nil)
		return
	}

	s.writeJSON(w, MergeConfigsResponse{
		Merged:      string(yamlBytes),
		BaseDevices: len(base.Devices),
		OverlayDevs: len(overlay.Devices),
		MergedDevs:  len(merged.Devices),
	})
}

// mergeConfigsByName replaces base devices whose name appears in overlay with
// the overlay version, then appends any overlay-only devices. Mirrors the
// semantics of cmd/niac/config.go's mergeConfigs(); duplicated here so the API
// doesn't depend on the cmd/ binary package.
func mergeConfigsByName(base, overlay *config.Config) *config.Config {
	merged := &config.Config{Devices: make([]config.Device, 0, len(base.Devices)+len(overlay.Devices))}

	overlayByName := make(map[string]*config.Device, len(overlay.Devices))
	for i := range overlay.Devices {
		overlayByName[overlay.Devices[i].Name] = &overlay.Devices[i]
	}

	for _, dev := range base.Devices {
		if replacement, present := overlayByName[dev.Name]; present {
			merged.Devices = append(merged.Devices, *replacement)
			delete(overlayByName, dev.Name)
		} else {
			merged.Devices = append(merged.Devices, dev)
		}
	}
	for _, remaining := range overlayByName {
		merged.Devices = append(merged.Devices, *remaining)
	}
	return merged
}

// handleConfigImport converts a non-YAML config (currently: legacy Java DSL)
// to YAML, or normalises an already-YAML config. POST only.
func (s *Server) handleConfigImport(w http.ResponseWriter, r *http.Request) {
	// Method gating (POST-only) is enforced declaratively by the route registry.
	var req ConfigImportRequest
	if !decodeJSONStrict(w, r, &req, MaxRequestBodySize) {
		return
	}
	if strings.TrimSpace(req.Content) == "" {
		writeError(w, r, http.StatusBadRequest, "missing_content", "content is required", nil)
		return
	}

	cfg, err := loadAnyFormatBytes(req.Format, req.Content)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "import_failed", err.Error(), nil)
		return
	}

	yamlBytes, err := config.MarshalConfigYAML(cfg)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, "marshal_failed",
			"Failed to render YAML", nil)
		return
	}

	s.writeJSON(w, ConfigImportResponse{
		YAML:    string(yamlBytes),
		Devices: len(cfg.Devices),
	})
}

// loadAnyFormatBytes dispatches to the appropriate parser based on `format`.
// "yaml" parses in-memory; "java-dsl" writes to a sandboxed temp file and
// uses config.LoadLegacy (which only accepts paths) before unlinking.
func loadAnyFormatBytes(format, content string) (*config.Config, error) {
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "", "yaml", "yml":
		return config.LoadYAMLBytes([]byte(content))
	case "java-dsl", "legacy", "cfg":
		return loadLegacyFromBytes([]byte(content))
	default:
		return nil, ErrConfigImportFormat
	}
}

// loadLegacyFromBytes shims around config.LoadLegacy by writing the content
// to a unique tempfile under os.TempDir(), parsing it, and unlinking. The
// tempfile path stays inside TempDir() so the inline barrier in the loader
// is happy. This is the temporary bridge until config gets a LoadLegacyBytes.
func loadLegacyFromBytes(content []byte) (*config.Config, error) {
	tmp, err := os.CreateTemp("", "niac-import-*.cfg")
	if err != nil {
		return nil, fmt.Errorf("create temp file: %w", err)
	}
	tmpPath := tmp.Name()
	defer func() {
		// Inline traversal guard so static analysers see the cleanup is bounded.
		cleaned := filepath.Clean(tmpPath)
		if !strings.Contains(cleaned, "..") {
			_ = os.Remove(cleaned)
		}
	}()

	if _, err = tmp.Write(content); err != nil {
		_ = tmp.Close()
		return nil, fmt.Errorf("write temp file: %w", err)
	}
	if err = tmp.Close(); err != nil {
		return nil, fmt.Errorf("close temp file: %w", err)
	}

	cfg, err := config.LoadLegacy(tmpPath)
	if err != nil {
		return nil, fmt.Errorf("parse legacy config: %w", err)
	}
	return cfg, nil
}
