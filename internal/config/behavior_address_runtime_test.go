package config

import (
	"errors"
	"net"
	"net/netip"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
)

func TestRuntimeAddressFaultPayloadValidation(t *testing.T) {
	targets := behaviorTargets(&Config{Devices: []Device{
		{Name: "server", DHCPConfig: &DHCPConfig{}},
		{Name: "peer", IPAddresses: []net.IP{net.ParseIP("192.0.2.20")}},
	}})
	valid := BehaviorFault{
		Device:  "server",
		Type:    string(devicestate.FaultDuplicateDHCPOffer),
		Address: netip.MustParseAddr("192.0.2.20"),
	}
	if err := validateBehaviorFault(targets, valid); err != nil {
		t.Fatalf("valid baseline rejected: %v", err)
	}
	for _, fault := range []BehaviorFault{
		{Device: "server", Type: valid.Type},
		{Device: "server", Type: valid.Type, Address: valid.Address, Value: 1},
		{Device: "server", Type: valid.Type, Address: netip.MustParseAddr("224.0.0.1")},
		{Device: "server", Type: "latency", Address: valid.Address, Value: 10},
		{Device: "server", Type: "latency", Value: 0},
	} {
		if err := validateBehaviorFault(targets, fault); !errors.Is(err, ErrBehaviorFaultValue) {
			t.Fatalf("fault %+v error = %v, expected value error", fault, err)
		}
	}
}
