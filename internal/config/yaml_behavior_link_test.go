package config_test

import (
	"errors"
	"fmt"
	"strconv"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
)

func TestBehaviorTimelineAuthorsLinkDown(t *testing.T) {
	for _, value := range []int{1, 100} {
		t.Run(strconv.Itoa(value), func(t *testing.T) {
			cfg, err := config.LoadYAMLBytes(linkDownTimeline("Gi0/1", value))
			if err != nil {
				t.Fatal(err)
			}
			fault := cfg.BehaviorTimelines[0].Phases[0].Faults[0]
			if fault.Type != "link_down" || fault.Interface != "Gi0/1" || fault.Value != value {
				t.Fatalf("link fault changed during load: %+v", fault)
			}
		})
	}
}

func TestBehaviorTimelineRejectsUnscopedLinkDown(t *testing.T) {
	if _, err := config.LoadYAMLBytes(linkDownTimeline("", 1)); !errors.Is(err, config.ErrBehaviorFaultScope) {
		t.Fatalf("LoadYAMLBytes() = %v, want ErrBehaviorFaultScope", err)
	}
}

func TestBehaviorTimelineRejectsInvalidLinkDown(t *testing.T) {
	for _, value := range []int{0, -1, 101} {
		if _, err := config.LoadYAMLBytes(linkDownTimeline("Gi0/1", value)); err == nil {
			t.Fatalf("accepted invalid link fault value %d", value)
		}
	}
	if _, err := config.LoadYAMLBytes(
		linkDownTimeline("missing", 1),
	); !errors.Is(
		err,
		config.ErrBehaviorTargetNotFound,
	) {
		t.Fatalf("accepted unknown interface: %v", err)
	}
}

func linkDownTimeline(iface string, value int) []byte {
	return fmt.Appendf(nil, `devices:
  - name: switch-1
    type: switch
    mac: "02:00:00:00:00:01"
    interfaces: [{name: Gi0/1}]
behavior_timelines:
  - name: outage
    repeat_count: 1
    phases:
      - name: down
        duration_ms: 1000
        reset: true
        faults: [{device: switch-1, interface: %q, type: link_down, value: %d}]
`, iface, value)
}
