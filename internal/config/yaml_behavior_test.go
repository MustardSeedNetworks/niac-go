package config_test

import (
	"errors"
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
