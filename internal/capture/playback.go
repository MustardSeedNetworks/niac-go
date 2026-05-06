package capture

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/pcap"

	"github.com/krisarmstrong/niac-go/internal/config"
)

// Playback constants.
const (
	debugLevelBasic  = 2    // debug level for basic playback logging
	initialPacketCap = 1000 // initial packet slice capacity
)

// Sentinel errors for playback.
var (
	ErrNoPlaybackConfiguration = errors.New("no playback configuration provided")
	ErrPlaybackAlreadyRunning  = errors.New("playback already running")
)

// PlaybackEngine handles PCAP file playback.
type PlaybackEngine struct {
	engine     *Engine
	config     *config.CapturePlayback
	debugLevel int
	running    bool
	stopChan   chan struct{}
	wg         sync.WaitGroup
	mu         sync.Mutex
}

// PlaybackPacket represents a packet with timestamp for playback.
type PlaybackPacket struct {
	Data      []byte
	Timestamp time.Time
}

// NewPlaybackEngine creates a new PCAP playback engine.
func NewPlaybackEngine(engine *Engine, playbackConfig *config.CapturePlayback, debugLevel int) *PlaybackEngine {
	return &PlaybackEngine{
		engine:     engine,
		config:     playbackConfig,
		debugLevel: debugLevel,
		stopChan:   make(chan struct{}),
	}
}

// Start begins PCAP playback.
func (p *PlaybackEngine) Start() error {
	logger := slog.Default()
	if p.config == nil {
		return ErrNoPlaybackConfiguration
	}

	// Check if PCAP file exists. The config.FileName has been validated by the
	// API layer (validatePcapFilePath), but make the bound explicit here so
	// static analysers see a barrier on this Stat call.
	cleanedName := filepath.Clean(p.config.FileName)
	if strings.Contains(cleanedName, "..") {
		return fmt.Errorf("playback file path must not contain path traversal: %s", p.config.FileName)
	}
	if _, err := os.Stat(cleanedName); err != nil {
		return fmt.Errorf("PCAP file not found: %s: %w", p.config.FileName, err)
	}

	p.mu.Lock()

	if p.running {
		p.mu.Unlock()

		return ErrPlaybackAlreadyRunning
	}

	p.running = true
	// Recreate stopChan so Start→Stop→Start is safe.
	p.stopChan = make(chan struct{})
	p.mu.Unlock()

	if p.debugLevel >= 1 {
		logger.Info("Starting PCAP playback", "filename", p.config.FileName)

		if p.config.ScaleTime > 0 && p.config.ScaleTime != 1.0 {
			logger.Info("PCAP playback time scaling", "scale", p.config.ScaleTime)
		}

		if p.config.LoopTime > 0 {
			logger.Info("PCAP playback loop interval", "intervalMs", p.config.LoopTime)
		}
	}

	// Start playback goroutine
	p.wg.Add(1)

	go p.playbackLoop()

	return nil
}

// Stop stops PCAP playback.
func (p *PlaybackEngine) Stop() {
	logger := slog.Default()
	p.mu.Lock()

	if !p.running {
		p.mu.Unlock()

		return
	}

	p.running = false
	p.mu.Unlock()

	close(p.stopChan)
	p.wg.Wait()

	if p.debugLevel >= 1 {
		logger.Info("Stopped PCAP playback")
	}
}

// playbackLoop is the main playback loop.
func (p *PlaybackEngine) playbackLoop() {
	defer p.wg.Done()

	// If LoopTime is specified, loop playback at that interval
	if p.config.LoopTime > 0 {
		loopInterval := time.Duration(p.config.LoopTime) * time.Millisecond

		ticker := time.NewTicker(loopInterval)
		defer ticker.Stop()

		// Play immediately on start
		p.playOnce()

		// Then play on each tick
		for {
			select {
			case <-ticker.C:
				p.playOnce()
			case <-p.stopChan:
				return
			}
		}
	} else {
		// Play once and exit
		p.playOnce()
	}
}

// playOnce plays the PCAP file once.
func (p *PlaybackEngine) playOnce() {
	packets, err := p.loadPCAP()
	if err != nil {
		p.logError("Error loading PCAP", "error", err)
		return
	}

	if len(packets) == 0 {
		p.logBasic("No packets found in PCAP file")
		return
	}

	p.logBasic("Replaying packets from PCAP", "count", len(packets), "filename", p.config.FileName)

	p.replayPackets(packets)
}

// replayPackets replays all packets with proper timing.
func (p *PlaybackEngine) replayPackets(packets []PlaybackPacket) {
	startTime := time.Now()
	firstPacketTime := packets[0].Timestamp
	totalPackets := len(packets)

	for i, pkt := range packets {
		if p.isStopped() {
			return
		}

		if !p.waitForPacketTiming(pkt, startTime, firstPacketTime) {
			return
		}

		p.sendPacketWithLogging(pkt, i+1, totalPackets)
	}

	p.logPlaybackComplete(startTime, totalPackets)
}

// isStopped checks if the playback should stop.
func (p *PlaybackEngine) isStopped() bool {
	select {
	case <-p.stopChan:
		return true
	default:
		return false
	}
}

// waitForPacketTiming waits until it's time to send the packet.
// Returns false if playback was stopped during the wait.
func (p *PlaybackEngine) waitForPacketTiming(pkt PlaybackPacket, startTime, firstPacketTime time.Time) bool {
	sleepDuration := p.calculatePacketDelay(pkt, startTime, firstPacketTime)
	if sleepDuration <= 0 {
		return true
	}

	select {
	case <-time.After(sleepDuration):
		return true
	case <-p.stopChan:
		return false
	}
}

// sendPacketWithLogging sends a packet and logs the result.
func (p *PlaybackEngine) sendPacketWithLogging(pkt PlaybackPacket, packetNum, totalPackets int) {
	err := p.engine.SendPacket(pkt.Data)
	if err != nil {
		p.logBasic("Error sending packet", "packetNum", packetNum, "error", err)
		return
	}
	if p.debugLevel >= debugLevelVerbose {
		slog.Debug("Sent packet", "packetNum", packetNum, "total", totalPackets, "bytes", len(pkt.Data))
	}
}

// logPlaybackComplete logs the playback completion message.
func (p *PlaybackEngine) logPlaybackComplete(startTime time.Time, packetCount int) {
	if p.debugLevel < debugLevelBasic {
		return
	}
	elapsed := time.Since(startTime)
	slog.Info("Playback complete", "packets", packetCount, "elapsed", elapsed)
}

// logError logs an error message if debug level is at least 1.
func (p *PlaybackEngine) logError(msg string, args ...any) {
	if p.debugLevel >= 1 {
		slog.Error(msg, args...)
	}
}

// logBasic logs an info message if debug level is at least debugLevelBasic.
func (p *PlaybackEngine) logBasic(msg string, args ...any) {
	if p.debugLevel >= debugLevelBasic {
		slog.Info(msg, args...)
	}
}

// loadPCAP loads packets from a PCAP file.
func (p *PlaybackEngine) loadPCAP() ([]PlaybackPacket, error) {
	// Open PCAP file. The path was bounded in Start() but reaffirm here so
	// each pcap.OpenOffline sink has a visible barrier on the cleaned path.
	cleanedName := filepath.Clean(p.config.FileName)
	if strings.Contains(cleanedName, "..") {
		return nil, fmt.Errorf("playback file path must not contain path traversal: %s", p.config.FileName)
	}
	handle, err := pcap.OpenOffline(cleanedName)
	if err != nil {
		return nil, fmt.Errorf("failed to open PCAP file: %w", err)
	}
	defer handle.Close()

	// Pre-allocate slice with reasonable initial capacity to reduce reallocations
	packets := make([]PlaybackPacket, 0, initialPacketCap)

	// Read all packets
	packetSource := gopacket.NewPacketSource(handle, handle.LinkType())
	for packet := range packetSource.Packets() {
		if packet == nil {
			break
		}

		// Store packet data and timestamp
		pkt := PlaybackPacket{
			Data:      packet.Data(),
			Timestamp: packet.Metadata().Timestamp,
		}
		packets = append(packets, pkt)
	}

	return packets, nil
}

// calculatePacketDelay calculates how long to wait before sending a packet
// based on relative timing and scaling factor.
func (p *PlaybackEngine) calculatePacketDelay(pkt PlaybackPacket, startTime, firstPacketTime time.Time) time.Duration {
	// Calculate delay relative to first packet
	relativeTime := pkt.Timestamp.Sub(firstPacketTime)

	// Apply time scaling
	if p.config.ScaleTime > 0 && p.config.ScaleTime != 1.0 {
		relativeTime = time.Duration(float64(relativeTime) * p.config.ScaleTime)
	}

	// Calculate when this packet should be sent
	targetTime := startTime.Add(relativeTime)
	now := time.Now()

	// Return sleep duration (0 if target time has passed)
	if targetTime.After(now) {
		return targetTime.Sub(now)
	}

	return 0
}

// IsRunning returns true if playback is currently running.
func (p *PlaybackEngine) IsRunning() bool {
	p.mu.Lock()
	defer p.mu.Unlock()

	return p.running
}

// GetConfig returns the playback configuration.
func (p *PlaybackEngine) GetConfig() *config.CapturePlayback {
	return p.config
}
