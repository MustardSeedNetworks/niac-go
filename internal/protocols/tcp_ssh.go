package protocols

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/layers"
	"golang.org/x/crypto/ssh"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/devicecli"
	"github.com/MustardSeedNetworks/niac-go/internal/safeconv"
	"github.com/MustardSeedNetworks/niac-go/internal/virtualtcp"
)

const (
	sshSegmentPayloadLimit = 1200
	sshMaxSessions         = 256
	sshSessionIdleTimeout  = 5 * time.Minute
	sshRetransmitTimeout   = 500 * time.Millisecond
	sshMaxRetransmits      = 4
)

type sshTCPHandler struct {
	stack    *Stack
	mu       sync.Mutex
	servers  map[*config.Device]*devicecli.SSHServer
	sessions map[string]*sshTCPSession
	now      func() time.Time
	hostKey  func(string) (ssh.Signer, error)
}

type sshTCPSession struct {
	mu                          sync.Mutex
	key                         string
	device                      *config.Device
	server                      *devicecli.SSHServer
	connection                  *virtualtcp.PacketConn
	sourceIP, destinationIP     net.IP
	sourceMAC, destinationMAC   net.HardwareAddr
	sourcePort, destinationPort layers.TCPPort
	vlan                        int
	clientNext, serverNext      uint32
	established                 bool
	lastActivity                time.Time
	unacknowledged              []sshOutboundSegment
	retransmitTimer             *time.Timer
	serverFINPending            bool
	serverFIN                   bool
	serverFINAcknowledged       bool
	clientFIN                   bool
	closed                      bool
}

type sshOutboundSegment struct {
	frame   []byte
	end     uint32
	retries int
}

func newSSHTCPHandler(stack *Stack) *sshTCPHandler {
	return &sshTCPHandler{
		stack: stack, servers: make(map[*config.Device]*devicecli.SSHServer),
		sessions: make(map[string]*sshTCPSession), now: time.Now, hostKey: loadOrCreateSSHHostSigner,
	}
}

func (h *sshTCPHandler) HandlePacket(
	packet *Packet,
	ip *layers.IPv4,
	tcp *layers.TCP,
	devices []*config.Device,
) {
	device := sshDevice(devices)
	if device == nil {
		if tcp.SYN && !tcp.ACK {
			h.stack.tcpHandler.sendRST(packet, ip, tcp, devices)
		}
		return
	}
	key := sshSessionKey(ip, tcp, packet.VLAN)
	if tcp.SYN && !tcp.ACK {
		h.openSession(key, packet, ip, tcp, device)
		return
	}
	session := h.session(key)
	if session == nil {
		return
	}
	if tcp.RST {
		h.closeSession(key)
		return
	}
	h.acceptSegment(session, tcp)
}

func sshDevice(devices []*config.Device) *config.Device {
	for _, device := range devices {
		if device.SSHConfig != nil && device.SSHConfig.Enabled {
			return device
		}
	}
	return nil
}

func (h *sshTCPHandler) openSession(
	key string,
	packet *Packet,
	ip *layers.IPv4,
	tcp *layers.TCP,
	device *config.Device,
) {
	if existing := h.retransmittableSession(key); existing != nil {
		h.retransmitOldest(existing)
		return
	}
	reserved, atCapacity := h.reserveSession(key)
	if !reserved {
		if !atCapacity {
			return
		}
		h.stack.tcpHandler.sendRST(packet, ip, tcp, []*config.Device{device})
		return
	}
	server, err := h.server(device)
	if err != nil {
		slog.Error("SSH session initialization failed", "device", device.Name)
		h.releaseReservation(key)
		h.stack.tcpHandler.sendRST(packet, ip, tcp, []*config.Device{device})
		return
	}
	identity, ok := h.stack.replyEthernet(packet, device)
	if !ok {
		h.releaseReservation(key)
		return
	}
	session := &sshTCPSession{
		key: key, device: device, server: server,
		sourceIP: append(net.IP(nil), ip.DstIP...), destinationIP: append(net.IP(nil), ip.SrcIP...),
		sourceMAC: cloneMAC(identity.source), destinationMAC: cloneMAC(identity.destination),
		sourcePort: tcp.DstPort, destinationPort: tcp.SrcPort, vlan: identity.vlan,
		clientNext: tcp.Seq + 1, serverNext: tcpInitialSeq, lastActivity: h.now(),
	}
	session.connection = virtualtcp.NewPacketConn(
		net.JoinHostPort(ip.DstIP.String(), strconv.Itoa(int(tcp.DstPort))),
		net.JoinHostPort(ip.SrcIP.String(), strconv.Itoa(int(tcp.SrcPort))),
		func(_ context.Context, payload []byte) error { return h.emitPayload(session, payload) },
	)
	h.mu.Lock()
	h.sessions[key] = session
	h.mu.Unlock()
	_ = h.sendSegment(session, nil, true, false)
}

func (h *sshTCPHandler) retransmittableSession(key string) *sshTCPSession {
	h.mu.Lock()
	session := h.sessions[key]
	if session == nil {
		h.mu.Unlock()
		return nil
	}
	session.mu.Lock()
	if h.now().Sub(session.lastActivity) < sshSessionIdleTimeout {
		session.mu.Unlock()
		h.mu.Unlock()
		return session
	}
	session.closed = true
	if session.retransmitTimer != nil {
		session.retransmitTimer.Stop()
		session.retransmitTimer = nil
	}
	delete(h.sessions, key)
	session.mu.Unlock()
	h.mu.Unlock()
	_ = session.connection.Close()
	return nil
}

func (h *sshTCPHandler) acceptSegment(session *sshTCPSession, tcp *layers.TCP) {
	if tcp.ACK {
		if h.acknowledge(session, tcp.Ack) {
			h.closeSessionIfCurrent(session.key, session)
			return
		}
	}
	session.mu.Lock()
	session.lastActivity = h.now()
	startServer := !session.established && tcp.ACK && tcp.Ack == session.serverNext
	if startServer {
		session.established = true
	}
	expectedSequence := session.clientNext
	payloadAccepted := len(tcp.Payload) > 0 && tcp.Seq == session.clientNext
	if payloadAccepted {
		session.clientNext += safeconv.Uint32(len(tcp.Payload))
	}
	finAccepted := tcp.FIN && (payloadAccepted || len(tcp.Payload) == 0 && tcp.Seq == expectedSequence)
	if finAccepted {
		session.clientNext++
		session.clientFIN = true
	}
	session.mu.Unlock()
	if startServer {
		go func() {
			_ = session.server.ServeConn(session.connection)
			h.finishServer(session)
		}()
	}
	if payloadAccepted {
		if err := session.connection.Deliver(tcp.Payload); err != nil {
			h.closeSession(session.key)
			return
		}
	}
	if len(tcp.Payload) > 0 || tcp.FIN {
		_ = h.sendSegment(session, nil, false, false)
	}
	if finAccepted {
		session.connection.FinishInbound()
		if h.sessionFinished(session) {
			h.closeSessionIfCurrent(session.key, session)
		}
	}
}

func (h *sshTCPHandler) sessionFinished(session *sshTCPSession) bool {
	session.mu.Lock()
	defer session.mu.Unlock()
	return session.clientFIN && session.serverFINAcknowledged
}

func (h *sshTCPHandler) finishServer(session *sshTCPSession) {
	session.mu.Lock()
	if session.closed || session.serverFIN || session.serverFINPending {
		session.mu.Unlock()
		return
	}
	session.serverFINPending = true
	session.mu.Unlock()
	err := h.sendSegment(session, nil, false, true)
	session.mu.Lock()
	session.serverFINPending = false
	session.serverFIN = err == nil
	if session.serverFIN && len(session.unacknowledged) == 0 {
		session.serverFINAcknowledged = true
	}
	finished := session.clientFIN && session.serverFINAcknowledged
	session.mu.Unlock()
	if err != nil || finished {
		h.closeSessionIfCurrent(session.key, session)
	}
}

func (h *sshTCPHandler) reserveSession(key string) (bool, bool) {
	h.mu.Lock()
	defer h.mu.Unlock()

	now := h.now()
	for currentKey, session := range h.sessions {
		if session == nil {
			continue
		}
		session.mu.Lock()
		stale := now.Sub(session.lastActivity) >= sshSessionIdleTimeout
		session.mu.Unlock()
		if stale {
			h.close(session)
			delete(h.sessions, currentKey)
		}
	}
	if _, exists := h.sessions[key]; exists {
		return false, false
	}
	if len(h.sessions) >= sshMaxSessions {
		return false, true
	}
	h.sessions[key] = nil
	return true, false
}

func (h *sshTCPHandler) releaseReservation(key string) {
	h.mu.Lock()
	if h.sessions[key] == nil {
		delete(h.sessions, key)
	}
	h.mu.Unlock()
}

func (h *sshTCPHandler) emitPayload(session *sshTCPSession, payload []byte) error {
	for len(payload) > 0 {
		length := min(len(payload), sshSegmentPayloadLimit)
		if err := h.sendSegment(session, payload[:length], false, false); err != nil {
			return err
		}
		payload = payload[length:]
	}
	return nil
}

func (h *sshTCPHandler) sendSegment(
	session *sshTCPSession,
	payload []byte,
	syn, fin bool,
) error {
	session.mu.Lock()
	sequence := session.serverNext
	acknowledgment := session.clientNext
	session.serverNext += safeconv.Uint32(len(payload))
	if syn || fin {
		session.serverNext++
	}
	endSequence := session.serverNext
	session.mu.Unlock()

	ip := &layers.IPv4{
		Version: tcpIPv4Version, IHL: tcpIPv4IHL, TTL: tcpIPv4TTL, Protocol: layers.IPProtocolTCP,
		SrcIP: session.sourceIP, DstIP: session.destinationIP,
	}
	tcp := &layers.TCP{
		SrcPort: session.sourcePort, DstPort: session.destinationPort,
		Seq: sequence, Ack: acknowledgment, SYN: syn, FIN: fin, ACK: true,
		PSH: len(payload) > 0, Window: tcpWindowSize,
	}
	if err := tcp.SetNetworkLayerForChecksum(ip); err != nil {
		return fmt.Errorf("set SSH TCP checksum: %w", err)
	}
	buffer := gopacket.NewSerializeBuffer()
	if err := gopacket.SerializeLayers(
		buffer,
		gopacket.SerializeOptions{FixLengths: true, ComputeChecksums: true},
		&layers.Ethernet{
			SrcMAC: session.sourceMAC, DstMAC: session.destinationMAC,
			EthernetType: layers.EthernetTypeIPv4,
		},
		ip, tcp, gopacket.Payload(payload),
	); err != nil {
		return fmt.Errorf("serialize SSH TCP segment: %w", err)
	}
	frame := buffer.Bytes()
	if len(payload) > 0 || syn || fin {
		session.mu.Lock()
		session.unacknowledged = append(session.unacknowledged, sshOutboundSegment{
			frame: append([]byte(nil), frame...), end: endSequence,
		})
		h.scheduleRetransmitLocked(session)
		session.mu.Unlock()
	}
	h.transmitFrame(session, frame)
	return nil
}

func (h *sshTCPHandler) transmitFrame(session *sshTCPSession, frame []byte) {
	h.stack.mu.Lock()
	h.stack.serialNumber++
	serial := h.stack.serialNumber
	h.stack.mu.Unlock()
	h.stack.Send(&Packet{
		Buffer: append([]byte(nil), frame...), Length: len(frame), SerialNumber: serial,
		Device: session.device, VLAN: session.vlan,
	})
}

func (h *sshTCPHandler) acknowledge(session *sshTCPSession, acknowledgment uint32) bool {
	session.mu.Lock()
	removed := false
	for len(session.unacknowledged) > 0 && acknowledgment >= session.unacknowledged[0].end {
		session.unacknowledged = session.unacknowledged[1:]
		removed = true
	}
	if removed && session.retransmitTimer != nil {
		session.retransmitTimer.Stop()
		session.retransmitTimer = nil
	}
	h.scheduleRetransmitLocked(session)
	if session.serverFIN && len(session.unacknowledged) == 0 {
		session.serverFINAcknowledged = true
	}
	finished := session.clientFIN && session.serverFINAcknowledged
	session.mu.Unlock()
	return finished
}

func (h *sshTCPHandler) scheduleRetransmitLocked(session *sshTCPSession) {
	if session.closed || session.retransmitTimer != nil || len(session.unacknowledged) == 0 {
		return
	}
	session.retransmitTimer = time.AfterFunc(sshRetransmitTimeout, func() { h.retransmit(session) })
}

func (h *sshTCPHandler) retransmit(session *sshTCPSession) {
	session.mu.Lock()
	session.retransmitTimer = nil
	if session.closed || len(session.unacknowledged) == 0 {
		session.mu.Unlock()
		return
	}
	segment := &session.unacknowledged[0]
	if segment.retries >= sshMaxRetransmits {
		session.mu.Unlock()
		h.closeSessionIfCurrent(session.key, session)
		return
	}
	segment.retries++
	frame := append([]byte(nil), segment.frame...)
	h.scheduleRetransmitLocked(session)
	session.mu.Unlock()
	h.transmitFrame(session, frame)
}

func (h *sshTCPHandler) retransmitOldest(session *sshTCPSession) {
	session.mu.Lock()
	if session.closed || len(session.unacknowledged) == 0 {
		session.mu.Unlock()
		return
	}
	frame := append([]byte(nil), session.unacknowledged[0].frame...)
	session.mu.Unlock()
	h.transmitFrame(session, frame)
}

func (h *sshTCPHandler) server(device *config.Device) (*devicecli.SSHServer, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if server := h.servers[device]; server != nil {
		return server, nil
	}
	password, found := os.LookupEnv(device.SSHConfig.PasswordEnv)
	if !found || password == "" {
		return nil, fmt.Errorf("SSH password environment variable %q is not set", device.SSHConfig.PasswordEnv)
	}
	hostSigner, err := h.hostKey(device.Name)
	if err != nil {
		return nil, err
	}
	server, err := devicecli.NewSSHServer(
		h.stack.deviceStates[device],
		devicecli.Credentials{Username: device.SSHConfig.Username, Password: password},
		hostSigner,
		h.stack.staticRouteValidator(device),
	)
	if err != nil {
		return nil, err
	}
	h.servers[device] = server
	return server, nil
}

func (h *sshTCPHandler) session(key string) *sshTCPSession {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.sessions[key]
}

func (h *sshTCPHandler) closeSession(key string) {
	h.mu.Lock()
	session := h.sessions[key]
	delete(h.sessions, key)
	h.mu.Unlock()
	if session != nil {
		h.close(session)
	}
}

func (h *sshTCPHandler) closeSessionIfCurrent(key string, expected *sshTCPSession) {
	h.mu.Lock()
	if h.sessions[key] == expected {
		delete(h.sessions, key)
	} else {
		expected = nil
	}
	h.mu.Unlock()
	if expected != nil {
		h.close(expected)
	}
}

func (h *sshTCPHandler) close(session *sshTCPSession) {
	session.mu.Lock()
	session.closed = true
	if session.retransmitTimer != nil {
		session.retransmitTimer.Stop()
		session.retransmitTimer = nil
	}
	session.mu.Unlock()
	_ = session.connection.Close()
}

func (h *sshTCPHandler) Reset() {
	h.mu.Lock()
	sessions := h.sessions
	h.sessions = make(map[string]*sshTCPSession)
	h.servers = make(map[*config.Device]*devicecli.SSHServer)
	h.mu.Unlock()
	for _, session := range sessions {
		if session != nil {
			h.close(session)
		}
	}
}

func sshSessionKey(ip *layers.IPv4, tcp *layers.TCP, vlan int) string {
	return fmt.Sprintf("%s:%d>%s:%d@%d", ip.SrcIP, tcp.SrcPort, ip.DstIP, tcp.DstPort, vlan)
}
