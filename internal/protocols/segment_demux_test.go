package protocols

import (
	"net"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/logging"
)

func segDevice(name, mac, ip string) config.Device {
	return config.Device{
		Name:        name,
		MACAddress:  mustMAC(mac),
		IPAddresses: []net.IP{net.ParseIP(ip)},
	}
}

func mustMAC(s string) net.HardwareAddr {
	m, err := net.ParseMAC(s)
	if err != nil {
		panic(err)
	}

	return m
}

// TestSegmentDemuxIsolatesReusedIP: two demos on different tags reuse the same
// IP; devicesFor routes each tag to its own device — the core of "run N demos
// at once".
func TestSegmentDemuxIsolatesReusedIP(t *testing.T) {
	cfg := &config.Config{
		Segments: []config.Segment{
			{Tag: 200, Devices: []config.Device{segDevice("demo1-host", "02:00:00:00:02:00", "10.0.0.1")}},
			{Tag: 300, Devices: []config.Device{segDevice("demo2-host", "02:00:00:00:03:00", "10.0.0.1")}},
		},
	}

	stack := NewStack(nil, cfg, logging.NewDebugConfig(0))
	ip := net.ParseIP("10.0.0.1")

	d200 := stack.devicesFor(200).GetByIP(ip)
	d300 := stack.devicesFor(300).GetByIP(ip)

	if len(d200) != 1 || len(d300) != 1 {
		t.Fatalf("each segment should resolve 10.0.0.1 to one device; got %d (200) and %d (300)", len(d200), len(d300))
	}

	if d200[0].Name != "demo1-host" {
		t.Errorf("VLAN 200 resolved 10.0.0.1 to %q, want demo1-host", d200[0].Name)
	}

	if d300[0].Name != "demo2-host" {
		t.Errorf("VLAN 300 resolved 10.0.0.1 to %q, want demo2-host", d300[0].Name)
	}

	// An unclaimed tag has no devices (decodePacket drops such frames).
	if stack.devicesFor(999) != nil {
		t.Errorf("unclaimed VLAN 999 should have no device table")
	}
}

func TestSegmentDemuxFailsClosedForDuplicateTags(t *testing.T) {
	cfg := &config.Config{
		Segments: []config.Segment{
			{Tag: 200, Devices: []config.Device{segDevice("first", "02:00:00:00:02:00", "10.0.0.1")}},
			{Tag: 200, Devices: []config.Device{segDevice("second", "02:00:00:00:03:00", "10.0.0.2")}},
		},
	}

	stack := NewStack(nil, cfg, logging.NewDebugConfig(0))

	if table := stack.devicesFor(200); table != nil {
		t.Fatalf("devicesFor(200) = %#v, want no table for duplicate tag", table)
	}
}

// TestFlatModeUnchanged: a config with no segments stays in flat mode —
// devicesFor returns the single global table for every VLAN.
func TestFlatModeUnchanged(t *testing.T) {
	cfg := &config.Config{
		Devices: []config.Device{segDevice("flat-host", "02:00:00:00:0f:00", "10.0.0.1")},
	}

	stack := NewStack(nil, cfg, logging.NewDebugConfig(0))

	if stack.segmentTables != nil {
		t.Fatal("a config without segments must not enter segment mode")
	}

	// Every VLAN resolves to the same global table.
	if stack.devicesFor(200) != stack.devices || stack.devicesFor(0) != stack.devices {
		t.Error("flat mode: devicesFor must return the global device table for any VLAN")
	}

	if got := stack.devicesFor(200).GetByIP(net.ParseIP("10.0.0.1")); len(got) != 1 {
		t.Errorf("flat mode: 10.0.0.1 should resolve to the one device, got %d", len(got))
	}
}
