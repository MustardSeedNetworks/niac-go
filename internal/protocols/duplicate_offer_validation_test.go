package protocols

import (
	"errors"
	"net/netip"
	"strings"
	"testing"
	"time"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
)

func TestSelectedAttachmentPreflightRejectsRemoteDuplicateOffer(t *testing.T) {
	stack, cfg := isolationRoutedDHCP(t)
	cfg.BehaviorTimelines = []config.BehaviorTimeline{{
		Name: "remote-conflict", RepeatCount: 1,
		Phases: []config.BehaviorPhase{{Name: "offer", Duration: time.Second, Faults: []config.BehaviorFault{
			{
				Device:  "remote",
				Type:    string(devicestate.FaultDuplicateDHCPOffer),
				Address: netip.MustParseAddr("10.20.0.1"),
			},
		}}},
	}}
	if err := stack.SetDeviceAddressFault(
		"remote",
		devicestate.FaultDuplicateDHCPOffer,
		netip.MustParseAddr("10.20.0.1"),
	); err == nil {
		t.Fatal("remote fault unexpectedly available on selected attachment")
	}
	if err := stack.ValidateBehaviorTargets(); err == nil {
		t.Fatal("startup preflight accepted a fault unavailable on the selected attachment")
	}
}

func TestRoutedDuplicateOfferRejectsSubnetEndpoints(t *testing.T) {
	for _, address := range []string{"10.10.200.0", "10.10.200.255"} {
		t.Run(address, func(t *testing.T) {
			stack, cfg := isolationRoutedDHCP(t)
			for i := range cfg.Devices {
				if cfg.Devices[i].Name != "b" {
					continue
				}
				if err := stack.deviceStates[&cfg.Devices[i]].UpdateInterface(
					"eth0",
					func(iface devicestate.Interface) (devicestate.Interface, error) {
						iface.Address = netip.MustParsePrefix(address + "/24")
						return iface, nil
					},
				); err != nil {
					t.Fatal(err)
				}
			}
			if err := stack.SetDeviceAddressFault(
				"a",
				devicestate.FaultDuplicateDHCPOffer,
				netip.MustParseAddr(address),
			); err == nil {
				t.Fatalf("accepted subnet endpoint %s as conflicting unicast host", address)
			}
		})
	}
}

func TestAuthoredDuplicateOfferRequiresEffectivePeerAddress(t *testing.T) {
	body := `devices:
  - name: server
    mac: '02:00:00:00:00:01'
    ips: [192.0.2.1]
    dhcp: {pool_start: 192.0.2.100, pool_end: 192.0.2.110}
  - name: peer
    mac: '02:00:00:00:00:02'
    ips: [192.0.2.20]
    interfaces: [{name: eth0, address: 192.0.2.30/24}]
behavior_timelines:
  - name: conflict
    repeat_count: 1
    phases:
      - name: offer
        duration_ms: 10
        faults:
          - {device: server, type: duplicate_dhcp_offer, address: 192.0.2.20}
`
	if _, err := config.LoadYAMLBytes([]byte(body)); !errors.Is(err, config.ErrBehaviorAddressTarget) {
		t.Fatalf("shadowed peer address: got %v, want address target error", err)
	}
}

func TestAuthoredDuplicateOfferRejectsFlatSubnetEndpoints(t *testing.T) {
	for _, address := range []string{"192.0.2.0", "192.0.2.255"} {
		body := `devices:
  - name: server
    mac: '02:00:00:00:00:01'
    ips: [192.0.2.1]
    dhcp: {pool_start: 192.0.2.100, pool_end: 192.0.2.110}
  - name: peer
    mac: '02:00:00:00:00:02'
    interfaces: [{name: eth0, address: ADDRESS/24}]
behavior_timelines:
  - name: conflict
    repeat_count: 1
    phases:
      - name: offer
        duration_ms: 10
        faults:
          - {device: server, type: duplicate_dhcp_offer, address: ADDRESS}
`
		_, err := config.LoadYAMLBytes([]byte(strings.ReplaceAll(body, "ADDRESS", address)))
		if !errors.Is(err, config.ErrBehaviorAddressTarget) {
			t.Errorf("subnet endpoint %s: got %v, want address target error", address, err)
		}
	}
}
