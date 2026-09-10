package protocols

import (
	"sync"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/logging"
)

func TestPeriodicDiscoveryDuringSegmentReload(t *testing.T) {
	advertisers := discoveryAdvertisers()
	delete(advertisers, "STP")
	for name, advertise := range advertisers {
		t.Run(name, func(t *testing.T) {
			cfg := &config.Config{Segments: []config.Segment{
				{Tag: 200, Devices: []config.Device{discoveryDevice("hospital", "02:00:00:00:02:00")}},
			}}
			stack := NewStack(nil, cfg, logging.NewDebugConfig(0))
			var workers sync.WaitGroup
			workers.Go(func() {
				for range 20 {
					advertise(stack)
				}
			})
			for range 20 {
				if err := stack.ReloadConfig(cfg); err != nil {
					t.Error(err)
				}
			}
			workers.Wait()
			if len(stack.sendQueue) != 20 {
				t.Fatalf("queued %d advertisements during reload, want 20", len(stack.sendQueue))
			}
		})
	}
}

func TestPeriodicDiscoveryUsesExplicitSegments(t *testing.T) {
	advertisers := discoveryAdvertisers()
	delete(advertisers, "STP")
	for name, advertise := range advertisers {
		t.Run(name, func(t *testing.T) {
			cfg := &config.Config{Segments: []config.Segment{
				{Tag: 200, Devices: []config.Device{discoveryDevice("hospital", "02:00:00:00:02:00")}},
				{Tag: 300, Devices: []config.Device{discoveryDevice("warehouse", "02:00:00:00:03:00")}},
			}}
			cfg.Segments[1].Devices[0].VLAN = 999
			stack := NewStack(nil, cfg, logging.NewDebugConfig(0))
			advertise(stack)
			if len(stack.sendQueue) != len(cfg.Segments) {
				t.Fatalf("queued %d advertisements, want %d", len(stack.sendQueue), len(cfg.Segments))
			}
			seen := make(map[*config.Device]bool)
			for range cfg.Segments {
				assertDiscoverySegmentPacket(t, stack, <-stack.sendQueue, seen)
			}
		})
	}
}

func assertDiscoverySegmentPacket(t *testing.T, stack *Stack, packet *Packet, seen map[*config.Device]bool) {
	t.Helper()
	device, ok := packet.Device.(*config.Device)
	if !ok || seen[device] {
		t.Fatalf("missing or duplicate device identity: %v", packet.Device)
	}
	seen[device] = true
	for _, segment := range stack.Config().Segments {
		if device != &segment.Devices[0] {
			continue
		}
		frame, vlan, err := stack.finalizeEgressFrame(packet)
		if err != nil || vlan != segment.Tag {
			t.Fatalf("%s egress VLAN=%d err=%v, want %d", device.Name, vlan, err, segment.Tag)
		}
		assertTaggedFrame(t, frame, segment.Tag)
		return
	}
	t.Fatal("advertisement device is not in the authored segments")
}
