// Package replay provides a single implementation of the PCAP playback
// controller used by both the daemon (internal/daemon) and the legacy CLI's
// runtime services (cmd/niac). Before this package existed, both call sites
// kept their own copy of the same logic — the daemon's was a stub returning
// ErrReplayNotImplemented, which was the bug fixed in PR #491. Consolidating
// here means future fixes (cleanup-on-error semantics, scale-time bounds,
// etc.) can't drift between the two consumers.
package replay

import (
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/MustardSeedNetworks/niac-go/internal/api"
	"github.com/MustardSeedNetworks/niac-go/internal/capture"
	"github.com/MustardSeedNetworks/niac-go/internal/config"
)

// Sentinel errors callers can match against with errors.Is.
var (
	// ErrEngineUnavailable means Start was called before a capture engine
	// was attached to the controller.
	ErrEngineUnavailable = errors.New("capture engine unavailable for replay")
	// ErrFileRequired means the request was missing a non-empty file path.
	ErrFileRequired = errors.New("pcap file path is required")
)

// Controller drives capture.PlaybackEngine for the API's ReplayManager
// interface. Concurrent-safe; methods serialise on a single internal mutex.
type Controller struct {
	engine     *capture.Engine
	debugLevel int

	mu      sync.Mutex
	current *capture.PlaybackEngine
	state   api.ReplayState
	cleanup string // path to delete when the next Start/Stop runs (uploaded files only)
}

// New returns a Controller bound to the given capture engine. debugLevel is
// passed through to each new PlaybackEngine instance.
func New(engine *capture.Engine, debugLevel int) *Controller {
	return &Controller{engine: engine, debugLevel: debugLevel}
}

// Status returns the current replay state, including live progress counters
// read from the running capture.PlaybackEngine (if any).
func (c *Controller) Status() api.ReplayState {
	c.mu.Lock()
	defer c.mu.Unlock()

	state := c.state
	if c.current != nil {
		progress := c.current.Progress()
		state.PacketsSent = progress.PacketsSent
		state.BytesSent = progress.BytesSent
		state.PacketsTotal = progress.TotalPackets
		state.BytesTotal = progress.TotalBytes
		state.PercentComplete = percentComplete(progress.PacketsSent, progress.TotalPackets)
		state.Passes = progress.Passes
	}
	return state
}

// percentScale converts a fraction to a percentage; percentRoundingFactor
// rounds percentComplete's result to two decimal places.
const (
	percentScale           = 100
	percentRoundingFactor  = 100
	percentCompleteMaximum = 100
)

// percentComplete returns sent/total as a percentage in [0, 100], rounded to
// two decimal places. It returns 0 (which ReplayState omits from its JSON
// via omitempty) when total is unknown rather than fabricating a value.
func percentComplete(sent, total uint64) float64 {
	if total == 0 {
		return 0
	}
	pct := float64(sent) / float64(total) * percentScale
	if pct > percentCompleteMaximum {
		pct = percentCompleteMaximum
	}
	return math.Round(pct*percentRoundingFactor) / percentRoundingFactor
}

// Start begins PCAP replay with the given request. If a replay is already
// running it is stopped first. On error from the underlying engine, an
// uploaded temp file (req.Uploaded == true) is removed before returning.
func (c *Controller) Start(req api.ReplayRequest) (api.ReplayState, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.engine == nil {
		return c.state, ErrEngineUnavailable
	}
	if strings.TrimSpace(req.File) == "" {
		return c.state, ErrFileRequired
	}

	if c.current != nil {
		c.current.Stop()
		c.current = nil
	}
	c.cleanupTempFile()

	cfg := &config.CapturePlayback{
		FileName:      req.File,
		RootDir:       req.RootDir,
		LoopTime:      req.LoopMs,
		ScaleTime:     req.Scale,
		RateMode:      config.RateMode(req.RateMode),
		PacketsPerSec: req.Pps,
		MbpsCap:       req.MbpsCap,
		LoopCount:     req.LoopCount,
	}
	player := capture.NewPlaybackEngine(c.engine, cfg, c.debugLevel)
	if err := player.Start(); err != nil {
		if req.Uploaded {
			// Inline traversal guard so the file deletion is bounded for
			// static analysers; the upload handler placed req.File under a
			// vetted temp dir but the inline check keeps the sink explicit.
			cleaned := filepath.Clean(req.File)
			if !strings.Contains(cleaned, "..") {
				_ = os.Remove(cleaned)
			}
		}
		return c.state, fmt.Errorf("failed to start playback: %w", err)
	}

	c.current = player
	c.state = api.ReplayState{
		Running:   true,
		File:      req.File,
		LoopMs:    req.LoopMs,
		Scale:     req.Scale,
		RateMode:  req.RateMode,
		Pps:       req.Pps,
		MbpsCap:   req.MbpsCap,
		LoopCount: req.LoopCount,
		StartedAt: time.Now().UTC(),
	}
	if req.Uploaded {
		c.cleanup = req.File
	} else {
		c.cleanup = ""
	}
	return c.state, nil
}

// Stop halts the current PCAP replay (if any) and cleans up any uploaded
// temp file. Idempotent.
func (c *Controller) Stop() (api.ReplayState, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.current != nil {
		c.current.Stop()
		c.current = nil
	}
	c.state.Running = false
	c.cleanupTempFile()
	return c.state, nil
}

func (c *Controller) cleanupTempFile() {
	if c.cleanup == "" {
		return
	}
	cleaned := filepath.Clean(c.cleanup)
	if !strings.Contains(cleaned, "..") {
		_ = os.Remove(cleaned)
	}
	c.cleanup = ""
}
