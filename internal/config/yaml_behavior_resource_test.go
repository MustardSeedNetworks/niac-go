package config_test

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
)

func TestResourceTimelineRoundTrip(t *testing.T) {
	for _, kind := range []string{"cpu_percent", "memory_percent", "disk_percent"} {
		for _, value := range []int{1, 50, 100} {
			t.Run(fmt.Sprintf("%s/%d", kind, value), func(t *testing.T) {
				cfg, err := config.LoadYAMLBytes(resourceTimeline(kind, "", value))
				if err != nil {
					t.Fatal(err)
				}
				rendered, err := config.MarshalConfigYAML(cfg)
				if err != nil {
					t.Fatal(err)
				}
				reloaded, err := config.LoadYAMLBytes(rendered)
				if err != nil || !reflect.DeepEqual(cfg.BehaviorTimelines, reloaded.BehaviorTimelines) {
					t.Fatalf("resource timeline changed on round trip: %v", err)
				}
			})
		}
	}
}

func TestResourceTimelineRejectsInvalidScopeAndValues(t *testing.T) {
	for _, kind := range []string{"cpu_percent", "memory_percent", "disk_percent"} {
		for _, value := range []int{-1, 0, 101} {
			if _, err := config.LoadYAMLBytes(resourceTimeline(kind, "", value)); err == nil {
				t.Fatalf("accepted %s value %d", kind, value)
			}
		}
		if _, err := config.LoadYAMLBytes(resourceTimeline(kind, "eth0", 50)); err == nil {
			t.Fatalf("accepted interface-scoped %s", kind)
		}
	}
}

func resourceTimeline(kind, iface string, value int) []byte {
	return fmt.Appendf(nil, `devices:
  - name: server-1
    type: server
    mac: "02:00:00:00:00:01"
    interfaces: [{name: eth0}]
behavior_timelines:
  - name: resource-pressure
    repeat_count: 1
    phases:
      - name: high-load
        duration_ms: 1000
        reset: true
        faults: [{device: server-1, interface: %q, type: %s, value: %d}]
`, iface, kind, value)
}
