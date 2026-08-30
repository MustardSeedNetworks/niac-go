package capture

import (
	"errors"
	"slices"
	"sync"

	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/layers"
	"github.com/gopacket/gopacket/pcap"
)

// errFakeHandle is the failure a fakeHandle reports when configured to reject.
var errFakeHandle = errors.New("fake handle rejected the call")

// fakeHandle substitutes for *pcap.Handle so the Engine tests can assert the
// frames Engine writes and the filter it installs. A real handle needs
// CAP_NET_RAW, so these paths were previously exercised only as "did not
// panic", if at all.
type fakeHandle struct {
	mu sync.Mutex

	written  [][]byte
	filters  []string
	closes   int
	linkType layers.LinkType
	stats    pcap.Stats

	// reads is drained one entry per ReadPacketData call; once empty the
	// handle reports io.EOF-equivalent end of capture via readErr.
	reads   []readResult
	readErr error

	// writeErr, filterErr and statsErr, when set, make the corresponding call
	// fail so the error paths are covered too.
	writeErr  error
	filterErr error
	statsErr  error
}

// readResult is one queued ReadPacketData outcome.
type readResult struct {
	data []byte
	ci   gopacket.CaptureInfo
	err  error
}

func (h *fakeHandle) WritePacketData(data []byte) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.writeErr != nil {
		return h.writeErr
	}

	h.written = append(h.written, slices.Clone(data))

	return nil
}

func (h *fakeHandle) ReadPacketData() ([]byte, gopacket.CaptureInfo, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if len(h.reads) == 0 {
		return nil, gopacket.CaptureInfo{}, h.readErr
	}

	next := h.reads[0]
	h.reads = h.reads[1:]

	return next.data, next.ci, next.err
}

func (h *fakeHandle) SetBPFFilter(expr string) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.filterErr != nil {
		return h.filterErr
	}

	h.filters = append(h.filters, expr)

	return nil
}

func (h *fakeHandle) LinkType() layers.LinkType {
	h.mu.Lock()
	defer h.mu.Unlock()

	return h.linkType
}

func (h *fakeHandle) Stats() (*pcap.Stats, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.statsErr != nil {
		return nil, h.statsErr
	}

	stats := h.stats

	return &stats, nil
}

func (h *fakeHandle) Close() {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.closes++
}

// writtenFrames returns a snapshot of the frames written so far.
func (h *fakeHandle) writtenFrames() [][]byte {
	h.mu.Lock()
	defer h.mu.Unlock()

	return slices.Clone(h.written)
}

// installedFilters returns the filter expressions passed to SetBPFFilter.
func (h *fakeHandle) installedFilters() []string {
	h.mu.Lock()
	defer h.mu.Unlock()

	return slices.Clone(h.filters)
}

// closeCount returns how many times Close was called.
func (h *fakeHandle) closeCount() int {
	h.mu.Lock()
	defer h.mu.Unlock()

	return h.closes
}

// newFakeEngine builds an Engine over a fakeHandle, returning both so a test
// can assert on what reached the handle.
func newFakeEngine(debugLevel int) (*Engine, *fakeHandle) {
	handle := &fakeHandle{linkType: layers.LinkTypeEthernet}
	engine := &Engine{
		interfaceName: "fake0",
		handle:        handle,
		debugLevel:    debugLevel,
	}

	return engine, handle
}
