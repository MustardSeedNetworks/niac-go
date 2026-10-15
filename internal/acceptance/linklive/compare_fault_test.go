package linklive_test

import (
	"slices"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/acceptance/linklive"
	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
	"github.com/MustardSeedNetworks/niac-go/internal/scenario"
)

// R-A2: on the 2026-10-14 captures every interface a pack starts with a fault
// on came back as a finding - the hospital's imaging uplinks and the closet
// uplinks that carry errors or discards warned, and the warehouse dock port
// with no PoE read Down - because the comparator read faults from behavior
// timelines only, never the ones authored on the interface itself.

func TestFromConfigCarriesEveryPackInterfaceFault(t *testing.T) {
	checked := 0
	for _, pack := range scenario.Packs() {
		t.Run(pack.ID, func(t *testing.T) {
			result, err := scenario.Generate(pack.Request)
			if err != nil {
				t.Fatal(err)
			}
			authored := authoredInterfacesByName(linklive.FromConfig(result.Config))
			for _, faulted := range interfaceFaults(result.Config.Devices) {
				got, found := authored[faulted.port]
				if !found {
					t.Fatalf("%s is not in the authored model", faulted.port)
				}
				if problem := faultExpectationProblem(got, faulted.fault); problem != "" {
					t.Errorf("%s %s: %s", faulted.port, faulted.fault.Type, problem)
				}
				checked++
			}
		})
	}
	if checked == 0 {
		t.Fatal("no pack authors an interface fault, so this test checks nothing")
	}
}

type faultedInterface struct {
	port  string
	fault config.InterfaceFault
}

func interfaceFaults(devices []config.Device) []faultedInterface {
	var result []faultedInterface
	for _, device := range devices {
		for _, iface := range device.Interfaces {
			for _, fault := range iface.Faults {
				result = append(result, faultedInterface{device.Name + " " + iface.Name, fault})
			}
		}
	}
	return result
}

func faultExpectationProblem(got linklive.AuthoredInterface, fault config.InterfaceFault) string {
	switch devicestate.FaultType(fault.Type) {
	case devicestate.FaultUtilization:
		if got.UtilizationPercent != float64(fault.Value) {
			return "utilization not the authored value"
		}
	case devicestate.FaultLinkDown, devicestate.FaultPoELoss:
		if got.Status != "Down" {
			return "status " + got.Status + ", want Down"
		}
	case devicestate.FaultFCS, devicestate.FaultInterface:
		if got.ExpectZeroErrors || !got.CounterFault {
			return "errors still expected zero"
		}
	case devicestate.FaultDiscards:
		if got.ExpectZeroDiscards || !got.CounterFault {
			return "discards still expected zero"
		}
	default:
		return "fault type the comparator does not model"
	}
	return ""
}

func TestCompareAcceptsWarningOnCounterFault(t *testing.T) {
	cases := []struct {
		name         string
		counterFault bool
		want         bool
	}{
		{"interface authored with a counter fault", true, false},
		{"clean interface", false, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			authored := authoredPair(t)
			authored.Devices[0].Interfaces[0].CounterFault = tc.counterFault
			authored.Devices[0].Interfaces[0].ExpectZeroErrors = !tc.counterFault
			observed := observedPair("Switch", "Full", "100 Gb")
			observed.Hosts[0].WorstProblem = "Warning(s)"
			observed.Hosts[0].Interfaces[0].WorstProblem = "Warning(s)"
			observed.Hosts[0].Interfaces[0].Interface.Errors.Percent = 0.01

			warnings := []linklive.FindingKind{
				linklive.FindingProblemConflict,
				linklive.FindingInterfaceProblemConflict,
				linklive.FindingInterfaceErrorConflict,
			}
			var kinds []linklive.FindingKind
			for _, finding := range linklive.Compare(authored, observed) {
				if slices.Contains(warnings, finding.Kind) {
					kinds = append(kinds, finding.Kind)
				}
			}
			if got := len(kinds) > 0; got != tc.want {
				t.Fatalf("warning findings = %v, want any = %v", kinds, tc.want)
			}
		})
	}
}

func authoredInterfacesByName(snapshot linklive.AuthoredSnapshot) map[string]linklive.AuthoredInterface {
	result := make(map[string]linklive.AuthoredInterface)
	for _, device := range snapshot.Devices {
		for _, iface := range device.Interfaces {
			result[device.Name+" "+iface.Name] = iface
		}
	}
	return result
}
