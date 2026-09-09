package config_test

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
)

func TestBehaviorTimelineValidationRejectsCrossTimelineTargetOverlap(t *testing.T) {
	yamlConfig := []byte(`
devices:
  - name: access-1
    type: switch
    mac: "02:00:00:00:00:01"
    interfaces:
      - name: Gi0/48
behavior_timelines:
  - name: first
    repeat_count: 1
    phases:
      - name: warning
        duration_ms: 2000
        reset: true
        faults:
          - device: access-1
            interface: Gi0/48
            type: fcs_errors
            value: 5
  - name: second
    start_offset_ms: 1000
    repeat_count: 1
    phases:
      - name: critical
        duration_ms: 2000
        reset: true
        faults:
          - device: access-1
            interface: Gi0/48
            type: fcs_errors
            value: 25
`)

	_, err := config.LoadYAMLBytes(yamlConfig)
	if !errors.Is(err, config.ErrBehaviorPhaseOverlap) {
		t.Fatalf("LoadYAMLBytes() error = %v, want %v", err, config.ErrBehaviorPhaseOverlap)
	}
}

func TestBehaviorTimelineValidationRejectsPersistentCrossTimelineConflict(t *testing.T) {
	yamlConfig := []byte(`
devices:
  - name: access-1
    type: switch
    mac: "02:00:00:00:00:01"
    interfaces: [{name: Gi0/48}]
behavior_timelines:
  - name: persistent
    repeat_count: 1
    phases:
      - name: baseline
        duration_ms: 1000
        faults: [{device: access-1, interface: Gi0/48, type: fcs_errors, value: 5}]
  - name: later
    start_offset_ms: 5000
    repeat_count: 1
    phases:
      - name: override
        duration_ms: 1000
        reset: true
        faults: [{device: access-1, interface: Gi0/48, type: fcs_errors, value: 25}]
`)

	_, err := config.LoadYAMLBytes(yamlConfig)
	if !errors.Is(err, config.ErrBehaviorPhaseOverlap) {
		t.Fatalf("LoadYAMLBytes() error = %v, want %v", err, config.ErrBehaviorPhaseOverlap)
	}
}

func TestBehaviorTimelineValidationBoundsCompiledActions(t *testing.T) {
	fault := "          - device: access-1\n            interface: Gi0/48\n            type: fcs_errors\n            value: 5\n"
	yamlConfig := `
devices:
  - name: access-1
    type: switch
    mac: "02:00:00:00:00:01"
    interfaces: [{name: Gi0/48}]
behavior_timelines:
  - name: oversized
    repeat_count: 1000
    phases:
      - name: phase
        duration_ms: 1000
        reset: true
        faults:
` + strings.Repeat(fault, 51)

	_, err := config.LoadYAMLBytes([]byte(yamlConfig))
	if !errors.Is(err, config.ErrBehaviorScheduleTooLarge) {
		t.Fatalf("LoadYAMLBytes() error = %v, want %v", err, config.ErrBehaviorScheduleTooLarge)
	}
}

func TestBehaviorTimelineValidationRejectsNestedCrossTimelineOverlap(t *testing.T) {
	yamlConfig := []byte(`
devices:
  - name: access-1
    type: switch
    mac: "02:00:00:00:00:01"
    interfaces:
      - name: Gi0/48
behavior_timelines:
  - name: long
    repeat_count: 1
    phases:
      - name: enclosing
        duration_ms: 100000
        faults: [{device: access-1, interface: Gi0/48, type: fcs_errors, value: 5}]
  - name: nested
    repeat_count: 1
    phases:
      - name: first
        start_offset_ms: 10000
        duration_ms: 10000
        faults: [{device: access-1, interface: Gi0/48, type: fcs_errors, value: 10}]
      - name: second
        start_offset_ms: 30000
        duration_ms: 10000
        faults: [{device: access-1, interface: Gi0/48, type: fcs_errors, value: 15}]
`)

	_, err := config.LoadYAMLBytes(yamlConfig)
	if !errors.Is(err, config.ErrBehaviorPhaseOverlap) {
		t.Fatalf("LoadYAMLBytes() error = %v, want %v", err, config.ErrBehaviorPhaseOverlap)
	}
}

func TestBehaviorTimelineYAMLRoundTrip(t *testing.T) {
	yamlConfig := []byte(`
devices:
  - name: access-1
    type: switch
    vendor: cisco
    ips: [192.0.2.10]
    interfaces:
      - name: Gi0/48
        speed: 10000
        admin_status: up
        oper_status: up
behavior_timelines:
  - name: uplink degradation
    start_offset_ms: 1000
    repeat_count: 2
    phases:
      - name: congested
        start_offset_ms: 2000
        duration_ms: 3000
        reset: true
        traffic:
          - device: access-1
            interface: Gi0/48
            utilization: 85
        faults:
          - device: access-1
            interface: Gi0/48
            type: packet_discards
            value: 12
`)

	cfg, err := config.LoadYAMLBytes(yamlConfig)
	if err != nil {
		t.Fatalf("LoadYAMLBytes() error = %v", err)
	}
	timeline := cfg.BehaviorTimelines[0]
	if timeline.StartOffset != time.Second || timeline.RepeatCount != 2 ||
		timeline.Phases[0].Duration != 3*time.Second {
		t.Fatalf("loaded timeline = %+v", timeline)
	}
	rendered, err := config.MarshalConfigYAML(cfg)
	if err != nil {
		t.Fatalf("MarshalConfigYAML() error = %v", err)
	}
	reloaded, err := config.LoadYAMLBytes(rendered)
	if err != nil {
		t.Fatalf("round-trip LoadYAMLBytes() error = %v\n%s", err, rendered)
	}
	if len(reloaded.BehaviorTimelines) != 1 ||
		reloaded.BehaviorTimelines[0].Phases[0].Traffic[0].Utilization != 85 {
		t.Fatalf("round-trip timelines = %+v", reloaded.BehaviorTimelines)
	}
}

func TestBehaviorTimelineValidationRejectsInvalidTargetsAndTiming(t *testing.T) {
	tests := map[string]string{
		"unknown device": `device: missing
            interface: Gi0/48
            utilization: 80`,
		"unknown interface": `device: access-1
            interface: Gi0/99
            utilization: 80`,
		"zero duration": `device: access-1
            interface: Gi0/48
            utilization: 80`,
	}
	for name, traffic := range tests {
		t.Run(name, func(t *testing.T) {
			duration := "duration_ms: 1000"
			if name == "zero duration" {
				duration = "duration_ms: 0"
			}
			yamlConfig := "devices:\n  - name: access-1\n    type: switch\n    vendor: cisco\n" +
				"    ips: [192.0.2.10]\n    interfaces:\n      - name: Gi0/48\n" +
				"behavior_timelines:\n  - name: test\n    repeat_count: 1\n    phases:\n" +
				"      - name: phase\n        " + duration + "\n        reset: true\n" +
				"        traffic:\n          - " + traffic + "\n"
			if _, err := config.LoadYAMLBytes([]byte(yamlConfig)); err == nil {
				t.Fatal("LoadYAMLBytes() accepted invalid behavior timeline")
			}
		})
	}
}

// A phase fault with no interface is device-scoped: the axis P2-1 added for
// service outcomes, authored rather than armed over the API. Omitting the
// interface is the discriminator because a service outage has no interface to
// be keyed by.
func TestBehaviorTimelineAuthorsADeviceScopedFault(t *testing.T) {
	yamlConfig := []byte(`
networks:
  - name: lan
    subnet: 10.0.0.0/24
devices:
  - name: server-1
    type: server
    mac: "02:00:00:00:00:01"
    interfaces: [{name: eth0, type: ethernet, network: lan, address: 10.0.0.5/24}]
    dhcp:
      pool_start: 10.0.0.100
      pool_end: 10.0.0.199
      subnet_mask: 255.255.255.0
      router: 10.0.0.1
behavior_timelines:
  - name: outage
    repeat_count: 1
    phases:
      - name: no-offer
        duration_ms: 5000
        reset: true
        faults: [{device: server-1, type: dhcp_no_offer, value: 1}]
`)

	cfg, err := config.LoadYAMLBytes(yamlConfig)
	if err != nil {
		t.Fatalf("LoadYAMLBytes() error = %v", err)
	}
	fault := cfg.BehaviorTimelines[0].Phases[0].Faults[0]
	if fault.Interface != "" {
		t.Errorf("fault.Interface = %q, want empty — the fault is device-scoped", fault.Interface)
	}
	if fault.Type != "dhcp_no_offer" {
		t.Errorf("fault.Type = %q, want dhcp_no_offer", fault.Type)
	}
}

// A device-scoped fault named against an interface would claim a scope it does
// not have, and an interface fault with no interface has nothing to apply to.
// Both are refused at load rather than at apply time.
func TestBehaviorTimelineRejectsMismatchedFaultScope(t *testing.T) {
	for name, fault := range map[string]string{
		"device fault with an interface": "{device: access-1, interface: Gi0/48, type: dhcp_no_offer, value: 1}",
		"interface fault with none":      "{device: access-1, type: fcs_errors, value: 5}",
	} {
		t.Run(name, func(t *testing.T) {
			yamlConfig := []byte(`
devices:
  - name: access-1
    type: switch
    mac: "02:00:00:00:00:01"
    interfaces: [{name: Gi0/48}]
behavior_timelines:
  - name: mismatch
    repeat_count: 1
    phases:
      - name: bad
        duration_ms: 1000
        faults: [` + fault + `]
`)

			_, err := config.LoadYAMLBytes(yamlConfig)
			if !errors.Is(err, config.ErrBehaviorFaultScope) {
				t.Fatalf("LoadYAMLBytes() error = %v, want %v", err, config.ErrBehaviorFaultScope)
			}
		})
	}
}

// The rate faults stop at 100 and latency is milliseconds; a single ceiling
// would either cap a delay at a tenth of a second or let a rate be authored
// at 60000. The per-type ceiling is devicestate's, not this package's.
func TestBehaviorTimelineAppliesThePerTypeFaultCeiling(t *testing.T) {
	timeline := func(faultType string, value int) []byte {
		return fmt.Appendf(nil, `
networks:
  - name: lan
    subnet: 10.0.0.0/24
devices:
  - name: server-1
    type: server
    mac: "02:00:00:00:00:01"
    interfaces: [{name: eth0, type: ethernet, network: lan, address: 10.0.0.5/24}]
behavior_timelines:
  - name: ceiling
    repeat_count: 1
    phases:
      - name: apply
        duration_ms: 1000
        faults: [{device: server-1, type: %s, value: %d}]
`, faultType, value)
	}

	if _, err := config.LoadYAMLBytes(timeline("latency", 2500)); err != nil {
		t.Errorf("LoadYAMLBytes(latency 2500ms) error = %v, want accepted", err)
	}
	if _, err := config.LoadYAMLBytes(timeline("dns_nxdomain", 2500)); !errors.Is(
		err, config.ErrBehaviorFaultValue,
	) {
		t.Errorf("LoadYAMLBytes(dns_nxdomain 2500) error = %v, want %v", err, config.ErrBehaviorFaultValue)
	}
}
