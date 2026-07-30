package config_test

import (
	"testing"
	"time"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
)

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
	if len(reloaded.BehaviorTimelines) != 1 || reloaded.BehaviorTimelines[0].Phases[0].Traffic[0].Utilization != 85 {
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
