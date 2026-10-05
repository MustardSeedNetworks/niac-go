package api

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
)

const draftRoamStation = "02:c0:17:a4:03:6b"

// draftRoamScenario is two APs on clinic-corp with the station authored on
// AP-01 (the only one serving SNMP), and AP-03 serving only clinic-guest.
const draftRoamScenario = `devices:
  - name: AP-01
    type: ap
    mac: "00:0c:ce:88:23:c0"
    ips: [10.20.220.11]
    snmp_agent: {community: public}
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
            - {mac: "02:c0:17:a4:03:6b", ip_address: 10.20.220.51, associated_seconds: 600, signal_dbm: -55, signal_quality_pct: 83}
  - name: AP-02
    type: ap
    mac: "00:0c:ce:88:24:c0"
    ips: [10.20.220.12]
    interfaces:
      - {name: Dot11Radio0, type: ieee80211}
    wifi:
      radios:
        - {interface: Dot11Radio0, ssid: clinic-corp, bssid: "00:0c:ce:88:24:c7", band: 5GHz, channel: 149, tx_power_dbm: 20}
  - name: AP-03
    type: ap
    mac: "00:0c:ce:88:25:c0"
    ips: [10.20.220.13]
    interfaces:
      - {name: Dot11Radio0, type: ieee80211}
    wifi:
      radios:
        - {interface: Dot11Radio0, ssid: clinic-guest, bssid: "00:0c:ce:88:25:c7", band: 2.4GHz, channel: 1, tx_power_dbm: 17}
`

// TestDraftBehaviorRoams pins that the composer's replace request carries
// roams: it rebuilds every phase from the request, so a field the request
// cannot express is dropped from the draft on every save (#2534).
func TestDraftBehaviorRoams(t *testing.T) {
	for _, test := range []struct {
		name, roam string
		want       *config.BehaviorRoam
	}{
		{
			"plain", `{"station":"` + draftRoamStation + `","from":"AP-01","to":"AP-02"}`,
			&config.BehaviorRoam{Station: draftRoamStation, From: "AP-01", To: "AP-02"},
		},
		{
			"radio down", `{"station":"` + draftRoamStation + `","from":"AP-01","to":"AP-02","cause":"radio_down"}`,
			&config.BehaviorRoam{
				Station: draftRoamStation, From: "AP-01", To: "AP-02", Cause: devicestate.RoamCauseRadioDown,
			},
		},
		{
			"tx power drop",
			`{"station":"` + draftRoamStation + `","from":"AP-01","to":"AP-02","cause":"tx_power_drop","txPowerDbm":8}`,
			&config.BehaviorRoam{
				Station: draftRoamStation, From: "AP-01", To: "AP-02",
				Cause: devicestate.RoamCauseTxPowerDrop, TxPowerDBM: 8,
			},
		},
		{"ssid not served", `{"station":"` + draftRoamStation + `","from":"AP-01","to":"AP-03"}`, nil},
		{"power without drop", `{"station":"` + draftRoamStation + `","from":"AP-01","to":"AP-02","txPowerDbm":8}`, nil},
		{"power not a drop", `{"station":"` + draftRoamStation + `","from":"AP-01","to":"AP-02","cause":"tx_power_drop","txPowerDbm":17}`, nil},
		{"unknown field", `{"station":"` + draftRoamStation + `","from":"AP-01","to":"AP-02","ssid":"clinic-corp"}`, nil},
	} {
		t.Run(test.name, func(t *testing.T) {
			code, body, stored := replaceDraftRoam(t, test.roam)
			if test.want == nil {
				if code != http.StatusBadRequest || stored != nil {
					t.Fatalf("invalid roam accepted or draft changed: %d %s", code, body)
				}
				return
			}
			if code != http.StatusOK || stored == nil {
				t.Fatalf("save failed: %d %s", code, body)
			}
			got := stored.BehaviorTimelines[0].Phases[0].Roams
			if !reflect.DeepEqual(got, []config.BehaviorRoam{*test.want}) {
				t.Fatalf("saved roams = %+v, want %+v", got, *test.want)
			}
		})
	}
}

// replaceDraftRoam saves one phase holding roam over draftRoamScenario. It
// returns the stored config, or nil when the draft kept its first revision.
func replaceDraftRoam(t *testing.T, roam string) (int, string, *config.Config) {
	t.Helper()
	server, _ := newTestServer(t)
	lib := attachDraftLibrary(t, server)
	draft, err := lib.CreateDraft("roams", draftRoamScenario)
	if err != nil {
		t.Fatal(err)
	}
	body := fmt.Sprintf(
		`{"timelines":[{"name":"roaming","repeatCount":1,"phases":[{"name":"move","durationMs":1000,"reset":true,"roams":[%s]}]}]}`,
		roam,
	)
	rec := httptest.NewRecorder()
	server.handleLibraryDraftByName(
		rec, draftRequest(http.MethodPut, "/api/v1/library/drafts/roams/behaviors", body, draft.Revision),
	)
	stored, err := lib.ReadDraft("roams")
	if err != nil {
		t.Fatal(err)
	}
	if stored.Revision == draft.Revision {
		return rec.Code, rec.Body.String(), nil
	}
	cfg, err := config.LoadYAMLBytes([]byte(stored.Content))
	if err != nil {
		t.Fatal(err)
	}
	return rec.Code, rec.Body.String(), cfg
}
