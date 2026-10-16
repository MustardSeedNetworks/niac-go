package protocols

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"net"
	"strings"
	"testing"

	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/layers"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/safeconv"
)

// iperf3TestCookie is a client cookie: 36 characters and a NUL, as iperf3
// 3.21 sends it.
func iperf3TestCookie() []byte {
	return append([]byte("dgdterct3nbes4n5754y4tagwvdovv57p5lg"), 0)
}

const (
	iperf3TestControlPort layers.TCPPort = 40001
	iperf3TestDataPort    layers.TCPPort = 40002
	iperf3TestVLAN                       = 203
)

func iperf3TestServer(iperf3 *config.IPerf3Config) *config.Device {
	return &config.Device{
		Name:        "NTH-PERF01",
		MACAddress:  net.HardwareAddr{0x00, 0x00, 0xbc, 0x00, 0x00, 0x15},
		IPAddresses: []net.IP{net.ParseIP("10.71.240.15")},
		IPerf3:      iperf3,
	}
}

// iperf3Message is a length-prefixed JSON message of the control channel.
func iperf3Message(t *testing.T, value any) []byte {
	t.Helper()
	encoded, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}

	return append(binary.BigEndian.AppendUint32(nil, safeconv.Uint32(len(encoded))), encoded...)
}

// iperf3Wire drives one simulated iperf3 server with segments from a client
// that is not a simulated device, on a tagged VLAN, through a router.
type iperf3Wire struct {
	t       *testing.T
	stack   *Stack
	handler *TCPHandler
	device  *config.Device
}

func newIPerf3Wire(t *testing.T, device *config.Device) *iperf3Wire {
	t.Helper()
	stack := newTestStackInternal(t)
	stack.healthCheckHandler = NewHealthCheckHandler(stack)

	return &iperf3Wire{t: t, stack: stack, handler: NewTCPHandler(stack), device: device}
}

// send hands the handler one segment from client port from and returns every
// segment it queued in answer.
func (w *iperf3Wire) send(from layers.TCPPort, segment *layers.TCP) []*layers.TCP {
	w.t.Helper()
	segment.SrcPort = from
	pkt, ip := buildServiceSegment(w.t, w.device, TCPPortIPerf3, segment)
	pkt.VLAN = iperf3TestVLAN
	w.handler.routeToHandler(pkt, ip, segment, []*config.Device{w.device})

	var replies []*layers.TCP
	for {
		select {
		case reply := <-w.stack.sendQueue:
			decoded := gopacket.NewPacket(reply.Buffer, layers.LayerTypeEthernet, gopacket.Default)
			eth, _ := decoded.Layer(layers.LayerTypeEthernet).(*layers.Ethernet)
			tcpOut, _ := decoded.Layer(layers.LayerTypeTCP).(*layers.TCP)
			if eth == nil || tcpOut == nil {
				w.t.Fatalf("reply is not TCP over Ethernet: %v", decoded)
			}
			// The client is not a simulated device: the reply goes back to the
			// MAC the request came from, the router's, on the request's VLAN.
			if !bytes.Equal(eth.DstMAC, net.HardwareAddr{0xaa, 0, 0, 0, 0, 1}) || reply.VLAN != iperf3TestVLAN {
				w.t.Errorf("reply to %s on VLAN %d, want aa:00:00:00:00:01 on VLAN %d",
					eth.DstMAC, reply.VLAN, iperf3TestVLAN)
			}
			replies = append(replies, tcpOut)
		default:
			return replies
		}
	}
}

// iperf3Want is one segment the server must answer with.
type iperf3Want struct {
	to       layers.TCPPort
	flags    string // S, F, R and P for SYN, FIN, RST and PSH; every reply is an ACK
	seq, ack uint32
	payload  []byte // nil: no payload
	results  bool   // payload is a length-prefixed results message then DISPLAY_RESULTS
}

func tcpFlags(segment *layers.TCP) string {
	var flags strings.Builder
	for _, flag := range []struct {
		set  bool
		name string
	}{{segment.SYN, "S"}, {segment.FIN, "F"}, {segment.RST, "R"}, {segment.PSH, "P"}} {
		if flag.set {
			flags.WriteString(flag.name)
		}
	}

	return flags.String()
}

// TestIPerf3Conversation plays a complete forward TCP test as the iperf3 3.21
// client sends it, step by step, as captured against a real iperf3 server.
func TestIPerf3Conversation(t *testing.T) {
	wire := newIPerf3Wire(t, iperf3TestServer(&config.IPerf3Config{Enabled: true, Port: TCPPortIPerf3}))
	params := iperf3Message(t, map[string]any{
		"tcp": true, "omit": 0, "time": 10, "num": 0, "blockcount": 0, "parallel": 1,
		"len": 131072, "pacing_timer": 1000, "client_version": "3.21",
	})
	clientResults := iperf3Message(t, map[string]any{
		"cpu_util_total": 2.9, "cpu_util_user": 0.1, "cpu_util_system": 2.8, "sender_has_retransmits": 1,
		"congestion_used": "cubic",
		"streams": []map[string]any{{
			"id": 1, "bytes": 1100, "retransmits": 0, "jitter": 0, "errors": 0, "omitted_errors": 0,
			"packets": 0, "omitted_packets": 0, "start_time": 0, "end_time": 10,
		}},
	})
	bulk := bytes.Repeat([]byte{0x5a}, 1000)
	paramsEnd := 138 + safeconv.Uint32(len(params))
	resultsEnd := paramsEnd + 1 + safeconv.Uint32(len(clientResults))
	var serverResultsLen uint32

	steps := []struct {
		name    string
		from    layers.TCPPort
		segment *layers.TCP
		want    []iperf3Want
	}{
		{
			name: "control SYN", from: iperf3TestControlPort, segment: &layers.TCP{SYN: true, Seq: 100},
			want: []iperf3Want{{to: iperf3TestControlPort, flags: "S", seq: 1000, ack: 101}},
		},
		{
			name: "handshake ACK", from: iperf3TestControlPort, segment: &layers.TCP{ACK: true, Seq: 101, Ack: 1001},
		},
		{
			name: "cookie asks for parameters", from: iperf3TestControlPort,
			segment: &layers.TCP{ACK: true, PSH: true, Seq: 101, Ack: 1001, Payload: iperf3TestCookie()},
			want: []iperf3Want{{
				to: iperf3TestControlPort, flags: "P", seq: 1001, ack: 138, payload: []byte{iperf3ParamExchange},
			}},
		},
		{
			name: "parameters ask for streams", from: iperf3TestControlPort,
			segment: &layers.TCP{ACK: true, PSH: true, Seq: 138, Ack: 1002, Payload: params},
			want: []iperf3Want{{
				to: iperf3TestControlPort, flags: "P", seq: 1002, ack: paramsEnd, payload: []byte{iperf3CreateStreams},
			}},
		},
		{
			name: "data SYN", from: iperf3TestDataPort, segment: &layers.TCP{SYN: true, Seq: 500},
			want: []iperf3Want{{to: iperf3TestDataPort, flags: "S", seq: 1000, ack: 501}},
		},
		{
			name: "data cookie starts the test on the control connection", from: iperf3TestDataPort,
			segment: &layers.TCP{ACK: true, PSH: true, Seq: 501, Ack: 1001, Payload: iperf3TestCookie()},
			want: []iperf3Want{
				{to: iperf3TestDataPort, seq: 1001, ack: 538},
				{
					to: iperf3TestControlPort, flags: "P", seq: 1003, ack: paramsEnd,
					payload: []byte{iperf3TestStart, iperf3TestRunning},
				},
			},
		},
		{
			name: "bulk data is acknowledged and not answered", from: iperf3TestDataPort,
			segment: &layers.TCP{ACK: true, Seq: 538, Ack: 1001, Payload: bulk},
			want:    []iperf3Want{{to: iperf3TestDataPort, seq: 1001, ack: 1538}},
		},
		{
			name: "data past a gap gets a duplicate ACK", from: iperf3TestDataPort,
			segment: &layers.TCP{ACK: true, Seq: 3000, Ack: 1001, Payload: bulk[:100]},
			want:    []iperf3Want{{to: iperf3TestDataPort, seq: 1001, ack: 1538}},
		},
		{
			name: "data fills the gap", from: iperf3TestDataPort,
			segment: &layers.TCP{ACK: true, Seq: 1538, Ack: 1001, Payload: bulk[:100]},
			want:    []iperf3Want{{to: iperf3TestDataPort, seq: 1001, ack: 1638}},
		},
		{
			name: "TEST_END asks for results", from: iperf3TestControlPort,
			segment: &layers.TCP{ACK: true, PSH: true, Seq: paramsEnd, Ack: 1005, Payload: []byte{iperf3TestEnd}},
			want: []iperf3Want{{
				to: iperf3TestControlPort, flags: "P", seq: 1005, ack: paramsEnd + 1,
				payload: []byte{iperf3ExchangeResults},
			}},
		},
		{
			name: "client results are answered with the server's", from: iperf3TestControlPort,
			segment: &layers.TCP{ACK: true, PSH: true, Seq: paramsEnd + 1, Ack: 1006, Payload: clientResults},
			want:    []iperf3Want{{to: iperf3TestControlPort, flags: "P", seq: 1006, ack: resultsEnd, results: true}},
		},
	}
	for _, step := range steps {
		replies := wire.send(step.from, step.segment)
		checkIPerf3Replies(t, step.name, replies, step.want)
		if len(step.want) > 0 && step.want[len(step.want)-1].results {
			serverResultsLen = safeconv.Uint32(len(replies[len(replies)-1].Payload))
		}
	}

	done := wire.send(iperf3TestControlPort, &layers.TCP{
		ACK: true, PSH: true, FIN: true, Seq: resultsEnd, Ack: 1006 + serverResultsLen,
		Payload: []byte{iperf3Done},
	})
	if len(done) != 1 || tcpFlags(done[0]) != "F" || done[0].Ack != resultsEnd+2 {
		t.Fatalf("IPERF_DONE with FIN answered with %+v, want one FIN acknowledging %d", done, resultsEnd+2)
	}
	if conns, tests := len(wire.handler.iperf3.conns), len(wire.handler.iperf3.tests); conns != 0 || tests != 0 {
		t.Errorf("after IPERF_DONE %d connections and %d tests remain, want none", conns, tests)
	}
	// The test is over: what the client still sends on its data stream is reset.
	if reset := wire.send(
		iperf3TestDataPort,
		&layers.TCP{ACK: true, FIN: true, Seq: 1638, Ack: 1001},
	); len(reset) != 1 ||
		!reset[0].RST {
		t.Errorf("data FIN after the test answered with %+v, want RST", reset)
	}
}

// checkIPerf3Replies checks the segments the server answered one step with.
// A results reply carries the 1,100 bytes the stream received after its
// cookie, under the client's stream id 1.
func checkIPerf3Replies(t *testing.T, step string, replies []*layers.TCP, wants []iperf3Want) {
	t.Helper()
	if len(replies) != len(wants) {
		t.Fatalf("%s: %d replies %+v, want %d", step, len(replies), replies, len(wants))
	}
	for i, want := range wants {
		reply := replies[i]
		if reply.DstPort != want.to || tcpFlags(reply) != want.flags || !reply.ACK ||
			reply.Seq != want.seq || reply.Ack != want.ack {
			t.Fatalf("%s: reply %d to %d flags %q seq %d ack %d, want to %d flags %q seq %d ack %d",
				step, i, reply.DstPort, tcpFlags(reply), reply.Seq, reply.Ack,
				want.to, want.flags, want.seq, want.ack)
		}
		if want.results {
			checkIPerf3Results(t, reply.Payload, 1, 1100)
		} else if !bytes.Equal(reply.Payload, want.payload) {
			t.Fatalf("%s: reply %d payload % x, want % x", step, i, reply.Payload, want.payload)
		}
	}
}

// checkIPerf3Results checks a length-prefixed results message followed by
// DISPLAY_RESULTS against the fields the iperf3 client requires of it.
func checkIPerf3Results(t *testing.T, payload []byte, id int, received uint64) {
	t.Helper()
	if len(payload) < 5 || payload[len(payload)-1] != iperf3DisplayResults {
		t.Fatalf("results % x do not end with DISPLAY_RESULTS", payload)
	}
	length := binary.BigEndian.Uint32(payload)
	if int(length) != len(payload)-5 {
		t.Fatalf("results length prefix %d, want %d", length, len(payload)-5)
	}
	var results struct {
		CPUTotal             *float64 `json:"cpu_util_total"`
		CPUUser              *float64 `json:"cpu_util_user"`
		CPUSystem            *float64 `json:"cpu_util_system"`
		SenderHasRetransmits *int     `json:"sender_has_retransmits"`
		Streams              []struct {
			ID          *int     `json:"id"`
			Bytes       *uint64  `json:"bytes"`
			Retransmits *int     `json:"retransmits"`
			Jitter      *float64 `json:"jitter"`
			Errors      *int     `json:"errors"`
			Packets     *int     `json:"packets"`
		} `json:"streams"`
	}
	if err := json.Unmarshal(payload[4:len(payload)-1], &results); err != nil {
		t.Fatalf("results are not JSON: %v", err)
	}
	if results.CPUTotal == nil || results.CPUUser == nil || results.CPUSystem == nil ||
		results.SenderHasRetransmits == nil || len(results.Streams) != 1 {
		t.Fatalf("results %s miss a field the client requires", payload[4:len(payload)-1])
	}
	stream := results.Streams[0]
	if stream.ID == nil || stream.Bytes == nil || stream.Retransmits == nil || stream.Jitter == nil ||
		stream.Errors == nil || stream.Packets == nil {
		t.Fatalf("results stream %s misses a field the client requires", payload[4:len(payload)-1])
	}
	if *stream.ID != id || *stream.Bytes != received {
		t.Errorf("results stream id %d bytes %d, want id %d bytes %d", *stream.ID, *stream.Bytes, id, received)
	}
}

// TestIPerf3RefusesWhatItCannotServe covers the requests the server turns
// down: a device without iperf3 keeps the closed-port RST, and a test the
// server would have to send data for ends with SERVER_ERROR rather than a
// client that waits for streams that never come.
func TestIPerf3RefusesWhatItCannotServe(t *testing.T) {
	unimplemented := []byte{
		iperf3ServerError, 0, 0, 0, iperf3ErrUnimplemented, 0, 0, 0, 0,
	}
	tests := []struct {
		name   string
		iperf3 *config.IPerf3Config
		params map[string]any
		want   []byte // answer to the parameters; nil: SYN is reset
	}{
		{name: "no iperf3", iperf3: nil},
		{name: "iperf3 disabled", iperf3: &config.IPerf3Config{Port: TCPPortIPerf3}},
		{name: "reverse", params: map[string]any{"tcp": true, "parallel": 1, "reverse": true}, want: unimplemented},
		{
			name:   "bidirectional",
			params: map[string]any{"tcp": true, "parallel": 1, "bidirectional": true}, want: unimplemented,
		},
		{name: "UDP", params: map[string]any{"udp": true, "parallel": 1}, want: unimplemented},
		{name: "forward", params: map[string]any{"tcp": true, "parallel": 2}, want: []byte{iperf3CreateStreams}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			iperf3 := test.iperf3
			if test.want != nil {
				iperf3 = &config.IPerf3Config{Enabled: true, Port: TCPPortIPerf3}
			}
			wire := newIPerf3Wire(t, iperf3TestServer(iperf3))
			syn := wire.send(iperf3TestControlPort, &layers.TCP{SYN: true, Seq: 100})
			if test.want == nil {
				if len(syn) != 1 || !syn[0].RST {
					t.Fatalf("SYN answered with %+v, want RST", syn)
				}
				return
			}
			wire.send(iperf3TestControlPort, &layers.TCP{ACK: true, Seq: 101, Ack: 1001, Payload: iperf3TestCookie()})
			reply := wire.send(iperf3TestControlPort, &layers.TCP{
				ACK: true, Seq: 138, Ack: 1002, Payload: iperf3Message(t, test.params),
			})
			if len(reply) != 1 || !bytes.Equal(reply[0].Payload, test.want) {
				t.Fatalf("parameters answered with %+v, want % x", reply, test.want)
			}
		})
	}
}

// TestIPerf3ControlLossEndsTheTest covers a control connection that goes
// before its streams arrive: the test ends with it, and a stream that still
// sends its cookie is reset rather than started.
func TestIPerf3ControlLossEndsTheTest(t *testing.T) {
	tests := []struct {
		name string
		lose *layers.TCP
	}{
		{name: "reset", lose: &layers.TCP{RST: true, ACK: true, Seq: 0, Ack: 1003}},
		{name: "new SYN on the same ports", lose: &layers.TCP{SYN: true, Seq: 9000}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			wire := newIPerf3Wire(t, iperf3TestServer(&config.IPerf3Config{Enabled: true, Port: TCPPortIPerf3}))
			params := iperf3Message(t, map[string]any{"tcp": true, "parallel": 1})
			wire.send(iperf3TestControlPort, &layers.TCP{SYN: true, Seq: 100})
			wire.send(iperf3TestControlPort, &layers.TCP{
				ACK: true, Seq: 101, Ack: 1001, Payload: iperf3TestCookie(),
			})
			wire.send(iperf3TestControlPort, &layers.TCP{ACK: true, Seq: 138, Ack: 1002, Payload: params})
			test.lose.Seq = max(test.lose.Seq, 138+safeconv.Uint32(len(params)))
			wire.send(iperf3TestControlPort, test.lose)
			if tests := len(wire.handler.iperf3.tests); tests != 0 {
				t.Fatalf("%d tests outlive their control connection", tests)
			}
			wire.send(iperf3TestDataPort, &layers.TCP{SYN: true, Seq: 500})
			reply := wire.send(iperf3TestDataPort, &layers.TCP{
				ACK: true, Seq: 501, Ack: 1001, Payload: iperf3TestCookie(),
			})
			if len(reply) != 1 || reply[0].RST || len(reply[0].Payload) != 1 ||
				reply[0].Payload[0] != iperf3ParamExchange {
				t.Errorf("cookie of a finished test answered with %+v, want a new test's PARAM_EXCHANGE", reply)
			}
		})
	}
}
