package capture

import (
	"os"
	"slices"
	"sync"
	"testing"

	"github.com/gopacket/gopacket/pcapgo"
)

// recordingSender is the test double for PacketSender, the playback egress
// boundary. It keeps a copy of every frame handed to SendPacket so tests can
// assert on the bytes that actually reached the wire rather than on counters
// alone, and it can be told to fail from a given send onward to exercise the
// error path.
//
// Playback runs its send loop on its own goroutine, so every field is guarded:
// the -race job drives these tests too.
type recordingSender struct {
	mu     sync.Mutex
	frames [][]byte

	// failFrom is the 1-based send ordinal from which SendPacket returns
	// failErr. Zero means never fail.
	failFrom int
	failErr  error
}

// SendPacket records a copy of the frame. The caller owns pkt only for the
// duration of the call — playback streams from a reused reader buffer — so the
// copy is what makes later assertions meaningful.
func (s *recordingSender) SendPacket(pkt []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	ordinal := len(s.frames) + 1
	if s.failFrom > 0 && ordinal >= s.failFrom {
		return s.failErr
	}

	s.frames = append(s.frames, slices.Clone(pkt))

	return nil
}

// Frames returns a snapshot of the frames sent so far.
func (s *recordingSender) Frames() [][]byte {
	s.mu.Lock()
	defer s.mu.Unlock()

	return slices.Clone(s.frames)
}

// Count returns how many frames have been sent.
func (s *recordingSender) Count() int {
	s.mu.Lock()
	defer s.mu.Unlock()

	return len(s.frames)
}

// assertFramesMatchPCAP checks the recorded frames are byte-for-byte the
// packets createTestPCAP wrote, in order. This is the assertion the old
// loopback tests could not make: it fails if playback sends the right *number*
// of the wrong frames, reorders them, or truncates one.
func assertFramesMatchPCAP(t *testing.T, sender *recordingSender, pcapFile string) {
	t.Helper()

	want := readPCAPFrames(t, pcapFile)
	got := sender.Frames()

	if len(got) != len(want) {
		t.Fatalf("sent %d frames, want %d", len(got), len(want))
	}

	for i := range want {
		if !slices.Equal(got[i], want[i]) {
			t.Fatalf("frame %d = %x, want %x", i, got[i], want[i])
		}
	}
}

// readPCAPFrames reads every packet out of a PCAP file, independently of the
// playback engine, so assertions compare playback's output against the file
// rather than against playback's own reader.
func readPCAPFrames(t *testing.T, pcapFile string) [][]byte {
	t.Helper()

	f, err := os.Open(pcapFile)
	if err != nil {
		t.Fatalf("open PCAP %s: %v", pcapFile, err)
	}
	defer func() { _ = f.Close() }()

	reader, err := pcapgo.NewReader(f)
	if err != nil {
		t.Fatalf("read PCAP header %s: %v", pcapFile, err)
	}

	var frames [][]byte

	for {
		data, _, readErr := reader.ReadPacketData()
		if readErr != nil {
			break
		}

		frames = append(frames, slices.Clone(data))
	}

	return frames
}
