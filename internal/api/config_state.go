package api

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/protocols"
	"github.com/MustardSeedNetworks/niac-go/internal/topology"
)

type configDocument struct {
	Path        string    `json:"path"`
	Filename    string    `json:"filename"`
	ModifiedAt  time.Time `json:"modifiedAt"`
	SizeBytes   int64     `json:"sizeBytes"`
	DeviceCount int       `json:"deviceCount"`
	Content     string    `json:"content"`
}

func (s *Server) readConfigDocument() (*configDocument, int, error) {
	// SECURITY FIX #161: Thread-safe access to ConfigPath
	cfgPath := s.configPath()
	if cfgPath == "" {
		return nil, http.StatusBadRequest, ErrConfigPathNotAvailable
	}

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

func (s *Server) currentTopology() topology.Graph {
	s.configMu.RLock()
	stack := s.cfg.Stack
	cached := s.cfg.Topology
	s.configMu.RUnlock()

	if stack != nil {
		return stack.RuntimeTopology()
	}
	return cached
}

func (s *Server) replaceConfig(cfg *config.Config) {
	s.configMu.Lock()
	s.cfg.Config = cfg
	s.cfg.Topology = topology.Build(cfg)
	s.configMu.Unlock()
}

// SECURITY FIX #161: Thread-safe access to ConfigPath to prevent race conditions.
func (s *Server) configPath() string {
	s.configMu.RLock()
	defer s.configMu.RUnlock()

	return s.cfg.ConfigPath
}

// SECURITY FIX #161: Thread-safe access to Stack to prevent race conditions.
func (s *Server) currentStack() *protocols.Stack {
	s.configMu.RLock()
	defer s.configMu.RUnlock()

	return s.cfg.Stack
}

// SECURITY FIX #161: Thread-safe access to Interface to prevent race conditions.
func (s *Server) currentInterface() string {
	s.configMu.RLock()
	defer s.configMu.RUnlock()

	return s.cfg.Interface
}
