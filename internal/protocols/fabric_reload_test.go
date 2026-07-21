package protocols

import (
	"errors"
	"net/netip"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/logging"
)

func TestReloadConfigRecompilesRoutedFabric(t *testing.T) {
	cfg, topology, routerMAC := forwardingFixture(t)
	stack := NewStack(nil, cfg, logging.NewDebugConfig(0))
	stack.ConfigureFabric(topology)

	replacement, _, _ := forwardingFixture(t)
	replacement.Devices[1].Interfaces[0].Address = "10.20.0.20/24"
	if err := stack.ReloadConfig(replacement); err != nil {
		t.Fatalf("ReloadConfig(): %v", err)
	}

	if _, ok := stack.fabric.resolveIPv4(netip.MustParseAddr("10.20.0.10"), routerMAC); ok {
		t.Fatal("stale routed endpoint survived reload")
	}
	resolution, ok := stack.fabric.resolveIPv4(netip.MustParseAddr("10.20.0.20"), routerMAC)
	if !ok || resolution.device != &replacement.Devices[1] {
		t.Fatalf("replacement resolution = %#v, want replacement server", resolution)
	}
}

func TestReloadConfigRejectsUnsafeRoutedReplacementTransactionally(t *testing.T) {
	cfg, topology, routerMAC := forwardingFixture(t)
	stack := NewStack(nil, cfg, logging.NewDebugConfig(0))
	stack.ConfigureFabric(topology)

	replacement, _, _ := forwardingFixture(t)
	replacement.Devices[1].Interfaces[0].Address = "10.20.0.1/24"
	err := stack.ReloadConfig(replacement)
	if !errors.Is(err, ErrUnsafeFabricReload) {
		t.Fatalf("ReloadConfig() error = %v, want %v", err, ErrUnsafeFabricReload)
	}

	resolution, ok := stack.fabric.resolveIPv4(netip.MustParseAddr("10.20.0.10"), routerMAC)
	if !ok || resolution.device != &cfg.Devices[1] {
		t.Fatalf("original resolution after rejected reload = %#v", resolution)
	}
}
