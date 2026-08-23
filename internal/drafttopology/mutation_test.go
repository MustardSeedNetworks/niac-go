package drafttopology_test

import (
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/drafttopology"
)

func testConfig() *config.Config {
	return &config.Config{Devices: []config.Device{
		{
			Name:       "core-1",
			Type:       "switch",
			Interfaces: []config.Interface{{Name: "Ethernet1/1", Type: "ethernet"}},
		},
		{
			Name:       "dist-1",
			Type:       "switch",
			Interfaces: []config.Interface{{Name: "Ethernet1/49", Type: "ethernet"}},
		},
	}}
}

func TestApplyConnectUpdateDisconnect(t *testing.T) {
	cfg := testConfig()
	endpoints := drafttopology.LinkEndpoints{
		Source: drafttopology.Endpoint{Device: "core-1", Interface: "Ethernet1/1"},
		Target: drafttopology.Endpoint{Device: "dist-1", Interface: "Ethernet1/49"},
	}

	err := drafttopology.Apply(cfg, drafttopology.Mutation{
		Operation: drafttopology.Connect,
		Link: &drafttopology.LinkMutation{
			LinkEndpoints: endpoints,
			Properties:    drafttopology.LinkProperties{VLANs: []int{200, 210}, NativeVLAN: 200},
		},
	})
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	assertReciprocalLink(t, cfg, []int{200, 210}, 200)

	err = drafttopology.Apply(cfg, drafttopology.Mutation{
		Operation: drafttopology.UpdateLink,
		Link: &drafttopology.LinkMutation{
			LinkEndpoints: endpoints,
			Properties: drafttopology.LinkProperties{
				VLANs:      []int{220},
				NativeVLAN: 220,
				FDBOnly:    true,
			},
		},
	})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	assertReciprocalLink(t, cfg, []int{220}, 220)
	if !cfg.Devices[0].TrunkPorts[0].FDBOnly || !cfg.Devices[1].TrunkPorts[0].FDBOnly {
		t.Fatal("updated FDB-only flag was not mirrored")
	}

	err = drafttopology.Apply(cfg, drafttopology.Mutation{
		Operation: drafttopology.Disconnect,
		Link:      &drafttopology.LinkMutation{LinkEndpoints: endpoints},
	})
	if err != nil {
		t.Fatalf("disconnect: %v", err)
	}
	if len(cfg.Devices[0].TrunkPorts) != 0 || len(cfg.Devices[1].TrunkPorts) != 0 {
		t.Fatal("disconnect did not remove both link endpoints")
	}
}

func TestApplyRejectsOccupiedAndImpossiblePorts(t *testing.T) {
	cfg := testConfig()
	connect := drafttopology.Mutation{
		Operation: drafttopology.Connect,
		Link: &drafttopology.LinkMutation{
			Source:     drafttopology.Endpoint{Device: "core-1", Interface: "Ethernet1/1"},
			Target:     drafttopology.Endpoint{Device: "dist-1", Interface: "Ethernet1/49"},
			Properties: drafttopology.LinkProperties{VLANs: []int{200}, NativeVLAN: 200},
		},
	}
	if err := drafttopology.Apply(cfg, connect); err != nil {
		t.Fatalf("first connect: %v", err)
	}
	if err := drafttopology.Apply(cfg, connect); !drafttopology.IsConflict(err) {
		t.Fatalf("second connect error = %v, want conflict", err)
	}

	bad := testConfig()
	bad.Devices[1].Interfaces[0].Type = "loopback"
	if err := drafttopology.Apply(bad, connect); !drafttopology.IsInvalid(err) {
		t.Fatalf("loopback connect error = %v, want invalid", err)
	}
}

func TestApplyRejectsRemoteEndpointOccupiedByOneSidedLink(t *testing.T) {
	cfg := testConfig()
	cfg.Devices[0].TrunkPorts = []config.TrunkPort{{
		Interface: "Ethernet1/1", RemoteDevice: "dist-1", RemoteInterface: "Ethernet1/49",
	}}
	cfg.Devices = append(cfg.Devices, config.Device{
		Name: "access-1", Type: "switch",
		Interfaces: []config.Interface{{Name: "Ethernet1/1", Type: "ethernet"}},
	})
	err := drafttopology.Apply(cfg, drafttopology.Mutation{
		Operation: drafttopology.Connect,
		Link: &drafttopology.LinkMutation{
			Source: drafttopology.Endpoint{Device: "dist-1", Interface: "Ethernet1/49"},
			Target: drafttopology.Endpoint{Device: "access-1", Interface: "Ethernet1/1"},
		},
	})
	if !drafttopology.IsConflict(err) {
		t.Fatalf("connect error = %v, want occupied conflict", err)
	}
}

func TestApplyRejectsCrossSegmentLink(t *testing.T) {
	cfg := &config.Config{Segments: []config.Segment{
		{Tag: 200, Devices: []config.Device{{
			Name: "core-1", Type: "switch",
			Interfaces: []config.Interface{{Name: "Ethernet1/1", Type: "ethernet"}},
		}}},
		{Tag: 300, Devices: []config.Device{{
			Name: "dist-1", Type: "switch",
			Interfaces: []config.Interface{{Name: "Ethernet1/49", Type: "ethernet"}},
		}}},
	}}
	err := drafttopology.Apply(cfg, drafttopology.Mutation{
		Operation: drafttopology.Connect,
		Link: &drafttopology.LinkMutation{
			Source: drafttopology.Endpoint{Device: "core-1", Interface: "Ethernet1/1"},
			Target: drafttopology.Endpoint{Device: "dist-1", Interface: "Ethernet1/49"},
		},
	})
	if !drafttopology.IsInvalid(err) {
		t.Fatalf("connect error = %v, want invalid", err)
	}
}

func TestApplyRejectsAsymmetricLinkUpdate(t *testing.T) {
	cfg := testConfig()
	cfg.Devices[0].TrunkPorts = []config.TrunkPort{{
		Interface: "Ethernet1/1", RemoteDevice: "dist-1", RemoteInterface: "Ethernet1/49",
		VLANs: []int{200},
	}}
	cfg.Devices[1].TrunkPorts = []config.TrunkPort{{
		Interface: "Ethernet1/49", RemoteDevice: "core-1", RemoteInterface: "Ethernet1/1",
		VLANs: []int{200, 210},
	}}
	err := drafttopology.Apply(cfg, drafttopology.Mutation{
		Operation: drafttopology.UpdateLink,
		Link: &drafttopology.LinkMutation{
			Source:     drafttopology.Endpoint{Device: "core-1", Interface: "Ethernet1/1"},
			Target:     drafttopology.Endpoint{Device: "dist-1", Interface: "Ethernet1/49"},
			Properties: drafttopology.LinkProperties{VLANs: []int{200}},
		},
	})
	if !drafttopology.IsConflict(err) {
		t.Fatalf("update error = %v, want conflict", err)
	}
}

func TestValidateSourceRejectsConfigBackedSegments(t *testing.T) {
	if err := drafttopology.ValidateSource("devices: []\n"); err != nil {
		t.Fatalf("inline source: %v", err)
	}
	err := drafttopology.ValidateSource("segments:\n  - tag: 200\n    config: campus.yaml\n")
	if !drafttopology.IsInvalid(err) {
		t.Fatalf("config-backed source error = %v, want invalid", err)
	}
}

func TestApplyAddAndMoveDevice(t *testing.T) {
	cfg := &config.Config{}
	err := drafttopology.Apply(cfg, drafttopology.Mutation{
		Operation: drafttopology.AddDevice,
		Device: &drafttopology.DeviceMutation{
			Name: "access-1", Type: "switch", MAC: "00:11:22:33:44:55",
			SysObjectID: "1.3.6.1.4.1.9.1.2494",
			IPs: []string{
				"192.0.2.10",
			}, Interfaces: []drafttopology.Interface{{Name: "Ethernet1/1", Type: "ethernet", MTU: 1500, Speed: 1000, Duplex: "full", AdminStatus: "up", OperStatus: "up"}},
		},
	})
	if err != nil {
		t.Fatalf("add: %v", err)
	}
	if len(cfg.Devices) != 1 || cfg.Devices[0].Name != "access-1" {
		t.Fatalf("devices = %+v", cfg.Devices)
	}
	if cfg.Devices[0].Properties["sysObjectID"] != "1.3.6.1.4.1.9.1.2494" {
		t.Fatalf("sysObjectID = %q", cfg.Devices[0].Properties["sysObjectID"])
	}

	err = drafttopology.Apply(cfg, drafttopology.Mutation{
		Operation: drafttopology.MoveDevice,
		Position:  &drafttopology.PositionMutation{Device: "access-1", X: 125.5, Y: -42},
	})
	if err != nil {
		t.Fatalf("move: %v", err)
	}
	if cfg.Devices[0].Properties["topology_x"] != "125.5" ||
		cfg.Devices[0].Properties["topology_y"] != "-42" {
		t.Fatalf("position properties = %+v", cfg.Devices[0].Properties)
	}
}

func TestApplyRejectsUnknownOrMismatchedOperations(t *testing.T) {
	tests := []drafttopology.Mutation{
		{Operation: "rename"},
		{Operation: drafttopology.AddDevice},
		{
			Operation: drafttopology.MoveDevice,
			Device:    &drafttopology.DeviceMutation{Name: "unexpected"},
		},
	}
	for _, mutation := range tests {
		if err := drafttopology.Apply(testConfig(), mutation); !drafttopology.IsInvalid(err) {
			t.Errorf("Apply(%q) error = %v, want invalid", mutation.Operation, err)
		}
	}
}

func assertReciprocalLink(t *testing.T, cfg *config.Config, vlans []int, native int) {
	t.Helper()
	if len(cfg.Devices[0].TrunkPorts) != 1 || len(cfg.Devices[1].TrunkPorts) != 1 {
		t.Fatalf(
			"trunk counts = %d, %d",
			len(cfg.Devices[0].TrunkPorts),
			len(cfg.Devices[1].TrunkPorts),
		)
	}
	left, right := cfg.Devices[0].TrunkPorts[0], cfg.Devices[1].TrunkPorts[0]
	if left.RemoteDevice != "dist-1" || left.RemoteInterface != "Ethernet1/49" ||
		right.RemoteDevice != "core-1" || right.RemoteInterface != "Ethernet1/1" {
		t.Fatalf("link is not reciprocal: left=%+v right=%+v", left, right)
	}
	if left.NativeVLAN != native || right.NativeVLAN != native || len(left.VLANs) != len(vlans) ||
		len(right.VLANs) != len(vlans) {
		t.Fatalf("link properties are not mirrored: left=%+v right=%+v", left, right)
	}
}
