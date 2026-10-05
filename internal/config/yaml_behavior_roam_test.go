package config_test

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
)

const roamStation = "02:c0:17:a4:03:6b"

// roamScenario is two APs on one SSID, the station authored on the first, and
// a guest-only AP that cannot take it. Only the first serves SNMP. phases is
// the YAML of one timeline's phase list.
func roamScenario(phases string) []byte {
	return fmt.Appendf(nil, `devices:
  - name: MED-AP-01
    type: ap
    mac: "00:0c:ce:88:23:c0"
    ips: [10.20.220.11]
    snmp_agent:
      community: public
    interfaces:
      - {name: Dot11Radio0, type: ieee80211}
    wifi:
      radios:
        - interface: Dot11Radio0
          ssid: clinic-corp
          bssid: "00:0c:ce:88:23:c7"
          band: 2.4GHz
          channel: 6
          tx_power_dbm: 17
          clients:
            - mac: "%s"
              ip_address: 10.20.220.51
              associated_seconds: 600
              signal_dbm: -55
              signal_quality_pct: 83
  - name: MED-AP-02
    type: ap
    mac: "00:0c:ce:88:24:c0"
    ips: [10.20.220.12]
    interfaces:
      - {name: Dot11Radio0, type: ieee80211}
      - {name: Dot11Radio1, type: ieee80211}
    wifi:
      radios:
        - interface: Dot11Radio0
          ssid: clinic-guest
          bssid: "00:0c:ce:88:24:c7"
          band: 2.4GHz
          channel: 11
          tx_power_dbm: 17
        - interface: Dot11Radio1
          ssid: clinic-corp
          bssid: "00:0c:ce:88:24:c8"
          band: 5GHz
          channel: 149
          tx_power_dbm: 20
  - name: MED-AP-03
    type: ap
    mac: "00:0c:ce:88:25:c0"
    ips: [10.20.220.13]
    interfaces:
      - {name: Dot11Radio0, type: ieee80211}
    wifi:
      radios:
        - interface: Dot11Radio0
          ssid: clinic-guest
          bssid: "00:0c:ce:88:25:c7"
          band: 2.4GHz
          channel: 1
          tx_power_dbm: 17
behavior_timelines:
  - name: roaming
    repeat_count: 3
    phases:
%s`, roamStation, phases)
}

func roamPhase(roams string) string {
	return `      - name: on-ap-02
        duration_ms: 15000
        reset: true
        roams: [` + roams + "]\n"
}

func TestRoamTimelineRoundTrip(t *testing.T) {
	cfg, err := config.LoadYAMLBytes(roamScenario(roamPhase(
		"{station: \"" + roamStation + "\", from: MED-AP-01, to: MED-AP-02, cause: radio_down}")))
	if err != nil {
		t.Fatal(err)
	}
	want := []config.BehaviorRoam{{
		Station: roamStation, From: "MED-AP-01", To: "MED-AP-02", Cause: devicestate.RoamCauseRadioDown,
	}}
	if got := cfg.BehaviorTimelines[0].Phases[0].Roams; !reflect.DeepEqual(got, want) {
		t.Fatalf("roams = %#v, want %#v", got, want)
	}
	data, err := config.MarshalConfigYAML(cfg)
	if err != nil {
		t.Fatal(err)
	}
	reloaded, err := config.LoadYAMLBytes(data)
	if err != nil || !reflect.DeepEqual(cfg.BehaviorTimelines, reloaded.BehaviorTimelines) {
		t.Fatalf("round trip: %v", err)
	}
}

// TestRoamTimelineRejectsWhatNoAPPairCanServe covers what is known before the
// session runs. Whether the station is on `from` when the phase starts is left
// to the runtime: it depends on the phases before it.
func TestRoamTimelineRejectsWhatNoAPPairCanServe(t *testing.T) {
	for _, testCase := range []struct {
		name, roams, want string
	}{
		{"unknown station", `{station: "02:c0:17:a4:03:ff", from: MED-AP-01, to: MED-AP-02}`, "not an authored wireless client"},
		{"same AP", `{station: "` + roamStation + `", from: MED-AP-01, to: MED-AP-01}`, "roams from MED-AP-01 to itself"},
		{"unknown AP", `{station: "` + roamStation + `", from: MED-AP-01, to: MED-AP-09}`, "behavior target not found"},
		{"AP without the SSID", `{station: "` + roamStation + `", from: MED-AP-01, to: MED-AP-03}`, `MED-AP-03 has no radio serving "clinic-corp"`},
		{"not a MAC", `{station: "station-1", from: MED-AP-01, to: MED-AP-02}`, "is not a valid MAC address"},
		{"unknown cause", `{station: "` + roamStation + `", from: MED-AP-01, to: MED-AP-02, cause: rain}`, `"rain" is not one of [radio_down]`},
		{
			"cause on an AP without SNMP",
			`{station: "` + roamStation + `", from: MED-AP-02, to: MED-AP-01, cause: radio_down}`,
			"cause radio_down on MED-AP-02: the old access point serves no SNMP",
		},
		{
			"twice in one phase",
			`{station: "` + roamStation + `", from: MED-AP-01, to: MED-AP-02}, {station: "02:C0:17:A4:03:6B", from: MED-AP-02, to: MED-AP-01}`,
			"roams twice in one phase",
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := config.LoadYAMLBytes(roamScenario(roamPhase(testCase.roams)))
			if err == nil || !strings.Contains(err.Error(), testCase.want) {
				t.Fatalf("error = %v, want one containing %q", err, testCase.want)
			}
		})
	}
}

// TestRoamTimelinesConflictOnTheStation: a station is one thing wherever it
// is, so two timelines moving it at the same time conflict even though they
// name different access points.
func TestRoamTimelinesConflictOnTheStation(t *testing.T) {
	data := roamScenario(roamPhase(`{station: "` + roamStation + `", from: MED-AP-01, to: MED-AP-02}`))
	data = append(data, []byte(`  - name: roaming-back
    repeat_count: 1
    phases:
`+roamPhase(`{station: "`+roamStation+`", from: MED-AP-02, to: MED-AP-01}`))...)

	if _, err := config.LoadYAMLBytes(data); !errors.Is(err, config.ErrBehaviorPhaseOverlap) {
		t.Fatalf("error = %v, want ErrBehaviorPhaseOverlap", err)
	}
}
