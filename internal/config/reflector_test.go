package config

import (
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/converter"
)

func TestParseReflectorConfig(t *testing.T) {
	if got := parseReflectorConfig(nil); got != nil {
		t.Errorf("nil YAML reflector = %+v, want nil", got)
	}

	got := parseReflectorConfig(&converter.ReflectorConfig{
		LatencyMs: 25,
		JitterMs:  5,
		DSCP:      true,
	})
	if got == nil {
		t.Fatal("parseReflectorConfig returned nil for a populated config")
	}

	if got.LatencyMs != 25 || got.JitterMs != 5 || !got.DSCP {
		t.Errorf("parsed reflector = %+v, want {LatencyMs:25 JitterMs:5 DSCP:true}", got)
	}
}

// TestConvertYAMLDeviceReflector proves the reflector block wires through the
// full YAML -> domain device conversion.
func TestConvertYAMLDeviceReflector(t *testing.T) {
	yamlDevice := converter.Device{
		Name:      "reflector",
		MAC:       "bb:bb:bb:00:00:02",
		IPs:       []string{"10.20.200.100"},
		Reflector: &converter.ReflectorConfig{LatencyMs: 10, JitterMs: 2},
	}

	device, err := convertYAMLDevice(yamlDevice, "", nil)
	if err != nil {
		t.Fatalf("convertYAMLDevice: %v", err)
	}

	if device.ReflectorConfig == nil {
		t.Fatal("device.ReflectorConfig is nil; reflector block did not convert")
	}

	if device.ReflectorConfig.LatencyMs != 10 || device.ReflectorConfig.JitterMs != 2 {
		t.Errorf("converted reflector = %+v, want {LatencyMs:10 JitterMs:2}", device.ReflectorConfig)
	}
}
