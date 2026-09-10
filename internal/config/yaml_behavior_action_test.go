package config_test

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
)

func TestDeviceActionTimelineRoundTrip(t *testing.T) {
	for _, kind := range []string{"reboot", "stp_topology_change"} {
		t.Run(kind, func(t *testing.T) {
			cfg, err := config.LoadYAMLBytes(actionTimeline("{device: switch-1, type: " + kind + "}"))
			if err != nil {
				t.Fatal(err)
			}
			data, err := config.MarshalConfigYAML(cfg)
			if err != nil {
				t.Fatal(err)
			}
			reloaded, err := config.LoadYAMLBytes(data)
			if err != nil || !reflect.DeepEqual(cfg.BehaviorTimelines, reloaded.BehaviorTimelines) {
				t.Fatalf("round trip: %v", err)
			}
		})
	}
}

func TestDeviceActionTimelineRejectsInvalidActions(t *testing.T) {
	for _, actions := range []string{
		"{device: switch-1, type: unknown}", "{device: missing, type: reboot}",
		"{device: switch-1, type: reboot, value: 1}", "{device: switch-1, type: reboot, interface: eth0}",
		"{device: switch-1, type: reboot}, {device: switch-1, type: reboot}",
	} {
		t.Run(actions, func(t *testing.T) {
			if _, err := config.LoadYAMLBytes(actionTimeline(actions)); err == nil {
				t.Fatal("invalid action accepted")
			}
		})
	}
}

func actionTimeline(actions string) []byte {
	return fmt.Appendf(nil, `devices:
  - name: switch-1
    type: switch
    mac: "02:00:00:00:00:01"
behavior_timelines:
  - name: actions
    repeat_count: 1
    phases:
      - name: change
        duration_ms: 1000
        reset: true
        actions: [%s]
`, actions)
}
