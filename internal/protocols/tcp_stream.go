package protocols

import (
	"net/netip"
	"sync"
	"time"

	"github.com/gopacket/gopacket/layers"

	"github.com/MustardSeedNetworks/niac-go/internal/safeconv"
)

// streamMaxConnections bounds the connections a service reassembles at once;
// past it the oldest is forgotten.
const streamMaxConnections = 256

// streamAnswer reads the complete requests at the front of a connection's
// byte stream and returns the answer to them, in order, and how many bytes
// they took. ended reports that the conversation is over and nothing after it
// is answered. Each service bounds what it leaves unconsumed.
type streamAnswer func(stream []byte) (answer []byte, consumed int, ended bool)

// serviceStreams holds what has arrived of each connection's next request for
// a service that answers a byte stream rather than segments: clients write a
// request in pieces (dcmtk sends the PDU and PDV headers of a C-ECHO-RQ in
// one segment and the command in the next). Only the inbound direction is
// tracked: every answer is sent once, acknowledged by the client's next
// segment.
type serviceStreams struct {
	mu      sync.Mutex
	streams map[streamKey]*serviceStream
}

type streamKey struct {
	client, server netip.AddrPort
}

type serviceStream struct {
	next     uint32 // sequence number of the next in-order byte
	pending  []byte
	lastSeen time.Time
}

func newServiceStreams() *serviceStreams {
	return &serviceStreams{streams: make(map[streamKey]*serviceStream)}
}

func newStreamKey(ipLayer *layers.IPv4, tcp *layers.TCP) streamKey {
	client, _ := netip.AddrFromSlice(ipLayer.SrcIP.To4())
	server, _ := netip.AddrFromSlice(ipLayer.DstIP.To4())

	return streamKey{
		client: netip.AddrPortFrom(client, uint16(tcp.SrcPort)),
		server: netip.AddrPortFrom(server, uint16(tcp.DstPort)),
	}
}

// open starts a connection whose first data byte is next, replacing any
// earlier connection on the same ports.
func (s *serviceStreams) open(key streamKey, next uint32) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, known := s.streams[key]; !known && len(s.streams) >= streamMaxConnections {
		s.evictOldestLocked()
	}
	s.streams[key] = &serviceStream{next: next, lastSeen: time.Now()}
}

func (s *serviceStreams) close(key streamKey) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.streams, key)
}

// receive takes a segment's payload starting at seq and returns answer's
// reply to every request it completes. inOrder is false for a segment that
// is not the next one, which goes unacknowledged so the client sends the gap
// again; a repeat of the last segment is acknowledged without being answered
// twice. A connection the service never saw open (the simulation restarted)
// starts at the segment.
func (s *serviceStreams) receive(key streamKey, seq uint32, payload []byte, answer streamAnswer) ([]byte, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	stream := s.streams[key]
	if stream == nil {
		if len(s.streams) >= streamMaxConnections {
			s.evictOldestLocked()
		}
		stream = &serviceStream{next: seq}
		s.streams[key] = stream
	}
	switch end := seq + safeconv.Uint32(len(payload)); {
	case seq == stream.next:
	case end == stream.next:
		return nil, true
	default:
		return nil, false
	}
	stream.lastSeen = time.Now()
	stream.next += safeconv.Uint32(len(payload))
	stream.pending = append(stream.pending, payload...)
	reply, consumed, ended := answer(stream.pending)
	if ended {
		delete(s.streams, key)
		return reply, true
	}
	stream.pending = append(stream.pending[:0], stream.pending[consumed:]...)

	return reply, true
}

func (s *serviceStreams) evictOldestLocked() {
	var (
		oldest     streamKey
		oldestSeen time.Time
	)
	for key, stream := range s.streams {
		if oldestSeen.IsZero() || stream.lastSeen.Before(oldestSeen) {
			oldest, oldestSeen = key, stream.lastSeen
		}
	}
	delete(s.streams, oldest)
}
