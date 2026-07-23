// Package virtualtcp provides buffered byte streams for simulated TCP sessions.
package virtualtcp

import (
	"errors"
	"io"
	"net"
	"sync"
	"time"
)

const streamBufferDepth = 128

type address string

func (a address) Network() string { return "virtual-tcp" }
func (a address) String() string  { return string(a) }

// Conn is one endpoint of a buffered virtual TCP byte stream.
type Conn struct {
	local, remote net.Addr
	inbound       <-chan []byte
	outbound      chan<- []byte
	done          chan struct{}
	writeDone     chan struct{}
	peerDone      <-chan struct{}
	closeOnce     sync.Once
	readMu        sync.Mutex
	writeMu       sync.Mutex
	pending       []byte
}

// Pipe creates connected client and server stream endpoints.
func Pipe(clientAddress, serverAddress string) (*Conn, *Conn) {
	clientInbound := make(chan []byte, streamBufferDepth)
	serverInbound := make(chan []byte, streamBufferDepth)
	clientDone := make(chan struct{})
	serverDone := make(chan struct{})
	clientWriteDone := make(chan struct{})
	serverWriteDone := make(chan struct{})
	client := &Conn{
		local: address(clientAddress), remote: address(serverAddress),
		inbound: clientInbound, outbound: serverInbound, done: clientDone,
		writeDone: clientWriteDone, peerDone: serverWriteDone,
	}
	server := &Conn{
		local: address(serverAddress), remote: address(clientAddress),
		inbound: serverInbound, outbound: clientInbound, done: serverDone,
		writeDone: serverWriteDone, peerDone: clientWriteDone,
	}
	return client, server
}

// Read receives ordered bytes written by the peer.
func (c *Conn) Read(destination []byte) (int, error) {
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
	if len(c.pending) == 0 {
		select {
		case payload := <-c.inbound:
			c.pending = payload
		default:
		}
	}
	if len(c.pending) == 0 {
		select {
		case payload := <-c.inbound:
			c.pending = payload
		case <-c.done:
			return 0, net.ErrClosed
		case <-c.peerDone:
			select {
			case payload := <-c.inbound:
				c.pending = payload
			default:
				return 0, io.EOF
			}
		}
	}
	count := copy(destination, c.pending)
	c.pending = c.pending[count:]
	return count, nil
}

// Write sends one immutable byte segment to the peer.
func (c *Conn) Write(source []byte) (int, error) {
	if len(source) == 0 {
		return 0, nil
	}
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	select {
	case <-c.done:
		return 0, net.ErrClosed
	default:
	}
	select {
	case <-c.peerDone:
		return 0, io.ErrClosedPipe
	default:
	}
	payload := append([]byte(nil), source...)
	select {
	case c.outbound <- payload:
		return len(source), nil
	case <-c.done:
		return 0, net.ErrClosed
	case <-c.peerDone:
		return 0, io.ErrClosedPipe
	}
}

// Close closes this endpoint and notifies its peer.
func (c *Conn) Close() error {
	c.closeOnce.Do(func() {
		close(c.done)
		c.writeMu.Lock()
		close(c.writeDone)
		c.writeMu.Unlock()
	})
	return nil
}

// LocalAddr returns the simulated local endpoint.
func (c *Conn) LocalAddr() net.Addr { return c.local }

// RemoteAddr returns the simulated remote endpoint.
func (c *Conn) RemoteAddr() net.Addr { return c.remote }

// SetDeadline is unsupported because packet scheduling owns virtual timeouts.
func (c *Conn) SetDeadline(time.Time) error { return errors.ErrUnsupported }

// SetReadDeadline is unsupported because packet scheduling owns virtual timeouts.
func (c *Conn) SetReadDeadline(time.Time) error { return errors.ErrUnsupported }

// SetWriteDeadline is unsupported because packet scheduling owns virtual timeouts.
func (c *Conn) SetWriteDeadline(time.Time) error { return errors.ErrUnsupported }
