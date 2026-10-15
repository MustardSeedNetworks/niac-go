package protocols

import (
	"net/netip"
	"sync"
	"time"

	"github.com/gopacket/gopacket/layers"

	"github.com/MustardSeedNetworks/niac-go/internal/safeconv"
)

// Bounds on the DICOM reassembly state. A verification association's PDUs
// are a few hundred bytes; an SCU proposing every storage class sends an
// A-ASSOCIATE-RQ of a few kilobytes. A longer PDU is aborted from its header,
// and past dicomMaxStreams connections the oldest is forgotten.
const (
	dicomMaxStreams = 256
	dicomMaxPDU     = 64 << 10
)

// dicomStreams holds what has arrived of each DICOM connection's next PDU.
// SCUs write a PDU in pieces (dcmtk sends the PDU and PDV headers of a
// C-ECHO-RQ in one segment and the command in the next), so the service reads
// a byte stream rather than segments. Only the inbound direction is tracked:
// every answer is sent once, acknowledged by the client's next segment.
type dicomStreams struct {
	mu      sync.Mutex
	streams map[dicomStreamKey]*dicomStream
}

type dicomStreamKey struct {
	client, server netip.AddrPort
}

type dicomStream struct {
	next     uint32 // sequence number of the next in-order byte
	pending  []byte
	lastSeen time.Time
}

func newDICOMStreams() *dicomStreams {
	return &dicomStreams{streams: make(map[dicomStreamKey]*dicomStream)}
}

func newDICOMStreamKey(ipLayer *layers.IPv4, tcp *layers.TCP) dicomStreamKey {
	client, _ := netip.AddrFromSlice(ipLayer.SrcIP.To4())
	server, _ := netip.AddrFromSlice(ipLayer.DstIP.To4())

	return dicomStreamKey{
		client: netip.AddrPortFrom(client, uint16(tcp.SrcPort)),
		server: netip.AddrPortFrom(server, uint16(tcp.DstPort)),
	}
}

// open starts a connection whose first data byte is next, replacing any
// earlier connection on the same ports.
func (d *dicomStreams) open(key dicomStreamKey, next uint32) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if _, known := d.streams[key]; !known && len(d.streams) >= dicomMaxStreams {
		d.evictOldestLocked()
	}
	d.streams[key] = &dicomStream{next: next, lastSeen: time.Now()}
}

func (d *dicomStreams) close(key dicomStreamKey) {
	d.mu.Lock()
	defer d.mu.Unlock()
	delete(d.streams, key)
}

// receive takes a segment's payload starting at seq and returns the answer to
// every PDU it completes. inOrder is false for a segment that is not the next
// one, which goes unacknowledged so the client sends the gap again; a repeat
// of the last segment is acknowledged without being answered twice. A
// connection the service never saw open (the simulation restarted) starts at
// the segment.
func (d *dicomStreams) receive(key dicomStreamKey, seq uint32, payload []byte) ([]byte, bool) {
	d.mu.Lock()
	defer d.mu.Unlock()
	stream := d.streams[key]
	if stream == nil {
		if len(d.streams) >= dicomMaxStreams {
			d.evictOldestLocked()
		}
		stream = &dicomStream{next: seq}
		d.streams[key] = stream
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
	answer, consumed, ended := dicomAnswer(stream.pending)
	if ended {
		delete(d.streams, key)
		return answer, true
	}
	stream.pending = append(stream.pending[:0], stream.pending[consumed:]...)

	return answer, true
}

func (d *dicomStreams) evictOldestLocked() {
	var (
		oldest     dicomStreamKey
		oldestSeen time.Time
	)
	for key, stream := range d.streams {
		if oldestSeen.IsZero() || stream.lastSeen.Before(oldestSeen) {
			oldest, oldestSeen = key, stream.lastSeen
		}
	}
	delete(d.streams, oldest)
}
