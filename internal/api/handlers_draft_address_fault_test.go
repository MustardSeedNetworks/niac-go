package api

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
)

func TestDraftAddressFaultPayloadRoundTrip(t *testing.T) {
	for _, test := range []struct {
		payload string
		valid   bool
	}{
		{`"type":"duplicate_dhcp_offer","address":"192.0.2.20"`, true},
		{`"type":"duplicate_dhcp_offer","address":"192.0.2.20","value":0`, false},
		{`"type":"duplicate_dhcp_offer","address":"bad"`, false},
		{`"type":"duplicate_dhcp_offer","address":"192.0.2.30"`, false},
		{`"type":"latency","address":"192.0.2.20","value":5`, false},
		{`"type":"latency"`, false},
	} {
		server, _ := newTestServer(t)
		lib := attachDraftLibrary(t, server)
		draft, err := lib.CreateDraft("address", `devices:
  - name: server
    mac: '02:00:00:00:00:01'
    ips: [192.0.2.1]
    dhcp: {pool_start: 192.0.2.100, pool_end: 192.0.2.110}
  - name: peer
    mac: '02:00:00:00:00:02'
    ips: [192.0.2.20]
`)
		if err != nil {
			t.Fatal(err)
		}
		body := fmt.Sprintf(
			`{"timelines":[{"name":"conflict","repeatCount":1,"phases":[{"name":"offer","durationMs":10,"reset":true,"faults":[{"device":"server",%s}]}]}]}`,
			test.payload,
		)
		rec := httptest.NewRecorder()
		server.handleLibraryDraftByName(
			rec,
			draftRequest(http.MethodPut, "/api/v1/library/drafts/address/behaviors", body, draft.Revision),
		)
		stored, err := lib.ReadDraft("address")
		if err != nil {
			t.Fatal(err)
		}
		if !test.valid {
			if rec.Code != http.StatusBadRequest || stored.Revision != draft.Revision ||
				stored.Content != draft.Content {
				t.Fatalf("invalid payload mutated draft: %s status=%d %s", test.payload, rec.Code, rec.Body.String())
			}
			continue
		}
		if rec.Code != http.StatusOK {
			t.Fatalf("status=%d: %s", rec.Code, rec.Body.String())
		}
		cfg, err := config.LoadYAMLBytes([]byte(stored.Content))
		if err != nil {
			t.Fatal(err)
		}
		fault := cfg.BehaviorTimelines[0].Phases[0].Faults[0]
		if fault.Address.String() != "192.0.2.20" || fault.Value != 0 || strings.Contains(stored.Content, "value:") {
			t.Fatalf("saved payload changed: %s", stored.Content)
		}
	}
}
