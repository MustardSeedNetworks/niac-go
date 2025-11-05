package capture

import (
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/pcap"
	"github.com/krisarmstrong/niac-go/pkg/config"
)

// PlaybackEngine handles PCAP file playback
type PlaybackEngine struct {
	engine      *Engine
	config      *config.CapturePlayback
	debugLevel  int
	running     bool
	stopChan    chan struct{}
	wg          sync.WaitGroup
	mu          sync.Mutex
}

// PlaybackPacket represents a packet with timestamp for playback
type PlaybackPacket struct {
	Data      []byte
	Timestamp time.Time
}

// NewPlaybackEngine creates a new PCAP playback engine
func NewPlaybackEngine(engine *Engine, playbackConfig *config.CapturePlayback, debugLevel int) *PlaybackEngine {
	return &PlaybackEngine{
		engine:     engine,
		config:     playbackConfig,
		debugLevel: debugLevel,
		stopChan:   make(chan struct{}),
	}
}

// Start begins PCAP playback
func (p *PlaybackEngine) Start() error {
	if p.config == nil {
		return fmt.Errorf("no playback configuration provided")
	}

	// Check if PCAP file exists
	if _, err := os.Stat(p.config.FileName); err != nil {
		return fmt.Errorf("PCAP file not found: %s: %w", p.config.FileName, err)
	}

	p.mu.Lock()
	if p.running {
		p.mu.Unlock()
		return fmt.Errorf("playback already running")
	}
	p.running = true
	p.mu.Unlock()

	if p.debugLevel >= 1 {
		log.Printf("Starting PCAP playback: %s", p.config.FileName)
		if p.config.ScaleTime > 0 && p.config.ScaleTime != 1.0 {
			log.Printf("  Time scaling: %.2fx", p.config.ScaleTime)
		}
		if p.config.LoopTime > 0 {
			log.Printf("  Loop interval: %dms", p.config.LoopTime)
		}
	}

	// Start playback goroutine
	p.wg.Add(1)
	go p.playbackLoop()

	return nil
}

// Stop stops PCAP playback
func (p *PlaybackEngine) Stop() {
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
		log.Printf("Stopped PCAP playback")
	}
}

// playbackLoop is the main playback loop
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

// playOnce plays the PCAP file once
func (p *PlaybackEngine) playOnce() {
	// Load packets from PCAP
	packets, err := p.loadPCAP()
	if err != nil {
		if p.debugLevel >= 1 {
			log.Printf("Error loading PCAP: %v", err)
		}
		return
	}

	if len(packets) == 0 {
		if p.debugLevel >= 2 {
			log.Printf("No packets found in PCAP file")
		}
		return
	}

	if p.debugLevel >= 2 {
		log.Printf("Replaying %d packets from %s", len(packets), p.config.FileName)
	}

	// Replay packets with timing
	startTime := time.Now()
	firstPacketTime := packets[0].Timestamp

	for i, pkt := range packets {
		// Check if we should stop
		select {
		case <-p.stopChan:
			return
		default:
		}

		// Calculate delay relative to first packet
		relativeTime := pkt.Timestamp.Sub(firstPacketTime)

		// Apply time scaling
		if p.config.ScaleTime > 0 && p.config.ScaleTime != 1.0 {
			relativeTime = time.Duration(float64(relativeTime) * p.config.ScaleTime)
		}

		// Calculate when this packet should be sent
		targetTime := startTime.Add(relativeTime)
		now := time.Now()

		// Sleep until target time
		if targetTime.After(now) {
			sleepDuration := targetTime.Sub(now)
			select {
			case <-time.After(sleepDuration):
			case <-p.stopChan:
				return
			}
		}

		// Send packet
		if err := p.engine.SendPacket(pkt.Data); err != nil {
			if p.debugLevel >= 2 {
				log.Printf("Error sending packet %d: %v", i+1, err)
			}
		} else if p.debugLevel >= 3 {
			log.Printf("Sent packet %d/%d (%d bytes)", i+1, len(packets), len(pkt.Data))
		}
	}

	if p.debugLevel >= 2 {
		elapsed := time.Since(startTime)
		log.Printf("Playback complete: %d packets in %v", len(packets), elapsed)
	}
}

// loadPCAP loads packets from a PCAP file
func (p *PlaybackEngine) loadPCAP() ([]PlaybackPacket, error) {
	// Open PCAP file
	handle, err := pcap.OpenOffline(p.config.FileName)
	if err != nil {
		return nil, fmt.Errorf("failed to open PCAP file: %w", err)
	}
	defer handle.Close()

	var packets []PlaybackPacket

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

// IsRunning returns true if playback is currently running
func (p *PlaybackEngine) IsRunning() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.running
}

// GetConfig returns the playback configuration
func (p *PlaybackEngine) GetConfig() *config.CapturePlayback {
	return p.config
}
