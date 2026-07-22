package virtualtcp_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net"
	"sync"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/virtualtcp"
)

func TestPipeBuffersSimultaneousWrites(t *testing.T) {
	client, server := virtualtcp.Pipe("10.0.0.10:40000", "10.0.0.1:22")
	defer client.Close()
	defer server.Close()

	clientWritten := make(chan error, 1)
	serverWritten := make(chan error, 1)
	go func() { _, err := client.Write([]byte("client-version")); clientWritten <- err }()
	go func() { _, err := server.Write([]byte("server-version")); serverWritten <- err }()

	assertRead(t, server, "client-version")
	assertRead(t, client, "server-version")
	if err := <-clientWritten; err != nil {
		t.Fatalf("client Write() error = %v", err)
	}
	if err := <-serverWritten; err != nil {
		t.Fatalf("server Write() error = %v", err)
	}
}

func TestPacketConnBridgesDeliveredAndEmittedSegments(t *testing.T) {
	var emitted bytes.Buffer
	connection := virtualtcp.NewPacketConn(
		"10.0.0.1:22", "10.0.0.10:40000",
		func(_ context.Context, payload []byte) error {
			_, _ = emitted.Write(payload)
			return nil
		},
	)
	defer connection.Close()
	if err := connection.Deliver([]byte("client")); err != nil {
		t.Fatalf("Deliver() error = %v", err)
	}
	assertRead(t, connection, "client")
	if _, err := connection.Write([]byte("server")); err != nil {
		t.Fatalf("Write() error = %v", err)
	}
	if got := emitted.String(); got != "server" {
		t.Fatalf("emitted = %q, want server", got)
	}
}

func TestPacketConnRejectsInboundDataInsteadOfBlockingWhenFull(t *testing.T) {
	connection := virtualtcp.NewPacketConn(
		"10.0.0.1:22", "10.0.0.10:40000", func(context.Context, []byte) error { return nil },
	)
	defer connection.Close()

	for range 128 {
		if err := connection.Deliver([]byte("segment")); err != nil {
			t.Fatalf("Deliver() before capacity error = %v", err)
		}
	}
	if err := connection.Deliver([]byte("overflow")); !errors.Is(err, virtualtcp.ErrReceiveBufferFull) {
		t.Fatalf("Deliver() error = %v, want ErrReceiveBufferFull", err)
	}
}

func TestPipeDrainsAcceptedDataAfterPeerClose(t *testing.T) {
	client, server := virtualtcp.Pipe("client", "server")
	if _, err := client.Write([]byte("final")); err != nil {
		t.Fatalf("Write() error = %v", err)
	}
	_ = client.Close()
	assertRead(t, server, "final")
	buffer := make([]byte, 1)
	if _, err := server.Read(buffer); !errors.Is(err, io.EOF) {
		t.Fatalf("Read() after drain error = %v, want EOF", err)
	}
}

func TestPipeRejectsWriteAfterPeerClose(t *testing.T) {
	client, server := virtualtcp.Pipe("client", "server")
	_ = server.Close()
	if _, err := client.Write([]byte("late")); !errors.Is(err, io.ErrClosedPipe) {
		t.Fatalf("Write() error = %v, want io.ErrClosedPipe", err)
	}
	_ = client.Close()
}

func TestPacketConnIgnoresEmptyDelivery(t *testing.T) {
	connection := virtualtcp.NewPacketConn(
		"server", "client", func(context.Context, []byte) error { return nil },
	)
	defer connection.Close()
	if err := connection.Deliver(nil); err != nil {
		t.Fatalf("Deliver(nil) error = %v", err)
	}
	if err := connection.Deliver([]byte("actual")); err != nil {
		t.Fatalf("Deliver(actual) error = %v", err)
	}
	assertRead(t, connection, "actual")
}

func TestConnectionsRejectOperationsAfterClose(t *testing.T) {
	client, server := virtualtcp.Pipe("client", "server")
	if _, err := server.Write([]byte("buffered")); err != nil {
		t.Fatalf("server Write() error = %v", err)
	}
	_ = client.Close()
	buffer := make([]byte, 8)
	if _, err := client.Read(buffer); !errors.Is(err, net.ErrClosed) {
		t.Fatalf("Conn.Read() error = %v, want net.ErrClosed", err)
	}
	if _, err := client.Write([]byte("late")); !errors.Is(err, net.ErrClosed) {
		t.Fatalf("Conn.Write() error = %v, want net.ErrClosed", err)
	}
	_ = server.Close()

	packet := virtualtcp.NewPacketConn("server", "client", func(context.Context, []byte) error { return nil })
	if err := packet.Deliver([]byte("buffered")); err != nil {
		t.Fatalf("Deliver() before close error = %v", err)
	}
	_ = packet.Close()
	if _, err := packet.Read(buffer); !errors.Is(err, net.ErrClosed) {
		t.Fatalf("PacketConn.Read() error = %v, want net.ErrClosed", err)
	}
	if err := packet.Deliver([]byte("late")); !errors.Is(err, net.ErrClosed) {
		t.Fatalf("Deliver() error = %v, want net.ErrClosed", err)
	}
	if _, err := packet.Write([]byte("late")); !errors.Is(err, net.ErrClosed) {
		t.Fatalf("PacketConn.Write() error = %v, want net.ErrClosed", err)
	}
}

func TestPacketConnSerializesConcurrentWrites(t *testing.T) {
	var emitted bytes.Buffer
	connection := virtualtcp.NewPacketConn("server", "client", func(_ context.Context, _ []byte) error {
		_, _ = emitted.WriteString("payload")
		return nil
	})
	defer connection.Close()

	var writers sync.WaitGroup
	for range 32 {
		writers.Go(func() {
			if _, err := connection.Write([]byte("payload")); err != nil {
				t.Errorf("Write() error = %v", err)
			}
		})
	}
	writers.Wait()
	if got, want := emitted.Len(), 32*len("payload"); got != want {
		t.Fatalf("emitted bytes = %d, want %d", got, want)
	}
}

func TestPacketConnCloseInterruptsReadAsLocallyClosed(t *testing.T) {
	connection := virtualtcp.NewPacketConn("server", "client", func(context.Context, []byte) error { return nil })
	readResult := make(chan error, 1)
	go func() {
		_, err := connection.Read(make([]byte, 1))
		readResult <- err
	}()
	_ = connection.Close()
	if err := <-readResult; !errors.Is(err, net.ErrClosed) {
		t.Fatalf("Read() error = %v, want net.ErrClosed", err)
	}
}

func TestPacketConnCloseUnblocksActiveWrite(t *testing.T) {
	emitStarted := make(chan struct{})
	connection := virtualtcp.NewPacketConn("server", "client", func(ctx context.Context, _ []byte) error {
		close(emitStarted)
		<-ctx.Done()
		return ctx.Err()
	})
	writeResult := make(chan error, 1)
	go func() {
		_, err := connection.Write([]byte("blocked"))
		writeResult <- err
	}()
	<-emitStarted
	_ = connection.Close()
	if err := <-writeResult; !errors.Is(err, net.ErrClosed) {
		t.Fatalf("Write() error = %v, want net.ErrClosed", err)
	}
}

func assertRead(t *testing.T, reader io.Reader, want string) {
	t.Helper()
	buffer := make([]byte, len(want))
	if _, err := io.ReadFull(reader, buffer); err != nil {
		t.Fatalf("ReadFull() error = %v", err)
	}
	if got := string(buffer); got != want {
		t.Fatalf("read = %q, want %q", got, want)
	}
}
