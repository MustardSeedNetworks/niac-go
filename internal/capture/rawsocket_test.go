//go:build rawsocket

// Package capture's raw-socket boundary. Everything else in this package is
// driven through the PacketSender and packetHandle seams and needs no
// privileges; this file is the part that cannot be: opening a live pcap handle,
// installing a real compiled BPF filter, and putting a frame on the wire.
//
// It is behind a build tag so the default `go test ./...` never picks it up,
// and it is run by one CI job that grants CAP_NET_RAW/CAP_NET_ADMIN explicitly.
// Nothing here skips: a missing capability, a missing loopback or a handle that
// will not open is a failure, because the whole point of the job is to prove
// those work. A suite that quietly no-ops is the defect it exists to catch.
package capture

import (
	"bytes"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/layers"
	"github.com/gopacket/gopacket/pcap"
)

// loopbackNames are the platform spellings of the loopback interface.
var loopbackNames = []string{"lo", "lo0", "Loopback"}

// requireLoopback returns the loopback interface name, failing when there is
// none. Unlike the old t.Skip, an absent loopback means the job's premise is
// broken and must be reported as such.
func requireLoopback(t *testing.T) string {
	t.Helper()

	for _, name := range loopbackNames {
		if InterfaceExists(name) {
			return name
		}
	}

	devices, err := pcap.FindAllDevs()
	if err != nil {
		t.Fatalf("no loopback interface among %v, and enumeration failed: %v", loopbackNames, err)
	}

	present := make([]string, len(devices))
	for i, device := range devices {
		present[i] = device.Name
	}

	t.Fatalf("no loopback interface among %v; interfaces present: %v", loopbackNames, present)

	return ""
}

// requireEngine opens a live capture engine on the loopback interface. A
// failure here is almost always a missing CAP_NET_RAW, so the message says so:
// the job is meaningless if the engine cannot open, and skipping would hide it.
func requireEngine(t *testing.T, debugLevel int) *Engine {
	t.Helper()

	iface := requireLoopback(t)

	engine, err := New(iface, debugLevel)
	if err != nil {
		t.Fatalf("cannot open a live capture handle on %s: %v\n"+
			"this job requires CAP_NET_RAW and CAP_NET_ADMIN on the test binary; "+
			"see the capture-rawsocket job in .github/workflows/ci.yml", iface, err)
	}

	t.Cleanup(engine.Close)

	return engine
}

// TestPreflight_RawSocketCapabilitiesPresent runs first and states the job's
// premise outright, so a capability regression fails with one clear message
// instead of a scattering of confusing downstream errors.
func TestPreflight_RawSocketCapabilitiesPresent(t *testing.T) {
	if os.Getenv("NIAC_RAWSOCKET_TESTS") != "1" {
		t.Fatal("NIAC_RAWSOCKET_TESTS must be set to 1 for this suite; " +
			"it exists so an accidental `go test -tags rawsocket` without the " +
			"capability grant fails loudly rather than reporting a false pass")
	}

	engine := requireEngine(t, 0)

	if engine.handle == nil {
		t.Fatal("live engine has a nil handle")
	}
}

// TestEngine_New_LiveInterface covers what the seam cannot: New really opens
// the named interface and records it.
func TestEngine_New_LiveInterface(t *testing.T) {
	iface := requireLoopback(t)
	engine := requireEngine(t, 0)

	if engine.interfaceName != iface {
		t.Errorf("interfaceName = %q, want %q", engine.interfaceName, iface)
	}

	if lt := engine.handle.LinkType(); lt == 0 {
		t.Errorf("live handle reports link type %v, want a real one", lt)
	}
}

// TestEngine_New_InvalidInterface_Live proves the error path is a real libpcap
// failure rather than an early argument check.
func TestEngine_New_InvalidInterface_Live(t *testing.T) {
	if _, err := New("definitely-does-not-exist-interface-12345", 0); err == nil {
		t.Fatal("New on a non-existent interface returned nil, want an error")
	}
}

// TestEngine_Close_LiveHandleIsIdempotent closes a live handle twice. libpcap
// double-free is the failure this guards; the seam's fake cannot reproduce it.
func TestEngine_Close_LiveHandleIsIdempotent(t *testing.T) {
	engine, err := New(requireLoopback(t), 0)
	if err != nil {
		t.Fatalf("cannot open a live capture handle: %v", err)
	}

	engine.Close()
	engine.Close()
}

// TestEngine_SetFilter_LiveCompile installs filters libpcap actually compiles —
// the one thing the substitute handle cannot check, since it only records the
// expression string.
func TestEngine_SetFilter_LiveCompile(t *testing.T) {
	engine := requireEngine(t, 0)

	for _, expr := range []string{"ip", "tcp", "udp port 53"} {
		if err := engine.SetFilter(expr); err != nil {
			t.Fatalf("SetFilter(%q) on a live handle: %v", expr, err)
		}

		if got := engine.Filter(); got != expr {
			t.Errorf("Filter() = %q, want %q", got, expr)
		}
	}

	if err := engine.SetFilter("this is not a bpf filter !!!"); err == nil {
		t.Fatal("libpcap accepted a malformed BPF filter")
	}

	if got := engine.Filter(); got != "udp port 53" {
		t.Errorf("after a rejected filter, Filter() = %q, want the last good one", got)
	}
}

// TestEngine_Stats_LiveHandle reads counters from the kernel rather than from a
// substitute.
func TestEngine_Stats_LiveHandle(t *testing.T) {
	engine := requireEngine(t, 0)

	if _, err := engine.Stats(); err != nil {
		t.Fatalf("Stats() on a live handle: %v", err)
	}
}

// rawSocketEtherType is an unassigned experimental EtherType. Filtering the
// capture on it makes the round trip deterministic: the reader can only ever
// see this test's own frames, however noisy the loopback interface is.
const rawSocketEtherType = 0x88b5

// TestEngine_SendAndCapture_RoundTrip is the whole reason this job exists: a
// frame written through the live handle is captured back off the wire with its
// payload intact.
func TestEngine_SendAndCapture_RoundTrip(t *testing.T) {
	reader := requireEngine(t, 0)

	// Filter on this test's own EtherType so nothing else on the interface can
	// satisfy the assertion, and so the read loop cannot be fooled by traffic
	// that merely happens to arrive.
	if err := reader.SetFilter("ether proto 0x88b5"); err != nil {
		t.Fatalf("SetFilter on the reader: %v", err)
	}

	writer := requireEngine(t, 0)

	payload := []byte("niac-rawsocket-round-trip")
	frame := buildTaggedFrame(t, payload)

	captured := make(chan []byte, 1)
	readErr := make(chan error, 1)

	go func() {
		buf := make([]byte, snapshotLength)

		deadline := time.Now().Add(10 * time.Second)
		for time.Now().Before(deadline) {
			data, err := reader.ReadPacket(buf)
			if err != nil {
				readErr <- err

				return
			}
			// A nil frame with a nil error is the capture timeout, which is how
			// the handle stays responsive; keep reading until the deadline.
			if data == nil {
				continue
			}
			if bytes.Contains(data, payload) {
				captured <- append([]byte(nil), data...)

				return
			}
		}

		readErr <- errors.New("no frame carrying the payload arrived within the deadline")
	}()

	// Send repeatedly: the reader goroutine may not have entered its loop yet,
	// and a frame sent before it does is genuinely gone. Sending more of them
	// cannot make a broken send path look like a working one.
	sendTicker := time.NewTicker(50 * time.Millisecond)
	defer sendTicker.Stop()

	if err := writer.SendPacket(frame); err != nil {
		t.Fatalf("SendPacket on a live handle: %v", err)
	}

	for {
		select {
		case data := <-captured:
			assertTaggedFrame(t, data, payload)

			return
		case err := <-readErr:
			t.Fatalf("capture failed: %v", err)
		case <-sendTicker.C:
			if err := writer.SendPacket(frame); err != nil {
				t.Fatalf("SendPacket on a live handle: %v", err)
			}
		}
	}
}

// buildTaggedFrame serializes an Ethernet frame carrying payload under
// rawSocketEtherType.
func buildTaggedFrame(t *testing.T, payload []byte) []byte {
	t.Helper()

	eth := &layers.Ethernet{
		SrcMAC:       []byte{0x02, 0x00, 0x00, 0x00, 0x00, 0x01},
		DstMAC:       []byte{0x02, 0x00, 0x00, 0x00, 0x00, 0x02},
		EthernetType: layers.EthernetType(rawSocketEtherType),
	}

	buf := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{FixLengths: true}

	if err := gopacket.SerializeLayers(buf, opts, eth, gopacket.Payload(payload)); err != nil {
		t.Fatalf("serialize frame: %v", err)
	}

	return buf.Bytes()
}

// assertTaggedFrame checks the captured bytes decode to the frame that was
// sent, rather than merely containing the payload somewhere.
func assertTaggedFrame(t *testing.T, data, payload []byte) {
	t.Helper()

	packet := gopacket.NewPacket(data, layers.LayerTypeEthernet, gopacket.Default)

	eth, ok := packet.Layer(layers.LayerTypeEthernet).(*layers.Ethernet)
	if !ok {
		t.Fatalf("captured frame has no Ethernet layer: %v", packet)
	}

	if eth.EthernetType != layers.EthernetType(rawSocketEtherType) {
		t.Errorf("captured ethertype = %v, want %#x", eth.EthernetType, rawSocketEtherType)
	}

	if !bytes.HasPrefix(eth.Payload, payload) {
		t.Errorf("captured payload = %x, want it to start with %x", eth.Payload, payload)
	}
}
