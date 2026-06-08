package device

import (
	"errors"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/MustardSeedNetworks/niac-go/internal/apperr"
	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/logging"
	"github.com/MustardSeedNetworks/niac-go/internal/protocols"
)

// createTrafficTestConfig builds a config with devices that have TrafficConfig set.
func createTrafficTestConfig(count int, trafficEnabled bool) *config.Config {
	cfg := &config.Config{
		Devices: make([]config.Device, count),
	}
	for i := range count {
		cfg.Devices[i] = config.Device{
			Name:        fmt.Sprintf("traf-dev-%d", i),
			Type:        "router",
			MACAddress:  net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, byte(0x55 + i)},
			IPAddresses: []net.IP{net.ParseIP(fmt.Sprintf("192.168.1.%d", i+1))},
			SNMPConfig: config.SNMPConfig{
				Community: "public",
			},
		}
		if trafficEnabled {
			cfg.Devices[i].TrafficConfig = &config.TrafficConfig{
				Enabled: true,
				ARPAnnouncements: &config.ARPAnnouncementConfig{
					Enabled:  true,
					Interval: 60,
				},
				PeriodicPings: &config.PeriodicPingConfig{
					Enabled:     true,
					Interval:    120,
					PayloadSize: 32,
				},
				RandomTraffic: &config.RandomTrafficConfig{
					Enabled:     true,
					Interval:    180,
					PacketCount: 1,
					Patterns:    []string{"broadcast_arp", "udp"},
				},
			}
		}
	}
	return cfg
}

func newTestSimAndStack(cfg *config.Config) (*Simulator, *protocols.Stack) {
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
	errorMgr := apperr.NewStateManager()
	sim := NewSimulator(cfg, stack, errorMgr, 0)
	return sim, stack
}

// TestNewTrafficGenerator tests creating a new traffic generator.
func TestNewTrafficGenerator(t *testing.T) {
	cfg := createTrafficTestConfig(1, true)
	sim, stack := newTestSimAndStack(cfg)

	tg := NewTrafficGenerator(sim, stack, 0)
	if tg == nil {
		t.Fatal("Expected traffic generator, got nil")
	}
	if tg.simulator != sim {
		t.Error("Simulator reference not set")
	}
	if tg.stack != stack {
		t.Error("Stack reference not set")
	}
	if tg.running.Load() {
		t.Error("Should not be running initially")
	}
	if tg.patterns == nil {
		t.Error("Patterns map should be initialized")
	}
}

// TestTrafficGenerator_StartStop tests the start/stop lifecycle.
func TestTrafficGenerator_StartStop(t *testing.T) {
	cfg := createTrafficTestConfig(1, false)
	sim, stack := newTestSimAndStack(cfg)
	tg := NewTrafficGenerator(sim, stack, 0)

	// Start
	err := tg.Start()
	if err != nil {
		t.Fatalf("Failed to start: %v", err)
	}
	if !tg.running.Load() {
		t.Error("Should be running after Start()")
	}

	// Stop
	tg.Stop()
	if tg.running.Load() {
		t.Error("Should not be running after Stop()")
	}
}

// TestTrafficGenerator_DoubleStart tests that starting twice returns an error.
func TestTrafficGenerator_DoubleStart(t *testing.T) {
	cfg := createTrafficTestConfig(1, false)
	sim, stack := newTestSimAndStack(cfg)
	tg := NewTrafficGenerator(sim, stack, 0)

	err := tg.Start()
	if err != nil {
		t.Fatalf("First start failed: %v", err)
	}
	defer tg.Stop()

	err = tg.Start()
	if !errors.Is(err, ErrTrafficGeneratorAlreadyRunning) {
		t.Errorf("Expected ErrTrafficGeneratorAlreadyRunning, got %v", err)
	}
}

// TestTrafficGenerator_DoubleStop tests that stopping twice is safe (no panic).
func TestTrafficGenerator_DoubleStop(t *testing.T) {
	cfg := createTrafficTestConfig(1, false)
	sim, stack := newTestSimAndStack(cfg)
	tg := NewTrafficGenerator(sim, stack, 0)

	err := tg.Start()
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	tg.Stop()
	// Second stop should be a no-op, not panic
	tg.Stop()
}

// TestTrafficGenerator_StopWithoutStart tests that Stop on a never-started generator is safe.
func TestTrafficGenerator_StopWithoutStart(_ *testing.T) {
	cfg := createTrafficTestConfig(1, false)
	sim, stack := newTestSimAndStack(cfg)
	tg := NewTrafficGenerator(sim, stack, 0)

	// Should not panic
	tg.Stop()
}

// TestTrafficGenerator_RestartAfterStop verifies that Start can be called
// again after Stop without panicking on `close of closed channel`.
// Regression for #463.
func TestTrafficGenerator_RestartAfterStop(t *testing.T) {
	cfg := createTrafficTestConfig(1, false)
	sim, stack := newTestSimAndStack(cfg)
	tg := NewTrafficGenerator(sim, stack, 0)

	for i := range 3 {
		if err := tg.Start(); err != nil {
			t.Fatalf("iter %d: Start failed: %v", i, err)
		}
		tg.Stop()
	}
}

// TestTrafficGenerator_AtomicBool tests that the [atomic.Bool] in TrafficGenerator
// works correctly under concurrent access (regression test for [atomic.Bool] usage).
func TestTrafficGenerator_AtomicBool(t *testing.T) {
	cfg := createTrafficTestConfig(1, false)
	sim, stack := newTestSimAndStack(cfg)
	tg := NewTrafficGenerator(sim, stack, 0)

	// Verify initial false
	if tg.running.Load() {
		t.Error("running should be false initially")
	}

	// CompareAndSwap false -> true
	if !tg.running.CompareAndSwap(false, true) {
		t.Error("CAS false->true should succeed")
	}

	// CompareAndSwap false -> true again should fail
	if tg.running.CompareAndSwap(false, true) {
		t.Error("CAS false->true should fail when already true")
	}

	// CompareAndSwap true -> false
	if !tg.running.CompareAndSwap(true, false) {
		t.Error("CAS true->false should succeed")
	}
}

// TestTrafficGenerator_ShouldGenerateTraffic tests the shouldGenerateTraffic predicate.
func TestTrafficGenerator_ShouldGenerateTraffic(t *testing.T) {
	tests := []struct {
		name     string
		state    State
		traffic  *config.TrafficConfig
		expected bool
	}{
		{
			name:     "device up with traffic enabled",
			state:    StateUp,
			traffic:  &config.TrafficConfig{Enabled: true},
			expected: true,
		},
		{
			name:     "device down with traffic enabled",
			state:    StateDown,
			traffic:  &config.TrafficConfig{Enabled: true},
			expected: false,
		},
		{
			name:     "device up with traffic disabled",
			state:    StateUp,
			traffic:  &config.TrafficConfig{Enabled: false},
			expected: false,
		},
		{
			name:     "device up with nil traffic config",
			state:    StateUp,
			traffic:  nil,
			expected: false,
		},
		{
			name:     "device starting with traffic enabled",
			state:    StateStarting,
			traffic:  &config.TrafficConfig{Enabled: true},
			expected: false,
		},
		{
			name:     "device in maintenance with traffic enabled",
			state:    StateMaintenance,
			traffic:  &config.TrafficConfig{Enabled: true},
			expected: false,
		},
	}

	cfg := createTrafficTestConfig(1, false)
	sim, stack := newTestSimAndStack(cfg)
	tg := NewTrafficGenerator(sim, stack, 0)

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dev := &SimulatedDevice{
				Config: &config.Device{
					TrafficConfig: tc.traffic,
				},
				State: tc.state,
			}
			got := tg.shouldGenerateTraffic(dev)
			if got != tc.expected {
				t.Errorf("shouldGenerateTraffic() = %v, want %v", got, tc.expected)
			}
		})
	}
}

// TestTrafficGenerator_GetPattern tests pattern creation and retrieval.
func TestTrafficGenerator_GetPattern(t *testing.T) {
	cfg := createTrafficTestConfig(1, false)
	sim, stack := newTestSimAndStack(cfg)
	tg := NewTrafficGenerator(sim, stack, 0)

	// First access creates the pattern
	p1 := tg.getPattern("dev-a", "arp_announcements")
	if p1 == nil {
		t.Fatal("Expected pattern, got nil")
	}
	if p1.Name != "arp_announcements" {
		t.Errorf("Expected name 'arp_announcements', got '%s'", p1.Name)
	}

	// Second access returns the same instance
	p2 := tg.getPattern("dev-a", "arp_announcements")
	if p1 != p2 {
		t.Error("Expected same pattern instance on second access")
	}

	// Different pattern name creates a new one
	p3 := tg.getPattern("dev-a", "periodic_pings")
	if p3 == p1 {
		t.Error("Different pattern name should create different instance")
	}

	// Different device creates a new map
	p4 := tg.getPattern("dev-b", "arp_announcements")
	if p4 == p1 {
		t.Error("Different device should have separate pattern instance")
	}
}

// TestTrafficGenerator_CheckARPAnnouncements tests ARP announcement interval gating.
func TestTrafficGenerator_CheckARPAnnouncements(t *testing.T) {
	cfg := createTrafficTestConfig(2, true)
	sim, stack := newTestSimAndStack(cfg)
	tg := NewTrafficGenerator(sim, stack, 0)

	deviceName := "traf-dev-0"
	device := sim.GetDevice(deviceName)
	trafficCfg := device.Config.TrafficConfig

	// First call should generate (LastRun is zero)
	now := time.Now()
	tg.checkARPAnnouncements(deviceName, device, trafficCfg, now)

	p := tg.getPattern(deviceName, patternARP)
	if p.LastRun.IsZero() {
		t.Error("LastRun should be set after first call")
	}
	if !p.Enabled {
		t.Error("Pattern should be enabled")
	}

	// Immediate second call should NOT generate (interval not elapsed)
	oldLastRun := p.LastRun
	tg.checkARPAnnouncements(deviceName, device, trafficCfg, now.Add(time.Second))
	if p.LastRun != oldLastRun {
		t.Error("LastRun should not change when interval hasn't elapsed")
	}

	// Call after interval should generate
	futureTime := now.Add(61 * time.Second)
	tg.checkARPAnnouncements(deviceName, device, trafficCfg, futureTime)
	if p.LastRun != futureTime {
		t.Error("LastRun should update after interval elapsed")
	}
}

// TestTrafficGenerator_CheckARPAnnouncements_Disabled tests that disabled ARP config is skipped.
func TestTrafficGenerator_CheckARPAnnouncements_Disabled(t *testing.T) {
	cfg := createTrafficTestConfig(1, true)
	sim, stack := newTestSimAndStack(cfg)
	tg := NewTrafficGenerator(sim, stack, 0)

	device := sim.GetDevice("traf-dev-0")

	// Disabled ARP
	disabledCfg := &config.TrafficConfig{
		Enabled: true,
		ARPAnnouncements: &config.ARPAnnouncementConfig{
			Enabled: false,
		},
	}
	tg.checkARPAnnouncements("traf-dev-0", device, disabledCfg, time.Now())
	// Should not create a pattern entry
	if _, exists := tg.patterns["traf-dev-0"]; exists {
		if _, pExists := tg.patterns["traf-dev-0"][patternARP]; pExists {
			t.Error("Should not create pattern for disabled ARP")
		}
	}

	// Nil ARP config
	nilCfg := &config.TrafficConfig{
		Enabled: true,
	}
	tg.checkARPAnnouncements("traf-dev-0", device, nilCfg, time.Now())
}

// TestTrafficGenerator_CheckPeriodicPings tests periodic ping interval gating.
func TestTrafficGenerator_CheckPeriodicPings(t *testing.T) {
	cfg := createTrafficTestConfig(2, true)
	sim, stack := newTestSimAndStack(cfg)
	tg := NewTrafficGenerator(sim, stack, 0)

	deviceName := "traf-dev-0"
	device := sim.GetDevice(deviceName)
	trafficCfg := device.Config.TrafficConfig

	now := time.Now()
	tg.checkPeriodicPings(deviceName, device, trafficCfg, now)

	p := tg.getPattern(deviceName, patternPing)
	if p.LastRun.IsZero() {
		t.Error("LastRun should be set after first call")
	}

	// Not enough time elapsed
	oldLastRun := p.LastRun
	tg.checkPeriodicPings(deviceName, device, trafficCfg, now.Add(10*time.Second))
	if p.LastRun != oldLastRun {
		t.Error("LastRun should not change when interval hasn't elapsed")
	}
}

// TestTrafficGenerator_CheckPeriodicPings_Disabled tests disabled pings are skipped.
func TestTrafficGenerator_CheckPeriodicPings_Disabled(_ *testing.T) {
	cfg := createTrafficTestConfig(1, false)
	sim, stack := newTestSimAndStack(cfg)
	tg := NewTrafficGenerator(sim, stack, 0)

	device := sim.GetDevice("traf-dev-0")

	disabledCfg := &config.TrafficConfig{
		Enabled:       true,
		PeriodicPings: &config.PeriodicPingConfig{Enabled: false},
	}
	tg.checkPeriodicPings("traf-dev-0", device, disabledCfg, time.Now())

	nilCfg := &config.TrafficConfig{Enabled: true}
	tg.checkPeriodicPings("traf-dev-0", device, nilCfg, time.Now())
}

// TestTrafficGenerator_CheckRandomTraffic tests random traffic interval gating.
func TestTrafficGenerator_CheckRandomTraffic(t *testing.T) {
	cfg := createTrafficTestConfig(2, true)
	sim, stack := newTestSimAndStack(cfg)
	tg := NewTrafficGenerator(sim, stack, 0)

	deviceName := "traf-dev-0"
	device := sim.GetDevice(deviceName)
	trafficCfg := device.Config.TrafficConfig

	now := time.Now()
	tg.checkRandomTraffic(deviceName, device, trafficCfg, now)

	p := tg.getPattern(deviceName, patternRandom)
	if p.LastRun.IsZero() {
		t.Error("LastRun should be set after first call")
	}
}

// TestTrafficGenerator_CheckRandomTraffic_Disabled tests disabled random traffic is skipped.
func TestTrafficGenerator_CheckRandomTraffic_Disabled(_ *testing.T) {
	cfg := createTrafficTestConfig(1, false)
	sim, stack := newTestSimAndStack(cfg)
	tg := NewTrafficGenerator(sim, stack, 0)

	device := sim.GetDevice("traf-dev-0")

	disabledCfg := &config.TrafficConfig{
		Enabled:       true,
		RandomTraffic: &config.RandomTrafficConfig{Enabled: false},
	}
	tg.checkRandomTraffic("traf-dev-0", device, disabledCfg, time.Now())

	nilCfg := &config.TrafficConfig{Enabled: true}
	tg.checkRandomTraffic("traf-dev-0", device, nilCfg, time.Now())
}

// TestTrafficGenerator_CheckAndGenerateTraffic tests the top-level check loop.
func TestTrafficGenerator_CheckAndGenerateTraffic(t *testing.T) {
	tests := []struct {
		name           string
		trafficEnabled bool
		deviceState    State
	}{
		{"traffic enabled and device up", true, StateUp},
		{"traffic disabled", false, StateUp},
		{"device down", true, StateDown},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg := createTrafficTestConfig(1, tc.trafficEnabled)
			sim, stack := newTestSimAndStack(cfg)
			tg := NewTrafficGenerator(sim, stack, 0)

			if tc.deviceState != StateUp {
				err := sim.SetDeviceState("traf-dev-0", tc.deviceState)
				if err != nil {
					t.Fatalf("Failed to set state: %v", err)
				}
			}

			// Should not panic regardless of config
			tg.checkAndGenerateTraffic()
		})
	}
}

// TestTrafficGenerator_SendGratuitousARP tests gratuitous ARP packet creation.
func TestTrafficGenerator_SendGratuitousARP(t *testing.T) {
	cfg := createTrafficTestConfig(1, true)
	sim, stack := newTestSimAndStack(cfg)
	tg := NewTrafficGenerator(sim, stack, 0)

	device := sim.GetDevice("traf-dev-0")

	err := tg.sendGratuitousARP(device)
	if err != nil {
		t.Fatalf("sendGratuitousARP failed: %v", err)
	}

	// Check counter was incremented
	counters := sim.GetCounters("traf-dev-0")
	if counters.PacketsSent != 1 {
		t.Errorf("Expected PacketsSent=1, got %d", counters.PacketsSent)
	}
}

// TestTrafficGenerator_SendGratuitousARP_NoMAC tests error when device has no MAC.
func TestTrafficGenerator_SendGratuitousARP_NoMAC(t *testing.T) {
	cfg := createTrafficTestConfig(1, true)
	sim, stack := newTestSimAndStack(cfg)
	tg := NewTrafficGenerator(sim, stack, 0)

	device := &SimulatedDevice{
		Config: &config.Device{
			Name:        "no-mac",
			MACAddress:  nil,
			IPAddresses: []net.IP{net.ParseIP("10.0.0.1")},
		},
		State:    StateUp,
		Counters: &Counters{},
	}

	err := tg.sendGratuitousARP(device)
	if err == nil {
		t.Error("Expected error for device with no MAC")
	}
}

// TestTrafficGenerator_SendGratuitousARP_NoIP tests error when device has no IP.
func TestTrafficGenerator_SendGratuitousARP_NoIP(t *testing.T) {
	cfg := createTrafficTestConfig(1, true)
	sim, stack := newTestSimAndStack(cfg)
	tg := NewTrafficGenerator(sim, stack, 0)

	device := &SimulatedDevice{
		Config: &config.Device{
			Name:       "no-ip",
			MACAddress: net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, 0x55},
		},
		State:    StateUp,
		Counters: &Counters{},
	}

	err := tg.sendGratuitousARP(device)
	if err == nil {
		t.Error("Expected error for device with no IP")
	}
}

// TestTrafficGenerator_SendPing tests ICMP ping packet creation.
func TestTrafficGenerator_SendPing(t *testing.T) {
	cfg := createTrafficTestConfig(2, true)
	sim, stack := newTestSimAndStack(cfg)
	tg := NewTrafficGenerator(sim, stack, 0)

	src := sim.GetDevice("traf-dev-0")
	dst := sim.GetDevice("traf-dev-1")

	err := tg.sendPing(src, dst)
	if err != nil {
		t.Fatalf("sendPing failed: %v", err)
	}

	counters := sim.GetCounters("traf-dev-0")
	if counters.PacketsSent != 1 {
		t.Errorf("Expected PacketsSent=1, got %d", counters.PacketsSent)
	}
}

// TestTrafficGenerator_SendBroadcastARP tests broadcast ARP packet creation.
func TestTrafficGenerator_SendBroadcastARP(t *testing.T) {
	cfg := createTrafficTestConfig(1, true)
	sim, stack := newTestSimAndStack(cfg)
	tg := NewTrafficGenerator(sim, stack, 0)

	device := sim.GetDevice("traf-dev-0")

	err := tg.sendBroadcastARP(device)
	if err != nil {
		t.Fatalf("sendBroadcastARP failed: %v", err)
	}

	counters := sim.GetCounters("traf-dev-0")
	if counters.PacketsSent != 1 {
		t.Errorf("Expected PacketsSent=1, got %d", counters.PacketsSent)
	}
}

// TestTrafficGenerator_SendMulticast tests multicast packet creation.
func TestTrafficGenerator_SendMulticast(t *testing.T) {
	cfg := createTrafficTestConfig(1, true)
	sim, stack := newTestSimAndStack(cfg)
	tg := NewTrafficGenerator(sim, stack, 0)

	device := sim.GetDevice("traf-dev-0")

	err := tg.sendMulticast(device)
	if err != nil {
		t.Fatalf("sendMulticast failed: %v", err)
	}

	counters := sim.GetCounters("traf-dev-0")
	if counters.PacketsSent != 1 {
		t.Errorf("Expected PacketsSent=1, got %d", counters.PacketsSent)
	}
}

// TestTrafficGenerator_SendRandomUDP tests random UDP packet creation.
func TestTrafficGenerator_SendRandomUDP(t *testing.T) {
	cfg := createTrafficTestConfig(2, true)
	sim, stack := newTestSimAndStack(cfg)
	tg := NewTrafficGenerator(sim, stack, 0)

	src := sim.GetDevice("traf-dev-0")
	dst := sim.GetDevice("traf-dev-1")

	err := tg.sendRandomUDP(src, dst)
	if err != nil {
		t.Fatalf("sendRandomUDP failed: %v", err)
	}

	counters := sim.GetCounters("traf-dev-0")
	if counters.PacketsSent != 1 {
		t.Errorf("Expected PacketsSent=1, got %d", counters.PacketsSent)
	}
}

// TestTrafficGenerator_GenerateDeviceTraffic tests that all traffic types are invoked.
func TestTrafficGenerator_GenerateDeviceTraffic(_ *testing.T) {
	cfg := createTrafficTestConfig(2, true)
	sim, stack := newTestSimAndStack(cfg)
	tg := NewTrafficGenerator(sim, stack, 0)

	device := sim.GetDevice("traf-dev-0")

	// Should not panic; exercises all three check* methods
	tg.generateDeviceTraffic("traf-dev-0", device, time.Now())
}

// TestTrafficGenerator_SendPeriodicPing_NoOtherDevices tests ping when only one device exists.
func TestTrafficGenerator_SendPeriodicPing_NoOtherDevices(t *testing.T) {
	cfg := createTrafficTestConfig(1, true)
	sim, stack := newTestSimAndStack(cfg)
	tg := NewTrafficGenerator(sim, stack, 0)

	device := sim.GetDevice("traf-dev-0")

	// Should not panic even with no other devices to ping
	tg.sendPeriodicPing(device, 32)

	// No packets should be sent (no target)
	counters := sim.GetCounters("traf-dev-0")
	if counters.PacketsSent != 0 {
		t.Errorf("Expected PacketsSent=0 with no targets, got %d", counters.PacketsSent)
	}
}

// TestTrafficGenerator_GenerateRandomTrafficForDevice_EmptyPatterns tests default patterns.
func TestTrafficGenerator_GenerateRandomTrafficForDevice_EmptyPatterns(_ *testing.T) {
	cfg := createTrafficTestConfig(2, true)
	sim, stack := newTestSimAndStack(cfg)
	tg := NewTrafficGenerator(sim, stack, 0)

	device := sim.GetDevice("traf-dev-0")

	// Empty patterns should fall back to defaults
	tg.generateRandomTrafficForDevice(device, 1, []string{})
}

// TestTrafficGenerator_GenerateRandomTrafficForDevice_NoDevices tests with single down device.
func TestTrafficGenerator_GenerateRandomTrafficForDevice_NoDevices(t *testing.T) {
	cfg := createTrafficTestConfig(1, true)
	sim, stack := newTestSimAndStack(cfg)
	tg := NewTrafficGenerator(sim, stack, 0)

	// Set the only device to down
	err := sim.SetDeviceState("traf-dev-0", StateDown)
	if err != nil {
		t.Fatalf("Failed to set state: %v", err)
	}

	device := sim.GetDevice("traf-dev-0")

	// Should return early with no up devices
	tg.generateRandomTrafficForDevice(device, 1, []string{"udp"})
}
