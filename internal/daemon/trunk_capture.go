package daemon

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"maps"
	"slices"
	"sync"
	"sync/atomic"

	"github.com/gopacket/gopacket"

	"github.com/MustardSeedNetworks/niac-go/internal/api"
)

const (
	dot1QEtherType       = 0x8100
	dot1QVLANMask        = 0x0fff
	dot1QHeaderLength    = 4
	ethernetHeaderLength = 14
	trunkIngressQueue    = 16384

	// A trunk can carry frames on any of the 4094 valid tags, but only the
	// operator-approved ones map to a session. Tracking every stray tag
	// individually would let whatever is on the wire size our map, so only
	// this many distinct unapproved tags are named; the rest are counted in
	// the total.
	maxTrackedUnapprovedVLANs = 16
)

var (
	// ErrTrunkVLANUnavailable means the requested 802.1Q VLAN tag is not
	// approved on the shared trunk capture.
	ErrTrunkVLANUnavailable = errors.New("trunk VLAN is not available")
	// ErrTrunkEgressVLAN means a frame queued for egress does not carry the
	// session's own physical VLAN tag.
	ErrTrunkEgressVLAN = errors.New("frame does not use the session's physical VLAN")
	// ErrTrunkCaptureFailed means the shared trunk capture goroutine has
	// stopped and can no longer deliver or send frames.
	ErrTrunkCaptureFailed = errors.New("shared trunk capture has stopped")
)

// nativeVLANKey is the demux slot for the native (untagged) session.
//
// A real trunk port carries a native VLAN alongside its tagged ones, so NIAC
// reserves key 0 for it — matching config.UntaggedTag, which already means
// "the native VLAN" everywhere else. Valid 802.1Q tags are 1..4094, so 0 can
// never collide with a tagged session (D19).
const nativeVLANKey uint16 = 0

type trunkPhysicalCapture interface {
	StartCaptureContext(context.Context, func(gopacket.Packet)) error
	SendPacket([]byte) error
	Close()
}

// trunkDrops separates why a frame was discarded. One aggregate counter cannot
// distinguish "the wire carries tags we do not serve" — normal on a shared
// trunk — from "a session cannot keep up", which is a real problem.
type trunkDrops struct {
	mu sync.Mutex
	// Untagged frames, and frames whose tag is outside the valid 1..4094 range.
	untagged uint64
	// Frames tagged for a VLAN with no session, keyed by tag up to
	// maxTrackedUnapprovedVLANs distinct tags.
	unapproved      uint64
	unapprovedByTag map[uint16]uint64
	// Frames dropped because the session's ingress queue was full, keyed by
	// the session's own VLAN and so bounded by the session count.
	overrun      uint64
	overrunByTag map[uint16]uint64
}

type trunkCapture struct {
	physical trunkPhysicalCapture

	mu       sync.RWMutex
	sessions map[uint16]*trunkSessionTransport
	drops    trunkDrops

	// Set once the physical capture stops for any reason other than a
	// deliberate cancel. Sessions bound to this trunk are dead once it is set,
	// so they must report it rather than continue to look healthy.
	failed  atomic.Bool
	failure atomic.Pointer[string]
}

type managedTrunkCapture struct {
	capture *trunkCapture
	cancel  context.CancelFunc
}

func newTrunkCapture(physical trunkPhysicalCapture) *trunkCapture {
	return &trunkCapture{
		physical: physical,
		sessions: make(map[uint16]*trunkSessionTransport),
	}
}

func (c *trunkCapture) register(vlan uint16) (*trunkSessionTransport, error) {
	// Starting a session on a trunk whose capture has died would report success
	// and then carry nothing.
	if c.failed.Load() {
		return nil, fmt.Errorf("%w", ErrTrunkCaptureFailed)
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, exists := c.sessions[vlan]; exists {
		return nil, fmt.Errorf("%w: %d", ErrTrunkVLANUnavailable, vlan)
	}
	transport := c.newTransport(vlan)
	c.sessions[vlan] = transport
	return transport, nil
}

func (c *trunkCapture) replace(vlan uint16) (*trunkSessionTransport, *trunkSessionTransport) {
	c.mu.Lock()
	defer c.mu.Unlock()
	previous := c.sessions[vlan]
	replacement := c.newTransport(vlan)
	c.sessions[vlan] = replacement
	return replacement, previous
}

func (c *trunkCapture) newTransport(vlan uint16) *trunkSessionTransport {
	return &trunkSessionTransport{
		parent: c,
		vlan:   vlan,
		rx:     make(chan []byte, trunkIngressQueue),
		closed: make(chan struct{}),
	}
}

func (c *trunkCapture) restore(vlan uint16, replacement, previous *trunkSessionTransport) {
	c.mu.Lock()
	if c.sessions[vlan] == replacement {
		c.sessions[vlan] = previous
	}
	c.mu.Unlock()
	replacement.close()
}

func (c *trunkCapture) unregister(vlan uint16, transport *trunkSessionTransport) {
	c.mu.Lock()
	if c.sessions[vlan] == transport {
		delete(c.sessions, vlan)
	}
	c.mu.Unlock()
	if transport != nil {
		transport.close()
	}
}

func (c *trunkCapture) run(ctx context.Context) error {
	return c.physical.StartCaptureContext(ctx, func(packet gopacket.Packet) {
		c.dispatchFrame(packet.Data())
	})
}

func (c *trunkCapture) dispatchFrame(frame []byte) bool {
	vlan, tagged := frameVLAN(frame)
	if !tagged {
		// Untagged frames belong to the native session, exactly as a trunk
		// port's native VLAN works. Without one registered they are still
		// dropped and counted rather than delivered somewhere arbitrary (D19).
		vlan = nativeVLANKey
	}
	c.mu.RLock()
	transport := c.sessions[vlan]
	if transport == nil {
		c.mu.RUnlock()
		if !tagged {
			c.drops.recordUntagged()
		} else {
			c.drops.recordUnapproved(vlan)
		}
		return false
	}
	select {
	case transport.rx <- append([]byte(nil), frame...):
		c.mu.RUnlock()
		return true
	default:
		c.mu.RUnlock()
		c.drops.recordOverrun(vlan)
		return false
	}
}

func (d *trunkDrops) recordUntagged() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.untagged++
}

func (d *trunkDrops) recordUnapproved(vlan uint16) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.unapproved++
	if d.unapprovedByTag == nil {
		d.unapprovedByTag = make(map[uint16]uint64)
	}
	if _, tracked := d.unapprovedByTag[vlan]; !tracked &&
		len(d.unapprovedByTag) >= maxTrackedUnapprovedVLANs {
		return
	}
	d.unapprovedByTag[vlan]++
}

func (d *trunkDrops) recordOverrun(vlan uint16) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.overrun++
	if d.overrunByTag == nil {
		d.overrunByTag = make(map[uint16]uint64)
	}
	d.overrunByTag[vlan]++
}

func (d *trunkDrops) snapshot() api.TrunkDropStats {
	d.mu.Lock()
	defer d.mu.Unlock()
	stats := api.TrunkDropStats{
		Untagged:   d.untagged,
		Unapproved: d.unapproved,
		Overrun:    d.overrun,
		Total:      d.untagged + d.unapproved + d.overrun,
	}
	if len(d.unapprovedByTag) > 0 {
		stats.UnapprovedByVLAN = maps.Clone(d.unapprovedByTag)
	}
	if len(d.overrunByTag) > 0 {
		stats.OverrunByVLAN = maps.Clone(d.overrunByTag)
	}
	return stats
}

// fail marks the trunk dead and wakes every session reading from it, so a
// blocked ReadPacket returns instead of hanging on a capture that will never
// deliver another frame.
func (c *trunkCapture) fail(err error) {
	if err == nil || !c.failed.CompareAndSwap(false, true) {
		return
	}
	message := err.Error()
	c.failure.Store(&message)
	c.mu.RLock()
	transports := make([]*trunkSessionTransport, 0, len(c.sessions))
	for _, transport := range c.sessions {
		transports = append(transports, transport)
	}
	c.mu.RUnlock()
	for _, transport := range transports {
		transport.close()
	}
}

func (c *trunkCapture) health(iface string) api.TrunkCaptureHealth {
	health := api.TrunkCaptureHealth{
		Interface: iface,
		Healthy:   !c.failed.Load(),
		Drops:     c.drops.snapshot(),
	}
	if message := c.failure.Load(); message != nil {
		health.Error = *message
	}
	c.mu.RLock()
	health.SessionVLANs = slices.Sorted(maps.Keys(c.sessions))
	c.mu.RUnlock()
	return health
}

func (c *trunkCapture) close() {
	c.physical.Close()
}

type trunkSessionTransport struct {
	parent *trunkCapture
	vlan   uint16
	rx     chan []byte
	closed chan struct{}
	once   sync.Once
}

func (t *trunkSessionTransport) ReadPacket(buffer []byte) ([]byte, error) {
	var frame []byte
	select {
	case frame = <-t.rx:
	case <-t.closed:
		return nil, nil
	}
	if len(frame) <= len(buffer) {
		copy(buffer, frame)
		return buffer[:len(frame)], nil
	}
	return frame, nil
}

func (t *trunkSessionTransport) SendPacket(frame []byte) error {
	// A dead trunk cannot carry the frame. Say so rather than writing into a
	// handle that will never deliver it.
	if t.parent.failed.Load() {
		return fmt.Errorf("%w on the interface serving VLAN %d", ErrTrunkCaptureFailed, t.vlan)
	}
	vlan, tagged := frameVLAN(frame)
	if t.vlan == nativeVLANKey {
		// The native session is the trunk's untagged VLAN: it must emit
		// untagged frames, and must not smuggle a tag onto the wire (D19).
		if tagged {
			return fmt.Errorf("%w: the native session must send untagged", ErrTrunkEgressVLAN)
		}

		return t.parent.physical.SendPacket(frame)
	}
	if !tagged || vlan != t.vlan {
		return fmt.Errorf("%w: expected VLAN %d", ErrTrunkEgressVLAN, t.vlan)
	}

	return t.parent.physical.SendPacket(frame)
}

func (*trunkSessionTransport) SetFilter(value string) error {
	if value != "" {
		return errors.New("per-session capture filters are not supported on a shared trunk")
	}
	return nil
}

func (*trunkSessionTransport) Filter() string { return "" }

func (t *trunkSessionTransport) close() {
	t.once.Do(func() { close(t.closed) })
}

func frameVLAN(frame []byte) (uint16, bool) {
	if len(frame) < ethernetHeaderLength+dot1QHeaderLength ||
		binary.BigEndian.Uint16(frame[12:14]) != dot1QEtherType {
		return 0, false
	}
	vlan := binary.BigEndian.Uint16(frame[14:16]) & dot1QVLANMask
	return vlan, vlan > 0 && vlan < 4095
}
