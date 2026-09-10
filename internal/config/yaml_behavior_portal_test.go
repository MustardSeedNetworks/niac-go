package config_test

import (
	"reflect"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
)

func TestCaptivePortalTimelineRoundTrip(t *testing.T) {
	cfg, err := config.LoadYAMLBytes(resourceTimeline("captive_portal", "", 1))
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := config.MarshalConfigYAML(cfg)
	if err != nil {
		t.Fatal(err)
	}
	reloaded, err := config.LoadYAMLBytes(encoded)
	if err != nil || !reflect.DeepEqual(cfg.BehaviorTimelines, reloaded.BehaviorTimelines) {
		t.Fatalf("portal timeline changed: %v", err)
	}
}

func TestCaptivePortalTimelineRejectsNonBinaryAndInterfaceValues(t *testing.T) {
	for _, value := range []int{-1, 0, 2, 100} {
		if _, err := config.LoadYAMLBytes(resourceTimeline("captive_portal", "", value)); err == nil {
			t.Fatalf("accepted portal value %d", value)
		}
	}
	if _, err := config.LoadYAMLBytes(resourceTimeline("captive_portal", "eth0", 1)); err == nil {
		t.Fatal("accepted interface-scoped portal")
	}
}
