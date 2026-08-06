package daemon

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/gopacket/gopacket"
)

const (
	dot1QEtherType       = 0x8100
	dot1QVLANMask        = 0x0fff
	dot1QHeaderLength    = 4
	ethernetHeaderLength = 14
	trunkIngressQueue    = 16384
)

var (
	ErrTrunkVLANUnavailable = errors.New("trunk VLAN is not available")
	ErrTrunkEgressVLAN      = errors.New("frame does not use the session's physical VLAN")
)

type trunkPhysicalCapture interface {
	StartCaptureContext(context.Context, func(gopacket.Packet)) error
	SendPacket([]byte) error
	Close()
}

type trunkCapture struct {
	physical trunkPhysicalCapture

	mu       sync.RWMutex
	sessions map[uint16]*trunkSessionTransport
	drops    atomic.Uint64
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
	vlan, ok := frameVLAN(frame)
	if !ok {
		c.drops.Add(1)
		return false
	}
	c.mu.RLock()
	transport := c.sessions[vlan]
	if transport == nil {
		c.mu.RUnlock()
		c.drops.Add(1)
		return false
	}
	select {
	case transport.rx <- append([]byte(nil), frame...):
		c.mu.RUnlock()
		return true
	default:
		c.mu.RUnlock()
		c.drops.Add(1)
		return false
	}
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
	vlan, ok := frameVLAN(frame)
	if !ok || vlan != t.vlan {
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
