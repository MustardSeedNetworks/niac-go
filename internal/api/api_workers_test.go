package api

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/MustardSeedNetworks/niac-go/internal/api/ratelimit"
	"github.com/MustardSeedNetworks/niac-go/internal/api/sse"
	"github.com/MustardSeedNetworks/niac-go/internal/protocols"
)

type lockedLog struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (l *lockedLog) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.buf.Write(p)
}

func (l *lockedLog) contains(substr string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	return strings.Contains(l.buf.String(), substr)
}

// workerTestServer returns a server whose supervisor logs into the returned
// buffer, with an alert threshold set so the alert worker reads the stack on
// every tick. Its workers are stopped when the test ends.
func workerTestServer(t *testing.T) (*Server, *lockedLog) {
	t.Helper()
	server, _ := newTestServer(t)
	logs := &lockedLog{}
	server.logger = slog.New(slog.NewTextHandler(logs, nil))
	server.sseHub = sse.NewHub(sse.Config{})
	server.rateLimiter = ratelimit.NewRateLimiter(DefaultRateLimit, DefaultBurst)
	server.updateAlertConfig(AlertConfig{PacketsThreshold: 100})
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(ctx); err != nil {
			t.Errorf("Shutdown() = %v", err)
		}
	})
	return server, logs
}

// setStack swaps the stack the alert worker reads. A zero Stack has no
// statistics, so reading them panics, which stands in for any fault in a
// worker's loop.
func setStack(server *Server, stack *protocols.Stack) {
	server.configMu.Lock()
	server.cfg.Stack = stack
	server.configMu.Unlock()
}

func waitForCondition(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %s", what)
		}
		time.Sleep(time.Millisecond)
	}
}

func sseHubRunning(t *testing.T, server *Server) bool {
	t.Helper()
	rec := httptest.NewRecorder()
	server.handleSSEStatus(rec, httptest.NewRequest(http.MethodGet, "/api/v1/sse/status", nil))
	var status struct {
		Running bool `json:"running"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&status); err != nil {
		t.Fatalf("decode SSE status: %v", err)
	}
	return status.Running
}

func TestAPIWorkersRestartAPanickingWorkerAndKeepServing(t *testing.T) {
	server, logs := workerTestServer(t)
	healthy := server.cfg.Stack
	// Long enough between ticks to put the healthy stack back before the
	// restarted worker reads it again.
	server.startWorkers(time.Hour, 200*time.Millisecond)
	waitForCondition(t, "the SSE hub to run", func() bool { return sseHubRunning(t, server) })

	// Faults are armed after startWorkers returns: foundation v0.7.4's Start
	// does not survive a worker failing during launch (foundation#164).
	setStack(server, &protocols.Stack{})
	waitForCondition(t, "the alert worker's restart to be logged", func() bool {
		return logs.contains("supervised worker failed, restarting") && logs.contains("worker=api-alerts")
	})
	setStack(server, healthy)

	time.Sleep(500 * time.Millisecond)
	if logs.contains("supervised worker failed, stopping") {
		t.Fatal("one recovered panic stopped the API workers")
	}
	if !sseHubRunning(t, server) {
		t.Error("SSE hub stopped after another worker's recovered panic")
	}
}

func TestAPIWorkersStopTheSSEHubWhenAWorkerExhaustsItsRestarts(t *testing.T) {
	server, logs := workerTestServer(t)
	server.startWorkers(time.Hour, time.Millisecond)
	waitForCondition(t, "the SSE hub to run", func() bool { return sseHubRunning(t, server) })

	setStack(server, &protocols.Stack{})
	waitForCondition(t, "the worker group to stop", func() bool {
		return logs.contains("supervised worker failed, stopping")
	})
	// A hub left running with no supervisor would never be stopped; a hub
	// whose loop died without Stop would leave stream handlers blocked on
	// registration. The group's stop has to reach the hub either way.
	waitForCondition(t, "the SSE hub to stop", func() bool { return !sseHubRunning(t, server) })
}
