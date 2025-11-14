package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/krisarmstrong/niac-go/pkg/api"
	"github.com/krisarmstrong/niac-go/pkg/capture"
	"github.com/krisarmstrong/niac-go/pkg/config"
	"github.com/krisarmstrong/niac-go/pkg/protocols"
	"github.com/krisarmstrong/niac-go/pkg/storage"
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

func startRuntimeServices(engine *capture.Engine, stack *protocols.Stack, cfg *config.Config, interfaceName, configFile string) (*runtimeServices, error) {
	configPath := configFile
	if abs, err := filepath.Abs(configFile); err == nil {
		configPath = abs
	}

	rs := &runtimeServices{
		stack:         stack,
		engine:        engine,
		startTime:     time.Now(),
		interfaceName: interfaceName,
		configName:    filepath.Base(configPath),
		configPath:    configPath,
		deviceCount:   len(cfg.Devices),
	}

	var err error
	storagePath := servicesOpts.storagePath
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

	apiAddr := servicesOpts.apiListen
	metricsAddr := servicesOpts.metricsListen
	if apiAddr == "" && metricsAddr != "" {
		apiAddr = metricsAddr
		metricsAddr = ""
	}

	if apiAddr != "" {
		topology := api.BuildTopology(cfg)
		cfgCopy := &api.ServerConfig{
			Addr:        apiAddr,
			MetricsAddr: metricsAddr,
			Token:       servicesOpts.apiToken,
			Stack:       stack,
			Config:      cfg,
			ConfigPath:  rs.configPath,
			Storage:     rs.storage,
			Interface:   interfaceName,
			Version:     version,
			Topology:    topology,
			Alert: api.AlertConfig{
				PacketsThreshold: servicesOpts.alertPacketsThreshold,
				WebhookURL:       servicesOpts.alertWebhook,
			},
			ApplyConfig: rs.applyConfig,
			Replay:      rs.replay,
		}

		rs.apiServer = api.NewServer(*cfgCopy)
		if err := rs.apiServer.Start(); err != nil {
			if rs.storage != nil {
				rs.storage.Close()
			}
			return nil, fmt.Errorf("start API server: %w", err)
		}
	}

	return rs, nil
}

func (rs *runtimeServices) applyConfig(newCfg *config.Config) error {
	if rs == nil || newCfg == nil {
		return fmt.Errorf("runtime services not initialized")
	}
	if err := rs.stack.ReloadConfig(newCfg); err != nil {
		return err
	}
	configureServiceHandlers(rs.stack, newCfg, rs.stack.GetDebugLevel())
	rs.deviceCount = len(newCfg.Devices)
	return nil
}

func (rs *runtimeServices) Stop() {
	if rs.replay != nil {
		_, _ = rs.replay.Stop()
	}

	if rs.apiServer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = rs.apiServer.Shutdown(ctx)
	}

	if rs.storage != nil {
		stats := rs.stack.GetStats()
		record := storage.RunRecord{
			StartedAt:       rs.startTime,
			Duration:        time.Since(rs.startTime),
			Interface:       rs.interfaceName,
			ConfigName:      rs.configName,
			DeviceCount:     rs.deviceCount,
			PacketsSent:     stats.PacketsSent,
			PacketsReceived: stats.PacketsReceived,
			Errors:          stats.Errors,
		}
		_ = rs.storage.AddRun(record)
		rs.storage.Close()
	}
}

type replayController struct {
	engine     *capture.Engine
	debugLevel int
	mu         sync.Mutex
	current    *capture.PlaybackEngine
	state      api.ReplayState
	cleanup    string
}

func newReplayController(engine *capture.Engine, debugLevel int) *replayController {
	return &replayController{
		engine:     engine,
		debugLevel: debugLevel,
	}
}

func (rc *replayController) Status() api.ReplayState {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	return rc.state
}

func (rc *replayController) Start(req api.ReplayRequest) (api.ReplayState, error) {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	if rc.engine == nil {
		return rc.state, fmt.Errorf("capture engine unavailable for replay")
	}
	if strings.TrimSpace(req.File) == "" {
		return rc.state, fmt.Errorf("pcap file path is required")
	}

	if rc.current != nil {
		rc.current.Stop()
		rc.current = nil
	}
	rc.cleanupTempFile()

	cfg := &config.CapturePlayback{
		FileName:  req.File,
		LoopTime:  req.LoopMs,
		ScaleTime: req.Scale,
	}
	player := capture.NewPlaybackEngine(rc.engine, cfg, rc.debugLevel)
	if err := player.Start(); err != nil {
		if req.Uploaded {
			os.Remove(req.File)
		}
		return rc.state, err
	}

	rc.current = player
	rc.state = api.ReplayState{
		Running:   true,
		File:      req.File,
		LoopMs:    req.LoopMs,
		Scale:     req.Scale,
		StartedAt: time.Now().UTC(),
	}
	if req.Uploaded {
		rc.cleanup = req.File
	} else {
		rc.cleanup = ""
	}
	return rc.state, nil
}

func (rc *replayController) Stop() (api.ReplayState, error) {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	if rc.current != nil {
		rc.current.Stop()
		rc.current = nil
	}
	rc.state.Running = false
	rc.cleanupTempFile()
	return rc.state, nil
}

func (rc *replayController) cleanupTempFile() {
	if rc.cleanup != "" {
		_ = os.Remove(rc.cleanup)
		rc.cleanup = ""
	}
}
