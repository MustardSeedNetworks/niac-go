package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/krisarmstrong/niac-go/internal/config"
)

// configDocument represents a configuration file document response.
type configDocument struct {
	Path        string    `json:"path"`
	Filename    string    `json:"filename"`
	ModifiedAt  time.Time `json:"modified_at"`
	SizeBytes   int64     `json:"size_bytes"`
	DeviceCount int       `json:"device_count"`
	Content     string    `json:"content"`
}

func (s *Server) handleConfig(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.handleConfigGet(w, r)
	case http.MethodPut, http.MethodPatch, http.MethodPost:
		s.handleConfigUpdate(w, r)
	default:
		w.Header().Set("Allow", "GET, PUT, PATCH, POST")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleConfigGet(w http.ResponseWriter, r *http.Request) {
	doc, status, err := s.readConfigDocument()
	if err != nil {
		// SECURITY FIX MEDIUM-6: Don't expose internal error details
		s.logger.Error("[API] Failed to read config", "error", err)
		writeError(w, r, status, "config_read_failed",
			"Failed to read configuration", nil)

		return
	}

	s.writeJSON(w, doc)
}

func (s *Server) handleConfigUpdate(w http.ResponseWriter, r *http.Request) {
	// SECURITY FIX #111: Enforce request body size limit
	r.Body = http.MaxBytesReader(w, r.Body, MaxRequestBodySize)

	// SECURITY FIX #161: Thread-safe access to ConfigPath
	if s.configPath() == "" {
		http.Error(w, "config path not available", http.StatusBadRequest)

		return
	}

	var req struct {
		Content string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		if err.Error() == ErrMsgRequestBodyTooLarge {
			writeError(
				w,
				r,
				http.StatusRequestEntityTooLarge,
				"request_too_large",
				fmt.Sprintf(
					"Request body exceeds maximum size of %d bytes",
					MaxRequestBodySize,
				),
				nil,
			)

			return
		}
		// SECURITY FIX MEDIUM-6: Don't expose internal error details
		writeError(w, r, http.StatusBadRequest, "invalid_request",
			"Failed to parse request body", nil)

		return
	}

	if strings.TrimSpace(req.Content) == "" {
		writeError(w, r, http.StatusBadRequest, "validation_failed",
			"Configuration content is required", nil)

		return
	}

	newCfg, err := config.LoadYAMLBytes([]byte(req.Content))
	if err != nil {
		// SECURITY FIX MEDIUM-6: Log details server-side, return generic message
		s.logger.Error("[API] Config validation failed", "error", err)
		writeError(w, r, http.StatusBadRequest, "config_invalid",
			"Configuration validation failed", nil)

		return
	}

	prevCfg := s.currentConfig()
	if s.cfg.ApplyConfig != nil {
		applyErr := s.cfg.ApplyConfig(newCfg)
		if applyErr != nil {
			// SECURITY FIX MEDIUM-6: Don't expose internal error details
			s.logger.Error("[API] Failed to apply config", "error", applyErr)
			writeError(w, r, http.StatusInternalServerError, "config_apply_failed",
				"Failed to apply configuration", nil)

			return
		}
	}

	writeErr := s.writeConfigFile(req.Content)
	if writeErr != nil {
		if s.cfg.ApplyConfig != nil && prevCfg != nil {
			// Attempt rollback to previous config to avoid divergence.
			if rollbackErr := s.cfg.ApplyConfig(prevCfg); rollbackErr != nil {
				s.logger.Error("[API] CRITICAL: config rollback failed", "error", rollbackErr)
			}
		}
		// SECURITY FIX MEDIUM-6: Don't expose file paths
		s.logger.Error("[API] Failed to write config file", "error", writeErr)
		writeError(w, r, http.StatusInternalServerError, "config_write_failed",
			"Failed to save configuration", nil)

		return
	}

	s.replaceConfig(newCfg)

	doc, status, err := s.readConfigDocument()
	if err != nil {
		// SECURITY FIX MEDIUM-6: Don't expose internal error details
		s.logger.Error("[API] Failed to read updated config", "error", err)
		writeError(w, r, status, "config_read_failed",
			"Configuration updated but failed to retrieve", nil)

		return
	}

	s.writeJSON(w, doc)
}

func (s *Server) readConfigDocument() (*configDocument, int, error) {
	// SECURITY FIX #161: Thread-safe access to ConfigPath
	cfgPath := s.configPath()
	if cfgPath == "" {
		return nil, http.StatusBadRequest, ErrConfigPathNotAvailable
	}

	// #nosec G304 -- cfgPath is validated by configPath() which ensures thread-safe
	// access and path validation before use
	data, err := os.ReadFile(cfgPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, http.StatusNotFound, fmt.Errorf("%w: %s", ErrConfigFileNotFound, cfgPath)
		}

		return nil, http.StatusInternalServerError, fmt.Errorf("reading config: %w", err)
	}

	info, err := os.Stat(cfgPath)
	if err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("stat config: %w", err)
	}

	cfg := s.currentConfig()

	deviceCount := 0
	if cfg != nil {
		deviceCount = len(cfg.Devices)
	}

	return &configDocument{
		Path:        cfgPath,
		Filename:    filepath.Base(cfgPath),
		ModifiedAt:  info.ModTime().UTC(),
		SizeBytes:   info.Size(),
		DeviceCount: deviceCount,
		Content:     string(data),
	}, http.StatusOK, nil
}

func (s *Server) writeConfigFile(content string) error {
	// SECURITY FIX #161: Thread-safe access to ConfigPath
	cfgPath := s.configPath()
	dir := filepath.Dir(cfgPath)

	mkdirErr := os.MkdirAll(dir, 0o750)
	if mkdirErr != nil {
		return fmt.Errorf("create config directory: %w", mkdirErr)
	}

	tmp, createErr := os.CreateTemp(dir, ".niac-config-*")
	if createErr != nil {
		return fmt.Errorf("create temp file: %w", createErr)
	}

	tmpPath := tmp.Name()

	defer func() { _ = os.Remove(tmpPath) }()

	_, writeErr := tmp.WriteString(content)
	if writeErr != nil {
		_ = tmp.Close()

		return fmt.Errorf("write temp file: %w", writeErr)
	}

	syncErr := tmp.Sync()
	if syncErr != nil {
		_ = tmp.Close()

		return fmt.Errorf("sync temp file: %w", syncErr)
	}

	closeErr := tmp.Close()
	if closeErr != nil {
		return fmt.Errorf("close temp file: %w", closeErr)
	}

	chmodErr := os.Chmod(tmpPath, 0o600)
	if chmodErr != nil {
		return fmt.Errorf("chmod temp file: %w", chmodErr)
	}

	renameErr := os.Rename(tmpPath, cfgPath)
	if renameErr != nil {
		return fmt.Errorf("replace config: %w", renameErr)
	}

	return nil
}

func (s *Server) currentConfig() *config.Config {
	s.configMu.RLock()
	defer s.configMu.RUnlock()

	return s.cfg.Config
}

func (s *Server) replaceConfig(cfg *config.Config) {
	s.configMu.Lock()
	s.cfg.Config = cfg
	s.cfg.Topology = BuildTopology(cfg)
	s.configMu.Unlock()
}

// SECURITY FIX #161: Thread-safe access to ConfigPath to prevent race conditions.
func (s *Server) configPath() string {
	s.configMu.RLock()
	defer s.configMu.RUnlock()

	return s.cfg.ConfigPath
}
