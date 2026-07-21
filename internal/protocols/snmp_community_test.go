package protocols

import (
	"net"
	"slices"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/logging"
)

func TestSNMPFindAgentRequiresConfiguredCommunity(t *testing.T) {
	cfg := &config.Config{Devices: []config.Device{{
		Name:        "switch-1",
		MACAddress:  net.HardwareAddr{0x02, 0, 0, 0, 0, 1},
		IPAddresses: []net.IP{net.ParseIP("192.0.2.10")},
		SNMPConfig: config.SNMPConfig{
			Community: "NetAllyDemo",
			SysName:   "switch-1",
		},
	}}}
	stack := NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := NewSNMPHandler(stack)
	device := &cfg.Devices[0]
	source := net.ParseIP("192.0.2.20")

	if handler.findAgent(device, source, "NetAllyDemo") == nil {
		t.Fatal("configured community did not resolve an agent")
	}
	if handler.findAgent(device, source, "public") != nil {
		t.Error("unconfigured public community resolved an agent")
	}
	if handler.findAgent(device, source, "wrong-community") != nil {
		t.Error("unknown community resolved an agent")
	}
	agent := handler.findAgent(device, source, "NetAllyDemo")
	badNames, err := agent.HandleGet("1.3.6.1.2.1.11.4.0")
	if err != nil || badNames.Value != uint32(2) {
		t.Fatalf("snmpInBadCommunityNames = %#v, err=%v", badNames, err)
	}
}

func TestSNMPFindAgentCountsDisallowedCommunityUse(t *testing.T) {
	allowed := net.ParseIP("192.0.2.20")
	cfg := &config.Config{Devices: []config.Device{{
		Name: "switch-1", MACAddress: net.HardwareAddr{0x02, 0, 0, 0, 0, 1},
		IPAddresses: []net.IP{net.ParseIP("192.0.2.10")},
		SNMPConfig:  config.SNMPConfig{Community: "NetAllyDemo", AccessList: []net.IP{allowed}},
	}}}
	stack := NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := NewSNMPHandler(stack)
	device := &cfg.Devices[0]
	if handler.findAgent(device, net.ParseIP("198.51.100.1"), "NetAllyDemo") != nil {
		t.Fatal("disallowed source resolved an agent")
	}
	agent := handler.findAgent(device, allowed, "NetAllyDemo")
	badUses, err := agent.HandleGet("1.3.6.1.2.1.11.5.0")
	if err != nil || badUses.Value != uint32(1) {
		t.Fatalf("snmpInBadCommunityUses = %#v, err=%v", badUses, err)
	}
}

func TestConfiguredWalkFilesDeduplicatesSingularCompatibilityField(t *testing.T) {
	cfg := config.SNMPConfig{
		WalkFile:  "base.walk",
		WalkFiles: []string{"base.walk", "overlay.walk", "overlay.walk"},
	}

	if got, want := configuredWalkFiles(cfg), []string{"base.walk", "overlay.walk"}; !slices.Equal(got, want) {
		t.Fatalf("configuredWalkFiles() = %v, want %v", got, want)
	}
}
