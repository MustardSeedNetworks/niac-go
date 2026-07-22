package protocols

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net"
	"testing"
	"time"

	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/layers"

	"github.com/MustardSeedNetworks/niac-go/internal/virtualtcp"
)

func TestSSHTCPHandlerBoundsConcurrentSessions(t *testing.T) {
	handler := &sshTCPHandler{
		sessions: make(map[string]*sshTCPSession),
		now:      time.Now,
	}
	for index := range sshMaxSessions {
		reserved, _ := handler.reserveSession(string(rune(index)))
		if !reserved {
			t.Fatalf("reserveSession(%d) rejected before limit", index)
		}
	}
	reserved, atCapacity := handler.reserveSession("overflow")
	if reserved || !atCapacity {
		t.Fatal("reserveSession() accepted a session beyond the limit")
	}
}

func TestSSHTCPHandlerRejectsDuplicateReservationWithoutReportingCapacity(t *testing.T) {
	handler := &sshTCPHandler{sessions: make(map[string]*sshTCPSession), now: time.Now}
	reserved, _ := handler.reserveSession("client")
	if !reserved {
		t.Fatal("initial reservation was rejected")
	}
	reserved, atCapacity := handler.reserveSession("client")
	if reserved || atCapacity {
		t.Fatalf("duplicate reservation = (%v, %v), want (false, false)", reserved, atCapacity)
	}
}

func TestSSHTCPHandlerExpiresIdleSessionsBeforeApplyingLimit(t *testing.T) {
	now := time.Date(2026, time.July, 22, 12, 0, 0, 0, time.UTC)
	connection := virtualtcp.NewPacketConn(
		"10.0.0.1:22", "10.0.0.2:40000",
		func(context.Context, []byte) error { return nil },
	)
	handler := &sshTCPHandler{
		sessions: map[string]*sshTCPSession{
			"stale": {connection: connection, lastActivity: now.Add(-sshSessionIdleTimeout)},
		},
		now: func() time.Time { return now },
	}

	reserved, _ := handler.reserveSession("replacement")
	if !reserved {
		t.Fatal("reserveSession() rejected after stale-session cleanup")
	}
	if _, exists := handler.sessions["stale"]; exists {
		t.Fatal("stale session was not removed")
	}
	if _, err := connection.Write([]byte("closed")); err == nil {
		t.Fatal("stale connection remained open")
	}
}

func TestSSHTCPHandlerAcknowledgesRetransmissionWithoutRedelivery(t *testing.T) {
	stack := &Stack{sendQueue: make(chan *Packet, 2)}
	handler := &sshTCPHandler{stack: stack, now: time.Now}
	connection := virtualtcp.NewPacketConn(
		"10.0.0.1:22", "10.0.0.2:40000",
		func(context.Context, []byte) error { return nil },
	)
	session := &sshTCPSession{
		connection: connection, established: true, clientNext: 100, serverNext: 200,
		sourceIP: net.ParseIP("10.0.0.1"), destinationIP: net.ParseIP("10.0.0.2"),
		sourceMAC: net.HardwareAddr{0x02, 0, 0, 0, 0, 1}, destinationMAC: net.HardwareAddr{0x02, 0, 0, 0, 0, 2},
		sourcePort: TCPPortSSH, destinationPort: 40000,
	}
	segment := &layers.TCP{Seq: 100, BaseLayer: layers.BaseLayer{Payload: []byte("payload")}}
	handler.acceptSegment(session, segment)
	handler.acceptSegment(session, segment)

	buffer := make([]byte, len(segment.Payload))
	if _, err := io.ReadFull(connection, buffer); err != nil || string(buffer) != "payload" {
		t.Fatalf("first ReadFull() = %q, %v", buffer, err)
	}
	result := make(chan error, 1)
	go func() {
		_, err := connection.Read(buffer)
		result <- err
	}()
	select {
	case err := <-result:
		t.Fatalf("duplicate payload became readable: %v", err)
	case <-time.After(10 * time.Millisecond):
		_ = connection.Close()
	}
	if err := <-result; !errors.Is(err, net.ErrClosed) {
		t.Fatalf("second Read() after close = %v, want net.ErrClosed", err)
	}
}

func TestSSHTCPHandlerIgnoresOutOfWindowFIN(t *testing.T) {
	stack := &Stack{sendQueue: make(chan *Packet, 1)}
	handler := &sshTCPHandler{stack: stack, sessions: make(map[string]*sshTCPSession), now: time.Now}
	session := &sshTCPSession{
		key: "client", established: true, clientNext: 100, serverNext: 200,
		connection: virtualtcp.NewPacketConn(
			"10.0.0.1:22", "10.0.0.2:40000", func(context.Context, []byte) error { return nil },
		),
		sourceIP: net.ParseIP("10.0.0.1"), destinationIP: net.ParseIP("10.0.0.2"),
		sourceMAC:      net.HardwareAddr{0x02, 0, 0, 0, 0, 1},
		destinationMAC: net.HardwareAddr{0x02, 0, 0, 0, 0, 2},
		sourcePort:     TCPPortSSH, destinationPort: 40000,
	}
	handler.sessions[session.key] = session
	defer handler.closeSession(session.key)

	handler.acceptSegment(session, &layers.TCP{Seq: 99, FIN: true})

	if handler.session(session.key) == nil {
		t.Fatal("out-of-window FIN closed the SSH session")
	}
	if session.clientNext != 100 {
		t.Fatalf("clientNext = %d, want 100", session.clientNext)
	}
	response := <-stack.sendQueue
	packet := gopacket.NewPacket(response.Buffer, layers.LayerTypeEthernet, gopacket.Default)
	tcp, ok := packet.Layer(layers.LayerTypeTCP).(*layers.TCP)
	if !ok {
		t.Fatal("response did not contain TCP")
	}
	if !tcp.ACK || tcp.FIN || tcp.Ack != 100 {
		t.Fatalf("response flags ACK=%v FIN=%v ack=%d, want plain ACK 100", tcp.ACK, tcp.FIN, tcp.Ack)
	}
}

func TestSSHTCPHandlerRetransmitsUnacknowledgedServerSegment(t *testing.T) {
	stack := &Stack{sendQueue: make(chan *Packet, 2)}
	handler := &sshTCPHandler{stack: stack, sessions: make(map[string]*sshTCPSession), now: time.Now}
	session := &sshTCPSession{
		key: "client", connection: virtualtcp.NewPacketConn(
			"server", "client", func(context.Context, []byte) error { return nil },
		),
		sourceIP: net.ParseIP("10.0.0.1"), destinationIP: net.ParseIP("10.0.0.2"),
		sourceMAC: net.HardwareAddr{0x02, 0, 0, 0, 0, 1}, destinationMAC: net.HardwareAddr{0x02, 0, 0, 0, 0, 2},
		sourcePort: TCPPortSSH, destinationPort: 40000, serverNext: 200,
	}
	handler.sessions[session.key] = session
	defer handler.closeSession(session.key)
	if err := handler.sendSegment(session, []byte("server-data"), false, false); err != nil {
		t.Fatalf("sendSegment() error = %v", err)
	}
	first := <-stack.sendQueue
	handler.retransmit(session)
	second := <-stack.sendQueue
	if !bytes.Equal(first.Buffer, second.Buffer) {
		t.Fatal("retransmission changed the TCP segment")
	}
	handler.acknowledge(session, 211)
	session.mu.Lock()
	defer session.mu.Unlock()
	if len(session.unacknowledged) != 0 || session.retransmitTimer != nil {
		t.Fatal("acknowledged segment remained pending")
	}
}

func TestSSHTCPHandlerServeCompletionReleasesCurrentSession(t *testing.T) {
	connection := virtualtcp.NewPacketConn(
		"server", "client", func(context.Context, []byte) error { return nil },
	)
	handler := &sshTCPHandler{sessions: make(map[string]*sshTCPSession)}
	session := &sshTCPSession{key: "client", connection: connection}
	handler.sessions[session.key] = session
	handler.closeSessionIfCurrent(session.key, session)
	if handler.session(session.key) != nil {
		t.Fatal("completed SSH session retained its slot")
	}
	if _, err := connection.Write([]byte("closed")); !errors.Is(err, net.ErrClosed) {
		t.Fatalf("connection Write() error = %v, want net.ErrClosed", err)
	}
}
