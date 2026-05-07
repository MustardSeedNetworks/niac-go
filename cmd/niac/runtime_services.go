package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/krisarmstrong/niac-go/internal/api"
	"github.com/krisarmstrong/niac-go/internal/capture"
	"github.com/krisarmstrong/niac-go/internal/config"
	"github.com/krisarmstrong/niac-go/internal/protocols"
	"github.com/krisarmstrong/niac-go/internal/replay"
	"github.com/krisarmstrong/niac-go/internal/storage"
)

type runtimeServices struct {
	storage       *storage.Storage
	apiServer     *api.Server
	stack         *protocols.Stack
	engine        *capture.Engine
	startTime     time.Time
	interfaceName string
	configName    string
	configPath    string
	deviceCount   int
	replay        api.ReplayManager
}

// resolveAPIToken returns the API token, preferring environment variable over CLI flag.
// SECURITY FIX #101: Prefer NIAC_API_TOKEN environment variable over CLI flag.
func resolveAPIToken(cliToken string) string {
	if envToken := os.Getenv("NIAC_API_TOKEN"); envToken != "" {
		return envToken
	}

	// Warn if using deprecated CLI flag
	if cliToken != "" {
		fmt.Fprintln(os.Stderr, "WARNING: --api-token flag is deprecated and exposes token in process list")
		fmt.Fprintln(os.Stderr, "    Please use NIAC_API_TOKEN environment variable instead")
	}

	return cliToken
}

// initAPIServer creates and starts the API server with the given configuration.
func (rs *runtimeServices) initAPIServer(
	apiAddr, metricsAddr string,
	stack *protocols.Stack,
	cfg *config.Config,
	interfaceName string,
	services *serviceOptions,
) error {
	topology := api.BuildTopology(cfg)
	info := readVersionInfo()
	cfgCopy := &api.ServerConfig{
		Addr:        apiAddr,
		MetricsAddr: metricsAddr,
		Token:       resolveAPIToken(services.apiToken),
		Stack:       stack,
		Config:      cfg,
		ConfigPath:  rs.configPath,
		Storage:     rs.storage,
		Interface:   interfaceName,
		Version:     info.version,
		Commit:      info.commit,
		BuildTime:   info.date,
		UIBuildHash: info.uiBuildHash,
		Topology:    topology,
		Alert: api.AlertConfig{
			PacketsThreshold: services.alertPacketsThreshold,
			WebhookURL:       services.alertWebhook,
		},
		ApplyConfig: rs.applyConfig,
		Replay:      rs.replay,
	}

	rs.apiServer = api.NewServer(*cfgCopy)

	if startErr := rs.apiServer.Start(); startErr != nil {
		return fmt.Errorf("start API server: %w", startErr)
	}

	return nil
}

func startRuntimeServices(
	engine *capture.Engine,
	stack *protocols.Stack,
	cfg *config.Config,
	interfaceName, configFile string,
	services *serviceOptions,
) (*runtimeServices, error) {
	configPath := configFile
	abs, absErr := filepath.Abs(configFile)
	if absErr == nil {
		configPath = abs
	}

	rs := new(runtimeServices)
	rs.stack = stack
	rs.engine = engine
	rs.startTime = time.Now()
	rs.interfaceName = interfaceName
	rs.configName = filepath.Base(configPath)
	rs.configPath = configPath
	rs.deviceCount = len(cfg.Devices)

	var err error
	storagePath := services.storagePath
	if strings.EqualFold(storagePath, "disabled") {
		storagePath = ""
	}

	if storagePath != "" {
		rs.storage, err = storage.Open(storagePath)
		if err != nil {
			return nil, fmt.Errorf("open storage: %w", err)
		}
	}

	if engine != nil {
		rs.replay = newReplayController(engine, stack.GetDebugLevel())
	}

	apiAddr := services.apiListen
	metricsAddr := services.metricsListen
	if apiAddr == "" && metricsAddr != "" {
		apiAddr = metricsAddr
		metricsAddr = ""
	}

	if apiAddr == "" {
		return rs, nil
	}

	if initErr := rs.initAPIServer(apiAddr, metricsAddr, stack, cfg, interfaceName, services); initErr != nil {
		if rs.storage != nil {
			_ = rs.storage.Close()
		}

		return nil, initErr
	}

	return rs, nil
}

func (rs *runtimeServices) Stop() {
	logger := slog.Default()

	if rs.replay != nil {
		// SECURITY FIX #106: Log errors during shutdown instead of silently discarding
		_, stopErr := rs.replay.Stop()
		if stopErr != nil {
			logger.Error("Error stopping replay during shutdown", "error", stopErr)
		}
	}

	if rs.apiServer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), shortTimeout*time.Second)
		defer cancel()
		// SECURITY FIX #106: Log API server shutdown errors
		err := rs.apiServer.Shutdown(ctx)
		if err != nil {
			logger.Error("Error shutting down API server", "error", err)
		}
	}

	if rs.storage != nil {
		stats := rs.stack.GetStats()
		var record storage.RunRecord
		record.StartedAt = rs.startTime
		record.Duration = time.Since(rs.startTime)
		record.Interface = rs.interfaceName
		record.ConfigName = rs.configName
		record.DeviceCount = rs.deviceCount
		record.PacketsSent = stats.PacketsSent
		record.PacketsReceived = stats.PacketsReceived
		record.Errors = stats.Errors
		// SECURITY FIX #106: Log storage errors during shutdown
		err := rs.storage.AddRun(record)
		if err != nil {
			logger.Error("Error saving run record during shutdown", "error", err)
		}
		_ = rs.storage.Close()
	}
}

func (rs *runtimeServices) applyConfig(newCfg *config.Config) error {
	if rs == nil || newCfg == nil {
		return errors.New("runtime services not initialized")
	}
	err := rs.stack.ReloadConfig(newCfg)
	if err != nil {
		return fmt.Errorf("failed to reload config: %w", err)
	}
	configureServiceHandlers(rs.stack, newCfg, rs.stack.GetDebugLevel())
	rs.deviceCount = len(newCfg.Devices)
	return nil
}

// newReplayController returns the legacy CLI's replay controller. The
// implementation now lives in internal/replay (see PR #494) so the daemon
// and the legacy CLI share one canonical copy. Keeping the helper here
// preserves the call-site signature.
func newReplayController(engine *capture.Engine, debugLevel int) *replay.Controller {
	return replay.New(engine, debugLevel)
}
