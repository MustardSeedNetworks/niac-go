package protocols

import (
	"crypto/rand"
	"encoding/binary"
	"encoding/json"
	"net"
	"sync"
	"time"

	"github.com/gopacket/gopacket/layers"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/safeconv"
)

// TCPPortIPerf3 is the port an iperf3 server listens on unless authored.
const TCPPortIPerf3 = config.DefaultIPerf3Port

// Control-channel states, one byte each (iperf_api.h). SERVER_ERROR is -2
// and is followed by the iperf3 error number and errno, four bytes each.
const (
	iperf3TestStart       byte = 1
	iperf3TestRunning     byte = 2
	iperf3TestEnd         byte = 4
	iperf3ParamExchange   byte = 9
	iperf3CreateStreams   byte = 10
	iperf3ClientTerminate byte = 12
	iperf3ExchangeResults byte = 13
	iperf3DisplayResults  byte = 14
	iperf3Done            byte = 16
	iperf3ServerError     byte = 0xfe
)

// iperf3ErrUnimplemented is IEUNIMP, which the client reports as an option
// that is not implemented.
const iperf3ErrUnimplemented = 13

const (
	// iperf3CookieLen is the cookie that opens every connection of a test:
	// 36 characters and a NUL. The control connection is the first to send
	// it; the data streams send the same one.
	iperf3CookieLen = 37
	// iperf3MaxMessage bounds a length-prefixed JSON message: real
	// parameters and results are a few hundred bytes.
	iperf3MaxMessage = 64 << 10
	// iperf3MaxStreams is iperf3's own MAX_STREAMS.
	iperf3MaxStreams = 128
	// iperf3MaxConnections bounds the connections held at once.
	iperf3MaxConnections = 256
	// iperf3IdleTimeout forgets a test, or a connection that never joined
	// one, after this long without a segment.
	iperf3IdleTimeout = time.Minute
	// iperf3MSS is the segment size the SYN-ACK advertises, the IPv4 MSS of
	// a 1500-byte Ethernet MTU. Without it the client falls back to 536.
	iperf3MSS = 1460
	// iperf3LengthPrefix is the big-endian length before a JSON message.
	iperf3LengthPrefix = 4
	// tcpOptionMSSLen is the length of an MSS option: kind, length, value.
	tcpOptionMSSLen = 4
	// tcpOptionWindowScaleLen is the length of a window scale option.
	tcpOptionWindowScaleLen = 3
	// tcpMaxWindowShift is the largest window scale RFC 7323 allows.
	tcpMaxWindowShift = 14
	// tcpDefaultMSS is the MSS of a client that announces none (RFC 9293).
	tcpDefaultMSS = 536
	// iperf3WindowShift is the window scale the server offers a client that
	// offers one: its 65,535-byte window then stands for 8 MiB, so a forward
	// stream is not bounded by the window at lab round-trip times.
	iperf3WindowShift = 7
)

// The reverse sender. It paces each test at the device's download rate and
// keeps no more in flight than the client's window. Three duplicate ACKs
// resend the segment the client lacks and, until everything sent before
// them is acknowledged, the next one on each partial ACK (NewReno, RFC
// 6582); a retransmission timeout sends again from the first byte the
// client has not acknowledged.
const (
	// iperf3SendInterval is how often the sender runs.
	iperf3SendInterval = time.Millisecond
	// iperf3SendBurst bounds the bytes a late run may send at once.
	iperf3SendBurst = 64 << 10
	// iperf3RetransmitTimeout is RFC 6298's RTO for a sender with no round-
	// trip estimate. The simulator's own receive path can hold an ACK back
	// for well over Linux's 200 ms minimum, which would fire spuriously.
	iperf3RetransmitTimeout = time.Second
	// iperf3DuplicateACKs is the fast-retransmit threshold.
	iperf3DuplicateACKs = 3
	// iperf3SenderStall ends a reverse test whose client has sent nothing,
	// not even an ACK, for this long: it is gone, or was never there.
	iperf3SenderStall = 3 * time.Second
	// bytesPerMbps converts the authored rate in megabits to bytes.
	bytesPerMbps = 1e6 / 8
)

// iperf3Server serves iperf3 TCP tests. In a forward test the client sends
// and the server receives and counts; in a reverse test, the downstream leg
// of a NetAlly tester's default test, the server sends. It refuses a
// bidirectional or UDP test with SERVER_ERROR.
type iperf3Server struct {
	mu     sync.Mutex
	conns  map[streamKey]*iperf3Conn
	tests  map[string]*iperf3Test // by cookie
	filler []byte                 // the payload of the data the server sends
	// isn picks a connection's initial sequence number. It is random so a
	// spoofed client cannot complete a handshake blind and aim a reverse
	// test at the address it spoofed.
	isn func() uint32
}

type iperf3Conn struct {
	clientNext uint32 // next in-order byte from the client
	serverNext uint32 // sequence number of our next byte
	pending    []byte // control bytes not yet consumed
	test       *iperf3Test
	data       bool
	received   uint64 // data-stream bytes after the cookie
	lastSeen   time.Time
	// established is set once the client acknowledges our ISN, which a
	// spoofed source never sees.
	established bool

	mss    uint32 // the largest segment the client accepts
	shift  uint8  // the client's window scale, when both offered one
	window uint32 // the client's receive window in bytes

	// A reverse test's data stream sends from serverNext.
	first        uint32 // our first data byte
	unacked      uint32 // the first byte the client has not acknowledged
	sent         uint32 // one past the highest byte sent
	duplicates   int    // duplicate ACKs in a row
	retransmits  int    // segments sent again
	lastProgress time.Time
	recovering   bool   // resending what three duplicate ACKs reported lost
	recover      uint32 // recovery ends once the client acknowledges this
	resend       bool   // the segment at unacked is due again
}

type iperf3Test struct {
	control  streamKey
	state    byte // the last state sent on the control connection
	streams  int
	reverse  bool
	data     []*iperf3Conn
	dataKeys []streamKey
	started  time.Time
	ended    time.Time
	lastSeen time.Time

	route    iperf3Route // where a reverse test's data goes
	rate     float64     // bytes a second a reverse test sends
	budget   float64     // bytes the rate allows now
	lastSend time.Time
}

// iperf3Route is how the server reaches a test's client: the device that
// serves it and the frame identity of the client's segments.
type iperf3Route struct {
	device         *config.Device
	identity       replyEthernetIdentity
	client, server net.IP
}

// iperf3Burst is what the sender sends to one test's client.
type iperf3Burst struct {
	route    iperf3Route
	segments []iperf3Segment
}

// iperf3Params are the parameters the server acts on.
type iperf3Params struct {
	UDP           bool `json:"udp"`
	Reverse       bool `json:"reverse"`
	Bidirectional bool `json:"bidirectional"`
	Parallel      int  `json:"parallel"`
}

// iperf3Results is the results message, with every field the client
// requires of it.
type iperf3Results struct {
	CPUUtilTotal         float64              `json:"cpu_util_total"`
	CPUUtilUser          float64              `json:"cpu_util_user"`
	CPUUtilSystem        float64              `json:"cpu_util_system"`
	SenderHasRetransmits int                  `json:"sender_has_retransmits"`
	CongestionUsed       string               `json:"congestion_used,omitempty"`
	Streams              []iperf3StreamResult `json:"streams"`
}

type iperf3StreamResult struct {
	ID             int     `json:"id"`
	Bytes          uint64  `json:"bytes"`
	Retransmits    int     `json:"retransmits"`
	Jitter         float64 `json:"jitter"`
	Errors         int     `json:"errors"`
	OmittedErrors  int     `json:"omitted_errors"`
	Packets        int     `json:"packets"`
	OmittedPackets int     `json:"omitted_packets"`
	StartTime      float64 `json:"start_time"`
	EndTime        float64 `json:"end_time"`
}

// iperf3Segment is one segment to send once the server's lock is released.
type iperf3Segment struct {
	tcp     *layers.TCP
	payload []byte
}

func newIPerf3Server() *iperf3Server {
	return &iperf3Server{
		conns: make(map[streamKey]*iperf3Conn), tests: make(map[string]*iperf3Test), filler: make([]byte, iperf3MSS),
		isn: iperf3RandomISN,
	}
}

// iperf3RandomISN is an unpredictable initial sequence number.
func iperf3RandomISN() uint32 {
	var isn [4]byte
	_, _ = rand.Read(isn[:]) // crypto/rand.Read never fails

	return binary.BigEndian.Uint32(isn[:])
}

// iperf3Device is the device that serves iperf3 on port at ip, if any.
func (h *TCPHandler) iperf3Device(devices []*config.Device, ip net.IP, port layers.TCPPort) *config.Device {
	return h.serviceDevice(devices, ip, func(device *config.Device) bool {
		return device.IPerf3 != nil && device.IPerf3.Enabled && layers.TCPPort(device.IPerf3.Port) == port
	})
}

// handleIPerf3 serves the iperf3 port of a device with iperf3 enabled and
// resets a SYN to any other.
func (h *TCPHandler) handleIPerf3(pkt *Packet, ipLayer *layers.IPv4, tcp *layers.TCP, devices []*config.Device) {
	device := h.iperf3Device(devices, ipLayer.DstIP, tcp.DstPort)
	if device == nil {
		if tcp.SYN && !tcp.ACK {
			h.sendRST(pkt, ipLayer, tcp, devices)
		}
		return
	}
	identity, ok := h.stack.replyEthernet(pkt, device)
	if !ok {
		return
	}
	route := iperf3Route{device: device, identity: identity, client: ipLayer.SrcIP, server: ipLayer.DstIP}
	segments, started := h.iperf3.receive(
		newStreamKey(ipLayer, tcp), tcp, iperf3SegmentLength(ipLayer, tcp), route, time.Now(),
	)
	h.sendIPerf3(pkt.SerialNumber, iperf3Burst{route: route, segments: segments})
	if started {
		h.startIPerf3Sender()
	}
}

// sendIPerf3 sends a burst of segments to a test's client.
func (h *TCPHandler) sendIPerf3(serial int, burst iperf3Burst) {
	ipLayer := &layers.IPv4{SrcIP: burst.route.client, DstIP: burst.route.server}
	for _, segment := range burst.segments {
		if !h.sendStatelessFrame(
			serial, ipLayer, segment.tcp, segment.payload, burst.route.device, burst.route.identity, "iPerf3",
		) {
			return
		}
	}
}

// runIPerf3Sender sends reverse tests' data until none is running or the
// stack stops. One sender serves every test of the handler.
func (h *TCPHandler) runIPerf3Sender() {
	if !h.iperf3Sending.CompareAndSwap(false, true) {
		return
	}
	go func() {
		ticker := time.NewTicker(iperf3SendInterval)
		defer ticker.Stop()
		for {
			select {
			case <-h.stack.stopChan:
				h.iperf3Sending.Store(false)
				return
			case now := <-ticker.C:
				if h.pumpIPerf3(now) {
					continue
				}
				// A test may start between the last run and giving up the
				// sender: it found the flag set and left the work to us.
				h.iperf3Sending.Store(false)
				if !h.iperf3.sending() || !h.iperf3Sending.CompareAndSwap(false, true) {
					return
				}
			}
		}
	}()
}

// pumpIPerf3 sends what the reverse tests have due at now and reports
// whether any is still running.
func (h *TCPHandler) pumpIPerf3(now time.Time) bool {
	bursts, running := h.iperf3.tick(now)
	for _, burst := range bursts {
		h.sendIPerf3(0, burst)
	}

	return running
}

// iperf3SegmentLength is how much data a segment carries by its IP header.
// The capture keeps 1,600 bytes of a frame, and a GRO or TSO superframe
// handed up a veth is longer: a data stream is a sink, so what arrived is
// what the header says, not what was kept.
func iperf3SegmentLength(ipLayer *layers.IPv4, tcp *layers.TCP) int {
	length := int(ipLayer.Length) - int(ipLayer.IHL)*ipv4IHLBytes - int(tcp.DataOffset)*ipv4IHLBytes

	return max(length, len(tcp.Payload))
}

// receive takes one client segment carrying length bytes, which reached
// the server by route, and returns what the server sends in answer, which
// may be on another connection of the same test. It reports whether the
// segment started a reverse test, which the sender must now serve.
func (s *iperf3Server) receive(
	key streamKey,
	tcp *layers.TCP,
	length int,
	route iperf3Route,
	now time.Time,
) ([]iperf3Segment, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	switch conn := s.conns[key]; {
	case tcp.SYN && !tcp.ACK:
		return s.open(key, tcp, now), false
	case conn == nil:
		// A connection the server does not know: closed, forgotten, or from
		// before a restart. Only what would expect an answer is reset.
		if tcp.RST || (length == 0 && !tcp.FIN) {
			return nil, false
		}
		return []iperf3Segment{{tcp: &layers.TCP{
			SrcPort: layers.TCPPort(key.server.Port()), DstPort: layers.TCPPort(key.client.Port()),
			Seq: tcp.Ack, RST: true, ACK: true, Ack: tcp.Seq + safeconv.Uint32(length),
		}}}, false
	case tcp.RST:
		// Only a reset at the next expected byte is the client's (RFC 5961).
		if tcp.Seq == conn.clientNext {
			s.drop(key, conn)
		}
		return nil, false
	case !conn.established && (!tcp.ACK || tcp.Ack != conn.serverNext):
		return nil, false
	default:
		running := conn.test != nil && conn.test.state == iperf3TestRunning
		conn.established = true
		segments := s.segment(key, conn, tcp, length, route, now)
		test := conn.test

		return segments, !running && test != nil && test.reverse && test.state == iperf3TestRunning
	}
}

// sending reports whether a reverse test is running.
func (s *iperf3Server) sending() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, test := range s.tests {
		if test.reverse && test.state == iperf3TestRunning {
			return true
		}
	}

	return false
}

// open answers a SYN, replacing any earlier connection on the same ports.
func (s *iperf3Server) open(key streamKey, tcp *layers.TCP, now time.Time) []iperf3Segment {
	s.expire(now)
	if old := s.conns[key]; old != nil {
		s.drop(key, old)
	}
	if len(s.conns) >= iperf3MaxConnections {
		return []iperf3Segment{{tcp: &layers.TCP{
			SrcPort: layers.TCPPort(
				key.server.Port(),
			),
			DstPort: layers.TCPPort(key.client.Port()),
			RST:     true,
			ACK:     true,
			Ack:     tcp.Seq + 1,
		}}}
	}
	isn := s.isn()
	conn := &iperf3Conn{
		clientNext: tcp.Seq + 1, serverNext: isn + 1, lastSeen: now,
		mss: tcpDefaultMSS, window: uint32(tcp.Window),
	}
	options := []layers.TCPOption{{
		OptionType: layers.TCPOptionKindMSS, OptionLength: tcpOptionMSSLen,
		OptionData: binary.BigEndian.AppendUint16(nil, iperf3MSS),
	}}
	for _, option := range tcp.Options {
		switch {
		case option.OptionType == layers.TCPOptionKindMSS && len(option.OptionData) == 2:
			conn.mss = uint32(binary.BigEndian.Uint16(option.OptionData))
		case option.OptionType == layers.TCPOptionKindWindowScale && len(option.OptionData) == 1:
			conn.shift = min(option.OptionData[0], tcpMaxWindowShift)
			options = append(options, layers.TCPOption{OptionType: layers.TCPOptionKindNop},
				layers.TCPOption{
					OptionType: layers.TCPOptionKindWindowScale, OptionLength: tcpOptionWindowScaleLen,
					OptionData: []byte{iperf3WindowShift},
				})
		}
	}
	// Every host accepts a 536-byte segment, so a smaller MSS buys the client
	// nothing but more packets for the same rate.
	conn.mss = min(max(conn.mss, tcpDefaultMSS), iperf3MSS)
	s.conns[key] = conn

	return []iperf3Segment{{tcp: &layers.TCP{
		SrcPort: layers.TCPPort(
			key.server.Port(),
		),
		DstPort: layers.TCPPort(key.client.Port()),
		Seq:     isn,
		Ack:     tcp.Seq + 1,
		SYN:     true,
		ACK:     true,
		Window:  tcpWindowSize,
		Options: options,
	}}}
}

// segment takes a segment carrying length bytes on an open connection. Data
// in order is consumed and acknowledged; anything else gets a duplicate ACK
// so the client sends the gap again.
func (s *iperf3Server) segment(
	key streamKey,
	conn *iperf3Conn,
	tcp *layers.TCP,
	length int,
	route iperf3Route,
	now time.Time,
) []iperf3Segment {
	conn.window = uint32(tcp.Window) << conn.shift
	if conn.sent != conn.first {
		conn.acknowledged(tcp, length, now)
	}
	if length == 0 && !tcp.FIN {
		return nil // an ACK of ours
	}
	conn.lastSeen = now
	if conn.test != nil {
		conn.test.lastSeen = now
	}
	if tcp.Seq != conn.clientNext {
		return []iperf3Segment{s.ack(key, conn, nil, false)}
	}
	conn.clientNext += safeconv.Uint32(len(tcp.Payload))
	var out []iperf3Segment
	if conn.data {
		conn.received += uint64(len(tcp.Payload))
	} else {
		conn.pending = append(conn.pending, tcp.Payload...)
		out = s.advance(key, conn, route)
	}
	if s.conns[key] != conn {
		return out // the conversation ended and the connection was reset
	}
	// Bytes the capture cut off a superframe can only be test data: a data
	// stream counts them, and anything else cannot be read without them.
	if uncaptured := length - len(tcp.Payload); uncaptured > 0 {
		if !conn.data {
			return append(out, s.refuse(key, conn, false)...)
		}
		conn.clientNext += safeconv.Uint32(uncaptured)
		conn.received += uint64(uncaptured)
	}
	if tcp.FIN {
		conn.clientNext++
		s.drop(key, conn)
		return append(out, s.ack(key, conn, nil, true))
	}
	if len(out) == 0 || out[0].tcp.DstPort != layers.TCPPort(key.client.Port()) {
		out = append([]iperf3Segment{s.ack(key, conn, nil, false)}, out...)
	}

	return out
}

// advance consumes what a control connection, or a connection that has not
// sent its cookie yet, has received, and returns the server's answer.
func (s *iperf3Server) advance(key streamKey, conn *iperf3Conn, route iperf3Route) []iperf3Segment {
	if conn.test == nil {
		return s.cookie(key, conn, route)
	}
	switch conn.test.state {
	case iperf3ParamExchange:
		return s.params(key, conn)
	case iperf3TestRunning, iperf3DisplayResults:
		return s.clientState(key, conn)
	case iperf3ExchangeResults:
		return s.exchangeResults(key, conn)
	default:
		// The client sends nothing on the control connection while the
		// server waits for its streams.
		return s.refuse(key, conn, false)
	}
}

// cookie reads the cookie that opens a connection: the first connection to
// send it is a test's control connection, the later ones its data streams.
func (s *iperf3Server) cookie(key streamKey, conn *iperf3Conn, route iperf3Route) []iperf3Segment {
	if len(conn.pending) < iperf3CookieLen {
		return nil
	}
	cookie := string(conn.pending[:iperf3CookieLen])
	rest := conn.pending[iperf3CookieLen:]
	conn.pending = nil
	if test := s.tests[cookie]; test != nil {
		return s.join(key, conn, test, rest, route)
	}
	test := &iperf3Test{control: key, state: iperf3ParamExchange, lastSeen: conn.lastSeen}
	s.tests[cookie] = test
	conn.test = test
	conn.pending = append(conn.pending, rest...)

	return []iperf3Segment{s.ack(key, conn, []byte{iperf3ParamExchange}, false)}
}

// params reads the test parameters and asks for the data streams.
func (s *iperf3Server) params(key streamKey, conn *iperf3Conn) []iperf3Segment {
	message, ok := s.message(conn)
	switch {
	case !ok:
		return s.refuse(key, conn, false)
	case message == nil:
		return nil
	}
	var params iperf3Params
	if err := json.Unmarshal(message, &params); err != nil {
		return s.refuse(key, conn, false)
	}
	if params.UDP || params.Bidirectional {
		return s.refuse(key, conn, true)
	}
	conn.test.reverse = params.Reverse
	conn.test.streams = min(max(params.Parallel, 1), iperf3MaxStreams)
	conn.test.state = iperf3CreateStreams

	return []iperf3Segment{s.ack(key, conn, []byte{iperf3CreateStreams}, false)}
}

// clientState reads the one-byte state the client sends to end the test or,
// after the results, to finish.
func (s *iperf3Server) clientState(key streamKey, conn *iperf3Conn) []iperf3Segment {
	if len(conn.pending) == 0 {
		return nil
	}
	test, state := conn.test, conn.pending[0]
	conn.pending = conn.pending[1:]
	switch {
	case state == iperf3TestEnd && test.state == iperf3TestRunning:
		test.ended = conn.lastSeen
		test.state = iperf3ExchangeResults

		return []iperf3Segment{s.ack(key, conn, []byte{iperf3ExchangeResults}, false)}
	case state == iperf3Done && test.state == iperf3DisplayResults, state == iperf3ClientTerminate:
		s.finish(test)
		return nil
	default:
		return s.refuse(key, conn, false)
	}
}

// exchangeResults reads the client's results and answers with the server's
// and DISPLAY_RESULTS.
func (s *iperf3Server) exchangeResults(key streamKey, conn *iperf3Conn) []iperf3Segment {
	message, ok := s.message(conn)
	switch {
	case !ok:
		return s.refuse(key, conn, false)
	case message == nil:
		return nil
	}
	var client iperf3Results
	if err := json.Unmarshal(message, &client); err != nil {
		return s.refuse(key, conn, false)
	}
	results, err := json.Marshal(conn.test.results(client))
	if err != nil {
		return s.refuse(key, conn, false)
	}
	conn.test.state = iperf3DisplayResults
	answer := binary.BigEndian.AppendUint32(nil, safeconv.Uint32(len(results)))
	answer = append(append(answer, results...), iperf3DisplayResults)

	return []iperf3Segment{s.ack(key, conn, answer, false)}
}

// join adds a data stream to its test, and starts the test once every
// stream the parameters asked for has arrived. A reverse test's data goes
// back the way the last stream came.
func (s *iperf3Server) join(
	key streamKey,
	conn *iperf3Conn,
	test *iperf3Test,
	rest []byte,
	route iperf3Route,
) []iperf3Segment {
	if test.state != iperf3CreateStreams || key == test.control {
		s.close(key, conn)
		return []iperf3Segment{s.reset(key, conn)}
	}
	conn.test, conn.data, conn.received = test, true, uint64(len(rest))
	test.data = append(test.data, conn)
	test.dataKeys = append(test.dataKeys, key)
	if len(test.data) < test.streams {
		return nil
	}
	control := s.conns[test.control]
	if control == nil || control.test != test {
		s.finish(test)
		s.close(key, conn)
		return []iperf3Segment{s.reset(key, conn)}
	}
	test.state = iperf3TestRunning
	test.started = conn.lastSeen
	if test.reverse {
		test.route, test.lastSend = route, test.started
		test.rate = config.DefaultIPerf3DownloadMbps
		if rate := route.device.IPerf3.DownloadMbps; rate > 0 {
			test.rate = rate
		}
		test.rate *= bytesPerMbps
		for _, stream := range test.data {
			stream.first, stream.unacked, stream.sent = stream.serverNext, stream.serverNext, stream.serverNext
			stream.lastProgress = test.started
		}
	}

	return []iperf3Segment{
		s.ack(key, conn, nil, false),
		s.ack(test.control, control, []byte{iperf3TestStart, iperf3TestRunning}, false),
	}
}

// message takes one length-prefixed JSON message off conn's pending bytes.
// It returns nil, true while the message is incomplete and nil, false for a
// length no message can have.
func (s *iperf3Server) message(conn *iperf3Conn) ([]byte, bool) {
	if len(conn.pending) < iperf3LengthPrefix {
		return nil, true
	}
	length := binary.BigEndian.Uint32(conn.pending)
	if length == 0 || length > iperf3MaxMessage {
		return nil, false
	}
	end := iperf3LengthPrefix + int(length)
	if len(conn.pending) < end {
		return nil, true
	}
	message := conn.pending[iperf3LengthPrefix:end]
	conn.pending = conn.pending[end:]

	return message, true
}

// refuse ends a test the server cannot serve. A test it understood but does
// not implement gets SERVER_ERROR with IEUNIMP, which the client reports;
// anything else is reset.
func (s *iperf3Server) refuse(key streamKey, conn *iperf3Conn, unimplemented bool) []iperf3Segment {
	s.finish(conn.test)
	s.close(key, conn)
	if !unimplemented {
		return []iperf3Segment{s.reset(key, conn)}
	}
	answer := []byte{iperf3ServerError}
	answer = binary.BigEndian.AppendUint32(answer, iperf3ErrUnimplemented)
	answer = binary.BigEndian.AppendUint32(answer, 0)

	return []iperf3Segment{s.ack(key, conn, answer, true)}
}

// results are the server's results, under the stream ids the client
// numbered them with: what each stream received or, in a reverse test, what
// it sent and how many segments it sent again.
func (t *iperf3Test) results(client iperf3Results) iperf3Results {
	seconds := t.ended.Sub(t.started).Seconds()
	results := iperf3Results{SenderHasRetransmits: -1, Streams: make([]iperf3StreamResult, 0, len(t.data))}
	if t.reverse {
		results.SenderHasRetransmits = 1
	}
	for i, conn := range t.data {
		id := i + 1
		if i < len(client.Streams) {
			id = client.Streams[i].ID
		}
		stream := iperf3StreamResult{ID: id, Bytes: conn.received, Retransmits: -1, EndTime: seconds}
		if t.reverse {
			stream.Bytes, stream.Retransmits = uint64(conn.sent-conn.first), conn.retransmits
		}
		results.Streams = append(results.Streams, stream)
	}

	return results
}

// tick returns the data each running reverse test has due at now and
// reports whether any is still running.
func (s *iperf3Server) tick(now time.Time) ([]iperf3Burst, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var bursts []iperf3Burst
	running := false
	for _, test := range s.tests {
		if !test.reverse || test.state != iperf3TestRunning {
			continue
		}
		if now.Sub(test.lastSeen) > iperf3SenderStall {
			s.finish(test)
			continue
		}
		running = true
		if segments := test.send(now, s.filler); len(segments) > 0 {
			bursts = append(bursts, iperf3Burst{route: test.route, segments: segments})
		}
	}

	return bursts, running
}

// send returns the segments the rate and the client's windows allow at now,
// one stream after another so parallel streams share the rate.
func (t *iperf3Test) send(now time.Time, filler []byte) []iperf3Segment {
	t.budget = min(t.budget+t.rate*now.Sub(t.lastSend).Seconds(), iperf3SendBurst)
	t.lastSend = now
	for _, conn := range t.data {
		if conn.sent != conn.unacked && now.Sub(conn.lastProgress) > iperf3RetransmitTimeout {
			conn.serverNext, conn.lastProgress = conn.unacked, now
			conn.recovering, conn.resend = false, false
		}
	}
	var out []iperf3Segment
	for more := true; more; {
		more = false
		for i, conn := range t.data {
			switch {
			case float64(conn.mss) > t.budget:
				continue
			case conn.resend:
				conn.resend = false
				out = append(out, conn.segmentAt(t.dataKeys[i], conn.unacked, filler))
			case conn.serverNext-conn.unacked+conn.mss > conn.window:
				continue
			default:
				out = append(out, conn.next(t.dataKeys[i], filler, now))
			}
			t.budget -= float64(conn.mss)
			more = true
		}
	}

	return out
}

// next is the data segment at serverNext.
func (c *iperf3Conn) next(key streamKey, filler []byte, now time.Time) iperf3Segment {
	if c.serverNext == c.unacked && c.sent == c.unacked {
		c.lastProgress = now // the first byte in flight starts the timer
	}
	segment := c.segmentAt(key, c.serverNext, filler)
	c.serverNext += c.mss
	if seqBefore(c.sent, c.serverNext) {
		c.sent = c.serverNext
	}

	return segment
}

// segmentAt is the data segment at seq, counted as a retransmission when
// that data has been sent before.
func (c *iperf3Conn) segmentAt(key streamKey, seq uint32, filler []byte) iperf3Segment {
	if seqBefore(seq, c.sent) {
		c.retransmits++
	}

	return iperf3Segment{tcp: &layers.TCP{
		SrcPort: layers.TCPPort(key.server.Port()), DstPort: layers.TCPPort(key.client.Port()),
		Seq: seq, Ack: c.clientNext, ACK: true, Window: tcpWindowSize,
	}, payload: filler[:c.mss]}
}

// acknowledged takes the client's ACK of a stream the server sends on. New
// data acknowledged restarts the timer and, in recovery, a partial ACK marks
// the next missing segment due; the third duplicate starts recovery.
func (c *iperf3Conn) acknowledged(tcp *layers.TCP, length int, now time.Time) {
	switch {
	case !tcp.ACK || seqBefore(c.sent, tcp.Ack) || seqBefore(tcp.Ack, c.unacked):
		return // not an acknowledgement of anything in flight
	case seqBefore(c.unacked, tcp.Ack):
		c.unacked, c.duplicates, c.lastProgress = tcp.Ack, 0, now
		if seqBefore(c.serverNext, c.unacked) {
			c.serverNext = c.unacked
		}
		if c.recovering {
			c.recovering = seqBefore(c.unacked, c.recover)
			c.resend = c.recovering
		}
	case tcp.Ack == c.unacked && length == 0 && c.sent != c.unacked:
		c.duplicates++
		if c.duplicates == iperf3DuplicateACKs && !c.recovering {
			c.recovering, c.recover, c.resend = true, c.sent, true
		}
	}
	c.lastSeen = now
	if c.test != nil {
		c.test.lastSeen = now
	}
}

// seqBefore reports whether sequence number a comes before b, modulo 2^32.
func seqBefore(a, b uint32) bool {
	return a != b && b-a < 1<<31
}

// finish forgets a test and every data stream it opened. The control
// connection stays open for the client to close, outside any test.
func (s *iperf3Server) finish(test *iperf3Test) {
	if test == nil {
		return
	}
	if control := s.conns[test.control]; control != nil && control.test == test {
		control.test = nil
	}
	for cookie, candidate := range s.tests {
		if candidate == test {
			delete(s.tests, cookie)
		}
	}
	for i, key := range test.dataKeys {
		if s.conns[key] == test.data[i] {
			delete(s.conns, key)
		}
	}
}

// drop forgets a connection the client closed or reset. A test whose control
// connection goes is over, and so is a running reverse test that loses a
// stream the server sends on.
func (s *iperf3Server) drop(key streamKey, conn *iperf3Conn) {
	if test := conn.test; test != nil &&
		(test.control == key || test.reverse && conn.data && test.state == iperf3TestRunning) {
		s.finish(test)
	}
	s.close(key, conn)
}

func (s *iperf3Server) close(key streamKey, conn *iperf3Conn) {
	if s.conns[key] == conn {
		delete(s.conns, key)
	}
}

// expire forgets tests, and connections outside a test, that have been
// silent for iperf3IdleTimeout. A test's control connection is quiet while
// its streams run, so a test is as fresh as its newest segment.
func (s *iperf3Server) expire(now time.Time) {
	for _, test := range s.tests {
		if now.Sub(test.lastSeen) > iperf3IdleTimeout {
			s.close(test.control, s.conns[test.control])
			s.finish(test)
		}
	}
	for key, conn := range s.conns {
		if conn.test == nil && now.Sub(conn.lastSeen) > iperf3IdleTimeout {
			delete(s.conns, key)
		}
	}
}

// ack is our next segment on conn: an acknowledgement of everything the
// client has sent, carrying payload when there is one and closing our side
// when fin is set.
func (s *iperf3Server) ack(key streamKey, conn *iperf3Conn, payload []byte, fin bool) iperf3Segment {
	segment := iperf3Segment{tcp: &layers.TCP{
		SrcPort: layers.TCPPort(key.server.Port()), DstPort: layers.TCPPort(key.client.Port()),
		Seq: conn.serverNext, Ack: conn.clientNext, ACK: true, PSH: len(payload) > 0, FIN: fin,
		Window: tcpWindowSize,
	}, payload: payload}
	conn.serverNext += safeconv.Uint32(len(payload))
	if fin {
		conn.serverNext++
	}

	return segment
}

func (s *iperf3Server) reset(key streamKey, conn *iperf3Conn) iperf3Segment {
	return iperf3Segment{tcp: &layers.TCP{
		SrcPort: layers.TCPPort(key.server.Port()), DstPort: layers.TCPPort(key.client.Port()),
		Seq: conn.serverNext, Ack: conn.clientNext, RST: true, ACK: true,
	}}
}
