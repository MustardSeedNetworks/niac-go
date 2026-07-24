package api

import (
	"bufio"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/MustardSeedNetworks/niac-go/internal/api/sse"
	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/logging"
	"github.com/MustardSeedNetworks/niac-go/internal/protocols"
)

func TestStatsPublisherDeliversCurrentSnapshot(t *testing.T) {
	cfg := &config.Config{Devices: []config.Device{{Name: "edge-1", Type: "router"}}}
	server := &Server{
		cfg: ServerConfig{
			Stack:     protocols.NewStack(nil, cfg, logging.NewDebugConfig(0)),
			Config:    cfg,
			Interface: "en0",
			Version:   "test",
		},
		logger:    slog.Default(),
		sseHub:    sse.NewHub(sse.Config{}),
		bgStop:    make(chan struct{}),
		startTime: time.Now(),
	}
	go server.sseHub.Run()
	go server.startStatsPublisher(10 * time.Millisecond)
	defer func() {
		if err := server.Shutdown(context.Background()); err != nil {
			t.Errorf("shutdown: %v", err)
		}
	}()

	httpServer := httptest.NewServer(http.HandlerFunc(server.handleSSEStats))
	defer httpServer.Close()

	client := &http.Client{Timeout: 2 * time.Second}
	response, err := client.Get(httpServer.URL) // #nosec G107 -- local test server
	if err != nil {
		t.Fatalf("subscribe to stats stream: %v", err)
	}
	defer response.Body.Close()

	scanner := bufio.NewScanner(response.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		var event struct {
			Type string       `json:"type"`
			Data statsPayload `json:"data"`
		}
		if decodeErr := json.Unmarshal(
			[]byte(strings.TrimPrefix(line, "data: ")),
			&event,
		); decodeErr != nil {
			t.Fatalf("decode stats event: %v", decodeErr)
		}
		if event.Type != "stats" {
			continue
		}
		if strings.Contains(line, "device_count") || !strings.Contains(line, `"deviceCount":1`) {
			t.Fatalf("stats event did not use the camelCase wire contract: %s", line)
		}
		if event.Data.Interface != "en0" || event.Data.Version != "test" {
			t.Errorf("payload identity = %#v", event.Data)
		}
		if event.Data.DeviceCount != 1 {
			t.Errorf("device count = %d, want 1", event.Data.DeviceCount)
		}
		return
	}
	if scanErr := scanner.Err(); scanErr != nil {
		t.Fatalf("read stats stream: %v", scanErr)
	}
	t.Fatal("stats stream closed before publishing a snapshot")
}
