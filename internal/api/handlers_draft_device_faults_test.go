package api

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
)

func TestDraftDeviceFaultRoundTrip(t *testing.T) {
	for _, test := range []struct {
		faultType string
		value     int
		valid     bool
	}{
		{"dhcp_no_offer", 100, true},
		{"dhcp_no_offer", 101, false},
		{"dns_nxdomain", 100, true},
		{"dns_nxdomain", 101, false},
		{"dns_timeout", 100, true},
		{"dns_timeout", 101, false},
		{"latency", 60000, true},
		{"latency", 60001, false},
	} {
		t.Run(fmt.Sprintf("%s/%d", test.faultType, test.value), func(t *testing.T) {
			server, _ := newTestServer(t)
			lib := attachDraftLibrary(t, server)
			draft, err := lib.CreateDraft("device-fault", `devices:
  - name: resolver-1
    type: server
    mac: '02:00:00:00:00:10'
    ips: [192.0.2.10]
`)
			if err != nil {
				t.Fatal(err)
			}
			body := fmt.Sprintf(
				`{"timelines":[{"name":"outage","repeatCount":1,"phases":[{"name":"fault","durationMs":1000,"reset":true,"faults":[{"device":"resolver-1","type":%q,"value":%d}]}]}]}`,
				test.faultType,
				test.value,
			)
			rec := httptest.NewRecorder()
			server.handleLibraryDraftByName(rec, draftRequest(
				http.MethodPut, "/api/v1/library/drafts/device-fault/behaviors", body, draft.Revision,
			))
			stored, readErr := lib.ReadDraft("device-fault")
			if readErr != nil {
				t.Fatal(readErr)
			}
			if !test.valid {
				if rec.Code != http.StatusBadRequest || stored.Revision != draft.Revision {
					t.Fatalf(
						"invalid fault changed draft or was accepted: status=%d body=%s",
						rec.Code,
						rec.Body.String(),
					)
				}
				return
			}
			if rec.Code != http.StatusOK {
				t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
			}
			assertSavedDeviceFault(t, stored.Content, test.faultType, test.value)
		})
	}
}

func assertSavedDeviceFault(t *testing.T, content, faultType string, value int) {
	t.Helper()
	cfg, err := config.LoadYAMLBytes([]byte(content))
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.BehaviorTimelines) != 1 || len(cfg.BehaviorTimelines[0].Phases) != 1 {
		t.Fatalf("saved timelines=%+v", cfg.BehaviorTimelines)
	}
	faults := cfg.BehaviorTimelines[0].Phases[0].Faults
	if len(faults) != 1 || faults[0].Device != "resolver-1" || faults[0].Interface != "" ||
		faults[0].Type != faultType || faults[0].Value != value {
		t.Fatalf("saved faults=%+v", faults)
	}
}
