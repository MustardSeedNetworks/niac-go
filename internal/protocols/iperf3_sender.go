package protocols

import (
	"net"
	"time"

	"github.com/gopacket/gopacket/layers"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
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
