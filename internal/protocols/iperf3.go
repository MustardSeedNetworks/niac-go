package protocols

import (
	"encoding/binary"
	"encoding/json"
	"sync"
	"time"

	"github.com/gopacket/gopacket/layers"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/safeconv"
)

// TCPPortIPerf3 is the port an iperf3 server listens on.
const TCPPortIPerf3 = 5201

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
)

// iperf3Server serves iperf3 forward TCP tests: the client sends, the
// simulated server receives and counts. It speaks the control channel and
// acknowledges the data streams; it never sends test data, so a reverse,
// bidirectional or UDP test is refused with SERVER_ERROR.
type iperf3Server struct {
	mu    sync.Mutex
	conns map[streamKey]*iperf3Conn
	tests map[string]*iperf3Test // by cookie
}

type iperf3Conn struct {
	clientNext uint32 // next in-order byte from the client
	serverNext uint32 // sequence number of our next byte
	pending    []byte // control bytes not yet consumed
	test       *iperf3Test
	data       bool
	received   uint64 // data-stream bytes after the cookie
	lastSeen   time.Time
}

type iperf3Test struct {
	control  streamKey
	state    byte // the last state sent on the control connection
	streams  int
	data     []*iperf3Conn
	dataKeys []streamKey
	started  time.Time
	ended    time.Time
	lastSeen time.Time
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
	return &iperf3Server{conns: make(map[streamKey]*iperf3Conn), tests: make(map[string]*iperf3Test)}
}

// handleIPerf3 serves port 5201 on a device with iperf3 enabled and resets
// a SYN to any other.
func (h *TCPHandler) handleIPerf3(pkt *Packet, ipLayer *layers.IPv4, tcp *layers.TCP, devices []*config.Device) {
	device := h.serviceDevice(devices, ipLayer.DstIP, func(device *config.Device) bool {
		return device.IPerf3 != nil && device.IPerf3.Enabled
	})
	if device == nil {
		if tcp.SYN && !tcp.ACK {
			h.sendRST(pkt, ipLayer, tcp, devices)
		}
		return
	}
	segments := h.iperf3.receive(newStreamKey(ipLayer, tcp), tcp, iperf3SegmentLength(ipLayer, tcp), time.Now())
	identity, ok := h.stack.replyEthernet(pkt, device)
	if !ok {
		return
	}
	for _, segment := range segments {
		if !h.sendStatelessFrame(pkt, ipLayer, segment.tcp, segment.payload, device, identity, "iPerf3") {
			return
		}
	}
}

// iperf3SegmentLength is how much data a segment carries by its IP header.
// The capture keeps 1,600 bytes of a frame, and a GRO or TSO superframe
// handed up a veth is longer: a data stream is a sink, so what arrived is
// what the header says, not what was kept.
func iperf3SegmentLength(ipLayer *layers.IPv4, tcp *layers.TCP) int {
	length := int(ipLayer.Length) - int(ipLayer.IHL)*ipv4IHLBytes - int(tcp.DataOffset)*ipv4IHLBytes

	return max(length, len(tcp.Payload))
}

// receive takes one client segment carrying length bytes and returns what
// the server sends in answer, which may be on another connection of the
// same test.
func (s *iperf3Server) receive(key streamKey, tcp *layers.TCP, length int, now time.Time) []iperf3Segment {
	s.mu.Lock()
	defer s.mu.Unlock()
	switch conn := s.conns[key]; {
	case tcp.SYN && !tcp.ACK:
		return s.open(key, tcp, now)
	case conn == nil:
		// A connection the server does not know: closed, forgotten, or from
		// before a restart. Only what would expect an answer is reset.
		if tcp.RST || (length == 0 && !tcp.FIN) {
			return nil
		}
		return []iperf3Segment{{tcp: &layers.TCP{
			SrcPort: layers.TCPPort(key.server.Port()), DstPort: layers.TCPPort(key.client.Port()),
			Seq: tcp.Ack, RST: true, ACK: true, Ack: tcp.Seq + safeconv.Uint32(length),
		}}}
	case tcp.RST:
		s.drop(key, conn)
		return nil
	default:
		return s.segment(key, conn, tcp, length, now)
	}
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
	s.conns[key] = &iperf3Conn{clientNext: tcp.Seq + 1, serverNext: tcpInitialSeq + 1, lastSeen: now}

	return []iperf3Segment{{tcp: &layers.TCP{
		SrcPort: layers.TCPPort(
			key.server.Port(),
		),
		DstPort: layers.TCPPort(key.client.Port()),
		Seq:     tcpInitialSeq,
		Ack:     tcp.Seq + 1,
		SYN:     true,
		ACK:     true,
		Window:  tcpWindowSize,
		Options: []layers.TCPOption{{
			OptionType: layers.TCPOptionKindMSS, OptionLength: tcpOptionMSSLen,
			OptionData: binary.BigEndian.AppendUint16(nil, iperf3MSS),
		}},
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
	now time.Time,
) []iperf3Segment {
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
		out = s.advance(key, conn)
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
func (s *iperf3Server) advance(key streamKey, conn *iperf3Conn) []iperf3Segment {
	if conn.test == nil {
		return s.cookie(key, conn)
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
func (s *iperf3Server) cookie(key streamKey, conn *iperf3Conn) []iperf3Segment {
	if len(conn.pending) < iperf3CookieLen {
		return nil
	}
	cookie := string(conn.pending[:iperf3CookieLen])
	rest := conn.pending[iperf3CookieLen:]
	conn.pending = nil
	if test := s.tests[cookie]; test != nil {
		return s.join(key, conn, test, rest)
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
	if params.UDP || params.Reverse || params.Bidirectional {
		return s.refuse(key, conn, true)
	}
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
// stream the parameters asked for has arrived.
func (s *iperf3Server) join(key streamKey, conn *iperf3Conn, test *iperf3Test, rest []byte) []iperf3Segment {
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

// results are the server's results: what each stream received, under the
// stream ids the client numbered them with.
func (t *iperf3Test) results(client iperf3Results) iperf3Results {
	seconds := t.ended.Sub(t.started).Seconds()
	results := iperf3Results{SenderHasRetransmits: -1, Streams: make([]iperf3StreamResult, 0, len(t.data))}
	for i, conn := range t.data {
		id := i + 1
		if i < len(client.Streams) {
			id = client.Streams[i].ID
		}
		results.Streams = append(results.Streams, iperf3StreamResult{
			ID: id, Bytes: conn.received, Retransmits: -1, EndTime: seconds,
		})
	}

	return results
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
// connection goes is over.
func (s *iperf3Server) drop(key streamKey, conn *iperf3Conn) {
	if conn.test != nil && conn.test.control == key {
		s.finish(conn.test)
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
