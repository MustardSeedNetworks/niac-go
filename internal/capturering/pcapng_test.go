package capturering_test

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/gopacket/gopacket/pcapgo"

	"github.com/MustardSeedNetworks/niac-go/internal/capturering"
	"github.com/MustardSeedNetworks/niac-go/internal/protocols"
)

func readBack(t *testing.T, raw []byte) (*pcapgo.NgReader, func() ([]byte, pcapgo.NgPacketOptions, time.Time, bool)) {
	t.Helper()
	reader, err := pcapgo.NewNgReader(bytes.NewReader(raw), pcapgo.DefaultNgReaderOptions)
	if err != nil {
		t.Fatalf("NewNgReader: %v", err)
	}

	return reader, func() ([]byte, pcapgo.NgPacketOptions, time.Time, bool) {
		data, ci, opts, readErr := reader.ReadPacketDataWithOptions()
		if errors.Is(readErr, io.EOF) {
			return nil, opts, time.Time{}, false
		}
		if readErr != nil {
			t.Fatalf("ReadPacketDataWithOptions: %v", readErr)
		}

		return data, opts, ci.Timestamp, true
	}
}

func TestWritePcapngRoundTripsFramesTimestampsAndComments(t *testing.T) {
	stamp := time.Unix(1_756_000_000, 0).UTC()
	frames := []capturering.Frame{
		{
			Timestamp: stamp,
			Direction: "rx",
			Serial:    12,
			VLAN:      101,
			Trace: protocols.FabricTrace{
				IngressNetwork: "clinical",
				PhysicalVLAN:   101,
				RouteDecision:  "forwarded",
				Hop:            "core-1",
				EgressNetwork:  "imaging",
			},
			Data: []byte{0xde, 0xad, 0xbe, 0xef},
		},
		{Timestamp: stamp.Add(time.Second), Direction: "tx", Serial: 13, VLAN: -1, Data: []byte{0x01, 0x02}},
	}

	var out bytes.Buffer
	if err := capturering.WritePcapng(&out, "eth0", frames); err != nil {
		t.Fatalf("WritePcapng: %v", err)
	}

	reader, next := readBack(t, out.Bytes())
	if reader.NInterfaces() != 1 {
		t.Fatalf("interface blocks = %d, want 1", reader.NInterfaces())
	}
	intf, err := reader.Interface(0)
	if err != nil {
		t.Fatalf("Interface(0): %v", err)
	}
	if intf.Name != "eth0" {
		t.Errorf("interface name = %q, want %q", intf.Name, "eth0")
	}

	data, opts, ts, ok := next()
	if !ok {
		t.Fatal("no first packet")
	}
	if !bytes.Equal(data, frames[0].Data) {
		t.Errorf("frame 0 bytes = % x, want % x", data, frames[0].Data)
	}
	if !ts.Equal(stamp) {
		t.Errorf("frame 0 timestamp = %v, want %v", ts, stamp)
	}
	comment := strings.Join(opts.Comments, "\n")
	for _, want := range []string{"direction=rx", "vlan=101", "ingress=clinical", "egress=imaging", "route=forwarded", "hop=core-1"} {
		if !strings.Contains(comment, want) {
			t.Errorf("comment %q is missing %q", comment, want)
		}
	}

	data, opts, _, ok = next()
	if !ok {
		t.Fatal("no second packet")
	}
	if !bytes.Equal(data, frames[1].Data) {
		t.Errorf("frame 1 bytes = % x, want % x", data, frames[1].Data)
	}
	if strings.Contains(strings.Join(opts.Comments, "\n"), "vlan=") {
		t.Errorf("an untagged frame carried a vlan comment: %q", opts.Comments)
	}

	if _, _, _, ok = next(); ok {
		t.Error("more packets than were written")
	}
}

func TestWritePcapngWritesAValidEmptyCapture(t *testing.T) {
	var out bytes.Buffer
	if err := capturering.WritePcapng(&out, "lo", nil); err != nil {
		t.Fatalf("WritePcapng: %v", err)
	}
	reader, next := readBack(t, out.Bytes())
	if reader.NInterfaces() != 1 {
		t.Fatalf("interface blocks = %d, want 1", reader.NInterfaces())
	}
	if _, _, _, ok := next(); ok {
		t.Error("empty capture yielded a packet")
	}
}
