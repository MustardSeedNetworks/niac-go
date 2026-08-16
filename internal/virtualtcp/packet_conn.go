package virtualtcp

import (
	"context"
	"errors"
	"io"
	"net"
	"sync"
	"time"
)

// ErrReceiveBufferFull indicates that a packet-backed stream cannot accept
// more inbound data without blocking the protocol stack.
var ErrReceiveBufferFull = errors.New("virtual TCP receive buffer full")

type writeRequest struct {
	payload []byte
	result  chan error
}

// PacketConn bridges packet payload delivery to a net.Conn byte stream.
type PacketConn struct {
	local, remote net.Addr
	inbound       chan []byte
	emit          func(context.Context, []byte) error
	done          chan struct{}
	inboundDone   chan struct{}
	writes        chan writeRequest
	cancelEmit    context.CancelFunc
	closeOnce     sync.Once
	finishOnce    sync.Once
	readMu        sync.Mutex
	deliverMu     sync.Mutex
	pending       []byte
}

// NewPacketConn creates a stream whose writes emit simulated TCP payloads.
// The emitter must return when its context is canceled.
func NewPacketConn(local, remote string, emit func(context.Context, []byte) error) *PacketConn {
	emitContext, cancelEmit := context.WithCancel(context.Background())
	connection := &PacketConn{
		local: address(local), remote: address(remote), inbound: make(chan []byte, streamBufferDepth),
		emit: emit, done: make(chan struct{}), inboundDone: make(chan struct{}),
		writes:     make(chan writeRequest, streamBufferDepth),
		cancelEmit: cancelEmit,
	}
	go connection.runEmitter(emitContext)
	return connection
}

// Deliver appends one received TCP payload to the stream.
func (c *PacketConn) Deliver(payload []byte) error {
	c.deliverMu.Lock()
	defer c.deliverMu.Unlock()
	select {
	case <-c.done:
		return net.ErrClosed
	default:
	}
	select {
	case <-c.inboundDone:
		return io.ErrClosedPipe
	default:
	}
	if len(payload) == 0 {
		return nil
	}
	segment := append([]byte(nil), payload...)
	select {
	case c.inbound <- segment:
		return nil
	case <-c.done:
		return net.ErrClosed
	default:
		return ErrReceiveBufferFull
	}
}

// Read receives delivered TCP payload bytes in order.
func (c *PacketConn) Read(destination []byte) (int, error) {
	c.readMu.Lock()
	defer c.readMu.Unlock()
	if len(destination) == 0 {
		return 0, nil
	}
	select {
	case <-c.done:
		return 0, net.ErrClosed
	default:
	}
	for len(c.pending) == 0 {
		select {
		case c.pending = <-c.inbound:
			continue
		default:
		}
		select {
		case c.pending = <-c.inbound:
		case <-c.inboundDone:
			select {
			case c.pending = <-c.inbound:
			default:
				return 0, io.EOF
			}
		case <-c.done:
			return 0, net.ErrClosed
		}
	}
	count := copy(destination, c.pending)
	c.pending = c.pending[count:]
	return count, nil
}

// FinishInbound marks the peer's send side closed while preserving payloads
// already accepted by Deliver. Reads return EOF after draining them.
func (c *PacketConn) FinishInbound() {
	c.deliverMu.Lock()
	c.finishOnce.Do(func() { close(c.inboundDone) })
	c.deliverMu.Unlock()
}

// Write emits bytes as a simulated TCP payload.
func (c *PacketConn) Write(payload []byte) (int, error) {
	request := writeRequest{payload: append([]byte(nil), payload...), result: make(chan error, 1)}
	select {
	case <-c.done:
		return 0, net.ErrClosed
	case c.writes <- request:
	}
	select {
	case <-c.done:
		return 0, net.ErrClosed
	case err := <-request.result:
		select {
		case <-c.done:
			return 0, net.ErrClosed
		default:
		}
		if err == nil {
			return len(payload), nil
		}
		return 0, err
	}
}

// Close closes the packet-backed stream.
func (c *PacketConn) Close() error {
	c.closeOnce.Do(func() {
		close(c.done)
		c.cancelEmit()
	})
	return nil
}

func (c *PacketConn) runEmitter(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case request := <-c.writes:
			request.result <- c.emit(ctx, request.payload)
		}
	}
}

// LocalAddr implements net.Conn.
func (c *PacketConn) LocalAddr() net.Addr { return c.local }

// RemoteAddr implements net.Conn.
func (c *PacketConn) RemoteAddr() net.Addr { return c.remote }

// SetDeadline implements net.Conn. Deadlines are not supported on a
// packet-backed connection; it always returns errors.ErrUnsupported.
func (c *PacketConn) SetDeadline(time.Time) error { return errors.ErrUnsupported }

// SetReadDeadline implements net.Conn. Deadlines are not supported on a
// packet-backed connection; it always returns errors.ErrUnsupported.
func (c *PacketConn) SetReadDeadline(time.Time) error { return errors.ErrUnsupported }

// SetWriteDeadline implements net.Conn. Deadlines are not supported on a
// packet-backed connection; it always returns errors.ErrUnsupported.
func (c *PacketConn) SetWriteDeadline(time.Time) error { return errors.ErrUnsupported }
