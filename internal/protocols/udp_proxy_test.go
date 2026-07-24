package protocols

import (
	"context"
	"net"
	"sync"
	"testing"
	"time"
)

func TestUDPProxyConcurrencyIsBoundedAndObservable(t *testing.T) {
	t.Parallel()

	stack := &Stack{stats: &Statistics{}}
	handler := NewUDPHandler(stack)
	started := make(chan struct{}, udpProxyConcurrencyLimit)
	release := make(chan struct{})

	for range udpProxyConcurrencyLimit {
		if !handler.launchProxy(func(context.Context) {
			started <- struct{}{}
			<-release
		}) {
			t.Fatal("proxy work rejected before reaching concurrency limit")
		}
	}
	for range udpProxyConcurrencyLimit {
		select {
		case <-started:
		case <-time.After(time.Second):
			t.Fatal("proxy work did not start")
		}
	}

	if handler.launchProxy(func(context.Context) {}) {
		t.Fatal("proxy work admitted above concurrency limit")
	}
	if got := stack.GetStats().UDPProxyOverloadDrops; got != 1 {
		t.Fatalf("UDP proxy overload drops = %d, want 1", got)
	}

	close(release)
	handler.Stop()
}

func TestUDPProxyStopClosesBlockedConnection(t *testing.T) {
	t.Parallel()

	stack := &Stack{stats: &Statistics{}}
	handler := NewUDPHandler(stack)
	conn := newBlockingProxyConn()
	handler.dial = func(context.Context, string) (net.Conn, error) {
		return conn, nil
	}

	request := udpProxyRequest{
		address: "192.0.2.1:9999",
		payload: []byte("probe"),
	}
	if !handler.launchProxy(func(ctx context.Context) {
		handler.runProxy(ctx, request)
	}) {
		t.Fatal("proxy work rejected")
	}

	select {
	case <-conn.readStarted:
	case <-time.After(time.Second):
		t.Fatal("proxy did not block waiting for a reply")
	}

	stopped := make(chan struct{})
	go func() {
		handler.Stop()
		close(stopped)
	}()

	select {
	case <-stopped:
	case <-time.After(time.Second):
		t.Fatal("UDP proxy shutdown did not close the blocked connection")
	}
}

type blockingProxyConn struct {
	readStarted chan struct{}
	closed      chan struct{}
	closeOnce   sync.Once
}

func newBlockingProxyConn() *blockingProxyConn {
	return &blockingProxyConn{
		readStarted: make(chan struct{}),
		closed:      make(chan struct{}),
	}
}

func (c *blockingProxyConn) Read([]byte) (int, error) {
	close(c.readStarted)
	<-c.closed
	return 0, net.ErrClosed
}

func (c *blockingProxyConn) Write(payload []byte) (int, error) {
	return len(payload), nil
}

func (c *blockingProxyConn) Close() error {
	c.closeOnce.Do(func() {
		close(c.closed)
	})
	return nil
}

func (*blockingProxyConn) LocalAddr() net.Addr              { return nil }
func (*blockingProxyConn) RemoteAddr() net.Addr             { return nil }
func (*blockingProxyConn) SetDeadline(time.Time) error      { return nil }
func (*blockingProxyConn) SetReadDeadline(time.Time) error  { return nil }
func (*blockingProxyConn) SetWriteDeadline(time.Time) error { return nil }

var _ net.Conn = (*blockingProxyConn)(nil)
