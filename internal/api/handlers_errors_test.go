package api

import (
	"errors"
	"slices"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
)

func TestAvailableErrorTypesOnlyAdvertiseObservableFaults(t *testing.T) {
	want := []string{
		"FCS Errors",
		"Packet Discards",
		"Interface Errors",
		"High Utilization",
	}

	types := availableErrorTypes()
	got := make([]string, 0, len(types))
	for _, faultType := range types {
		got = append(got, faultType["type"])
	}

	if !slices.Equal(got, want) {
		t.Fatalf("available fault types = %v, want %v", got, want)
	}
}

func TestParseInterfaceFaultType(t *testing.T) {
	tests := []struct {
		name string
		want devicestate.FaultType
	}{
		{"FCS Errors", devicestate.FaultFCS},
		{"Packet Discards", devicestate.FaultDiscards},
		{"Interface Errors", devicestate.FaultInterface},
		{"High Utilization", devicestate.FaultUtilization},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := parseInterfaceFaultType(test.name)
			if err != nil || got != test.want {
				t.Fatalf("parseInterfaceFaultType(%q) = %q, %v; want %q", test.name, got, err, test.want)
			}
		})
	}

	if _, err := parseInterfaceFaultType("High CPU"); !errors.Is(err, errInterfaceFaultTypeInvalid) {
		t.Fatalf("unsupported type error = %v, want %v", err, errInterfaceFaultTypeInvalid)
	}
}
