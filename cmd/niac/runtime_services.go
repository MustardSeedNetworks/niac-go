package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/krisarmstrong/niac-go/pkg/api"
	"github.com/krisarmstrong/niac-go/pkg/config"
	"github.com/krisarmstrong/niac-go/pkg/protocols"
	"github.com/krisarmstrong/niac-go/pkg/storage"
)

type runtimeServices struct {
	storage       *storage.Storage
	apiServer     *api.Server
	stack         *protocols.Stack
	startTime     time.Time
	interfaceName string
	configFile    string
	deviceCount   int
}

func startRuntimeServices(stack *protocols.Stack, cfg *config.Config, interfaceName, configFile string) (*runtimeServices, error) {
	rs := &runtimeServices{
		stack:         stack,
		startTime:     time.Now(),
		interfaceName: interfaceName,
		configFile:    configFile,
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
			Storage:     rs.storage,
			Interface:   interfaceName,
			Version:     version,
			Topology:    topology,
			Alert: api.AlertConfig{
				PacketsThreshold: servicesOpts.alertPacketsThreshold,
				WebhookURL:       servicesOpts.alertWebhook,
			},
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

func (rs *runtimeServices) Stop() {
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
			ConfigName:      rs.configFile,
			DeviceCount:     rs.deviceCount,
			PacketsSent:     stats.PacketsSent,
			PacketsReceived: stats.PacketsReceived,
			Errors:          stats.Errors,
		}
		_ = rs.storage.AddRun(record)
		rs.storage.Close()
	}
}
