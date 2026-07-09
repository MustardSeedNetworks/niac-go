package capture

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/layers"
	"github.com/gopacket/gopacket/pcapgo"
	"golang.org/x/net/bpf"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
)

// Playback constants.
const (
	debugLevelBasic = 2 // debug level for basic playback logging

	bitsPerByte = 8 // bits in a byte, for Mbps pacing
	bitsPerMbit = 1_000_000
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

	// Progress counters, read via Progress(). packetsSent/bytesSent reset at
	// the start of each playOnce (so a looped replay reports per-iteration
	// progress rather than an ever-growing total). Because playback now streams
	// the file rather than materializing it, the total isn't known until a pass
	// finishes: totalPackets/totalBytes stay 0 during the first pass (Progress
	// then omits percentComplete rather than faking it) and are populated from
	// cachedTotal* — the counts observed on the previous completed pass — at the
	// start of every subsequent looped pass, so a loop shows a real bar from
	// pass two on.
	packetsSent        atomic.Uint64
	bytesSent          atomic.Uint64
	totalPackets       atomic.Uint64
	totalBytes         atomic.Uint64
	cachedTotalPackets atomic.Uint64
	cachedTotalBytes   atomic.Uint64
	// passes counts completed replay iterations across the whole run (unlike
	// packetsSent it is not reset per pass), so a looping replay can report
	// "iteration N".
	passes atomic.Uint64
	// packetsFiltered counts packets skipped by BPFFilter this pass (reset
	// alongside packetsSent).
	packetsFiltered atomic.Uint64

	// bpfVM is the compiled replay filter, or nil when no BPFFilter is set. Set
	// under mu in Start before the playback goroutine spawns; read only by that
	// single goroutine, so its serial Run calls need no further locking.
	bpfVM *bpf.VM
}

// PlaybackPacket represents a packet with timestamp for playback.
type PlaybackPacket struct {
	Data      []byte
	Timestamp time.Time
}

// PlaybackProgress reports live counters for an in-progress (or just
// finished) replay iteration.
type PlaybackProgress struct {
	PacketsSent     uint64
	BytesSent       uint64
	TotalPackets    uint64
	TotalBytes      uint64
	Passes          uint64
	PacketsFiltered uint64
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

// Progress returns a snapshot of the current playback progress. Safe to call
// concurrently with playbackLoop.
func (p *PlaybackEngine) Progress() PlaybackProgress {
	return PlaybackProgress{
		PacketsSent:     p.packetsSent.Load(),
		BytesSent:       p.bytesSent.Load(),
		TotalPackets:    p.totalPackets.Load(),
		TotalBytes:      p.totalBytes.Load(),
		Passes:          p.passes.Load(),
		PacketsFiltered: p.packetsFiltered.Load(),
	}
}

// Start begins PCAP playback.
func (p *PlaybackEngine) Start() error {
	logger := slog.Default()
	if p.config == nil {
		return ErrNoPlaybackConfiguration
	}

	// Pre-flight the file (and compile any BPF filter) before spawning the
	// playback goroutine, so a bad path or filter is reported synchronously.
	if err := p.verifyPCAPOpens(); err != nil {
		return err
	}

	var filterVM *bpf.VM
	if p.config.BPFFilter != "" {
		compiled, err := p.compileFilter()
		if err != nil {
			return err
		}
		filterVM = compiled
	}

	p.mu.Lock()

	if p.running {
		p.mu.Unlock()

		return ErrPlaybackAlreadyRunning
	}

	p.running = true
	// Recreate stopChan so Start→Stop→Start is safe.
	p.stopChan = make(chan struct{})
	p.passes.Store(0)
	p.bpfVM = filterVM
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

// playbackLoop is the main playback loop. LoopTime>0 replays at that interval
// cadence; otherwise passes run back-to-back. LoopCount bounds the total passes
// (0 = unbounded when an interval is set, single-shot otherwise).
func (p *PlaybackEngine) playbackLoop() {
	defer p.wg.Done()

	if p.config.LoopTime > 0 {
		p.loopWithInterval()

		return
	}

	p.loopBackToBack()
}

// loopWithInterval replays once immediately, then on each LoopTime tick, until
// stopped or LoopCount passes have run.
func (p *PlaybackEngine) loopWithInterval() {
	ticker := time.NewTicker(time.Duration(p.config.LoopTime) * time.Millisecond)
	defer ticker.Stop()

	p.playOnce()
	if p.recordPassDone() {
		return
	}

	for {
		select {
		case <-ticker.C:
			p.playOnce()
			if p.recordPassDone() {
				return
			}
		case <-p.stopChan:
			return
		}
	}
}

// loopBackToBack replays with no inter-pass gap. With LoopCount==0 this is a
// single pass (the historical no-interval behavior); LoopCount>0 replays that
// many passes.
func (p *PlaybackEngine) loopBackToBack() {
	for {
		if p.isStopped() {
			return
		}

		p.playOnce()
		reached := p.recordPassDone()
		if p.config.LoopCount == 0 || reached {
			return
		}
	}
}

// recordPassDone increments the completed-pass counter and reports whether a
// configured LoopCount has been reached.
func (p *PlaybackEngine) recordPassDone() bool {
	done := p.passes.Add(1)

	return p.config.LoopCount > 0 && done >= uint64(p.config.LoopCount)
}

// playOnce streams the PCAP file once, sending each packet as it is read so
// the whole capture is never held in memory (multi-hundred-MB demo corpus).
func (p *PlaybackEngine) playOnce() {
	// Reset per-iteration progress. The total isn't known until this pass
	// finishes reading, so seed it from the previous pass's observed counts
	// (0 on the first pass → Progress omits percentComplete).
	p.packetsSent.Store(0)
	p.bytesSent.Store(0)
	p.packetsFiltered.Store(0)
	p.totalPackets.Store(p.cachedTotalPackets.Load())
	p.totalBytes.Store(p.cachedTotalBytes.Load())

	startTime := time.Now()

	var (
		read            uint64
		readBytes       uint64
		firstPacketTime time.Time
	)

	err := p.forEachPacket(func(pkt PlaybackPacket) bool {
		if p.isStopped() {
			return true
		}
		// Drop packets the BPF filter rejects before they affect timing or
		// counters — timing is relative to the first *sent* packet.
		if !p.matchesFilter(pkt.Data) {
			p.packetsFiltered.Add(1)

			return false
		}
		if read == 0 {
			firstPacketTime = pkt.Timestamp
		}
		// read/readBytes are the counts already sent, which the pps/mbps
		// governors pace against.
		if !p.sleepOrStop(p.pacingDelay(pkt, read, readBytes, startTime, firstPacketTime)) {
			return true
		}

		read++
		readBytes += uint64(len(pkt.Data))
		p.sendPacketWithLogging(pkt, read, p.totalPackets.Load())

		return false
	})
	if err != nil {
		p.logError("Error loading PCAP", "error", err)
		return
	}

	if read == 0 {
		p.logBasic("No packets found in PCAP file")
		return
	}

	// Publish the observed totals so this (and any subsequent looped) pass can
	// report a real percentComplete.
	p.cachedTotalPackets.Store(read)
	p.cachedTotalBytes.Store(readBytes)
	p.totalPackets.Store(read)
	p.totalBytes.Store(readBytes)

	p.logPlaybackComplete(startTime, read)
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

// waitForPacketTiming waits until it's time to send the packet under the
// original-timing rate mode. Returns false if playback was stopped during the
// wait.
func (p *PlaybackEngine) waitForPacketTiming(pkt PlaybackPacket, startTime, firstPacketTime time.Time) bool {
	return p.sleepOrStop(p.calculatePacketDelay(pkt, startTime, firstPacketTime))
}

// sleepOrStop waits for delay, returning true when it elapses or immediately if
// delay <= 0, and false if playback is stopped first.
func (p *PlaybackEngine) sleepOrStop(delay time.Duration) bool {
	if delay <= 0 {
		return true
	}

	select {
	case <-time.After(delay):
		return true
	case <-p.stopChan:
		return false
	}
}

// pacingDelay returns how long to wait before sending the current packet under
// the configured rate mode. sentPackets/sentBytes are the counts already sent
// this pass (the pps/mbps governors pace the average rate against them).
func (p *PlaybackEngine) pacingDelay(
	pkt PlaybackPacket,
	sentPackets, sentBytes uint64,
	startTime, firstPacketTime time.Time,
) time.Duration {
	switch p.config.RateMode {
	case config.RateTopspeed:
		return 0
	case config.RatePPS:
		// Send packet index N (0-based) at startTime + N/pps seconds.
		targetElapsed := float64(sentPackets) / p.config.PacketsPerSec * float64(time.Second)
		return rateDelay(startTime, time.Duration(targetElapsed))
	case config.RateMbps:
		// Hold average throughput at the cap: having already sent sentBytes,
		// the next send is due at startTime + sentBytes*8 / (mbps*1e6) seconds.
		targetElapsed := float64(sentBytes*bitsPerByte) / (p.config.MbpsCap * bitsPerMbit) * float64(time.Second)
		return rateDelay(startTime, time.Duration(targetElapsed))
	case config.RateTiming:
		return p.calculatePacketDelay(pkt, startTime, firstPacketTime)
	default: // empty / unknown → original timing
		return p.calculatePacketDelay(pkt, startTime, firstPacketTime)
	}
}

// rateDelay returns the wait needed for targetElapsed to have passed since
// startTime, or 0 if that moment is already past.
func rateDelay(startTime time.Time, targetElapsed time.Duration) time.Duration {
	if d := time.Until(startTime.Add(targetElapsed)); d > 0 {
		return d
	}

	return 0
}

// sendPacketWithLogging sends a packet and logs the result.
func (p *PlaybackEngine) sendPacketWithLogging(pkt PlaybackPacket, packetNum, totalPackets uint64) {
	err := p.engine.SendPacket(pkt.Data)
	if err != nil {
		p.logBasic("Error sending packet", "packetNum", packetNum, "error", err)
		return
	}
	p.packetsSent.Add(1)
	p.bytesSent.Add(uint64(len(pkt.Data)))
	if p.debugLevel >= debugLevelVerbose {
		slog.Debug("Sent packet", "packetNum", packetNum, "total", totalPackets, "bytes", len(pkt.Data))
	}
}

// logPlaybackComplete logs the playback completion message.
func (p *PlaybackEngine) logPlaybackComplete(startTime time.Time, packetCount uint64) {
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

// verifyPCAPOpens confirms the PCAP opens under os.Root containment (a symlink
// escaping its directory is rejected at the syscall), so Start reports a bad
// path synchronously rather than in the playback goroutine.
func (p *PlaybackEngine) verifyPCAPOpens() error {
	f, err := p.openPCAPFile()
	if err != nil {
		return err
	}

	return f.Close()
}

// compileFilter opens the capture to read its link type and compiles BPFFilter
// into a match VM. Called only when BPFFilter is set, so it never returns a nil
// VM with a nil error.
func (p *PlaybackEngine) compileFilter() (*bpf.VM, error) {
	f, err := p.openPCAPFile()
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()

	reader, err := newPacketReader(f)
	if err != nil {
		return nil, fmt.Errorf("open PCAP for filter: %w", err)
	}

	return compileBPFVM(reader.LinkType(), p.config.BPFFilter)
}

// openPCAPFile opens the configured playback file through an os.Root so the
// kernel rejects the open if any path component escapes the root — including
// via a symlink already on disk. The file is reopened on every loop
// iteration, so this containment runs on each open, closing the validate→open
// TOCTOU window that a plain pcap.OpenOffline (which re-resolves the path
// inside libpcap) would leave. The root is anchored at the API-validated
// allow-listed directory (config.RootDir) when set — covering intermediate
// path components — else at the file's own parent directory (operator-
// authored config / CLI paths). The caller owns the returned file.
func (p *PlaybackEngine) openPCAPFile() (*os.File, error) {
	cleanedName := filepath.Clean(p.config.FileName)
	if strings.Contains(cleanedName, "..") {
		return nil, fmt.Errorf("playback file path must not contain path traversal: %s", p.config.FileName)
	}

	rootDir := filepath.Clean(p.config.RootDir)
	if p.config.RootDir == "" {
		rootDir = filepath.Dir(cleanedName)
	}

	rel, err := filepath.Rel(rootDir, cleanedName)
	if err != nil {
		return nil, fmt.Errorf("resolve playback path under %s: %w", rootDir, err)
	}

	root, err := os.OpenRoot(rootDir)
	if err != nil {
		return nil, fmt.Errorf("open playback directory: %w", err)
	}
	defer func() { _ = root.Close() }()

	f, err := root.Open(rel)
	if err != nil {
		return nil, fmt.Errorf("open PCAP file %s: %w", p.config.FileName, err)
	}
	return f, nil
}

// packetReader is the shared surface of the pcapgo classic and pcapng
// readers used for playback.
type packetReader interface {
	ReadPacketData() ([]byte, gopacket.CaptureInfo, error)
	LinkType() layers.LinkType
}

// newPacketReader selects the classic-pcap or pcapng reader by sniffing
// the file magic, then rewinds. Pure-Go pcapgo replaces libpcap's
// pcap.OpenOffline on the replay-read path: it reads from an io.Reader
// (our os.Root-opened *os.File), so there is no fd shared with a C FILE*
// (the pcap.OpenOfflineFile fdopen hazard) and no CGO on this path.
func newPacketReader(f *os.File) (packetReader, error) {
	// pcapngSHB is the first four bytes of a pcapng Section Header Block;
	// classic pcap files start with a different magic (0xa1b2c3d4 et al.).
	pcapngSHB := [4]byte{0x0a, 0x0d, 0x0d, 0x0a}

	var magic [4]byte
	if _, err := io.ReadFull(f, magic[:]); err != nil {
		return nil, fmt.Errorf("read pcap magic: %w", err)
	}
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return nil, fmt.Errorf("rewind pcap: %w", err)
	}
	if magic == pcapngSHB {
		return pcapgo.NewNgReader(f, pcapgo.DefaultNgReaderOptions)
	}
	return pcapgo.NewReader(f)
}

// forEachPacket streams the configured PCAP file, invoking fn for each packet
// as it is read — never holding more than one packet, so replay memory is
// independent of file size. fn returns true to stop early (e.g. on the stop
// signal). It returns an error only when the file cannot be opened or its
// header cannot be read; a malformed/truncated trailing packet stops the read
// and replays the valid prefix (logged), matching the previous behaviour.
func (p *PlaybackEngine) forEachPacket(fn func(PlaybackPacket) bool) error {
	f, err := p.openPCAPFile()
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

	reader, err := newPacketReader(f)
	if err != nil {
		return fmt.Errorf("failed to open PCAP file: %w", err)
	}

	var read int
	for {
		data, ci, readErr := reader.ReadPacketData()
		if readErr != nil {
			// EOF is the normal terminator. A malformed/truncated trailing
			// packet stops the read and replays what was read so far — logged.
			if !errors.Is(readErr, io.EOF) {
				p.logError("stopped PCAP read at unreadable packet", "error", readErr, "read", read)
			}
			break
		}

		// Copy: pcapgo may reuse the backing array on the next read.
		buf := make([]byte, len(data))
		copy(buf, data)
		read++
		if fn(PlaybackPacket{Data: buf, Timestamp: ci.Timestamp}) {
			break
		}
	}

	return nil
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
