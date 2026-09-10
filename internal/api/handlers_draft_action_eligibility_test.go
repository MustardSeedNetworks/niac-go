package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDraftActionsRejectDisabledRequiredProtocols(t *testing.T) {
	for _, extra := range []string{"", "    snmp_agent: {community: reader}\n"} {
		t.Run(extra, func(t *testing.T) {
			server, _ := newTestServer(t)
			lib := attachDraftLibrary(t, server)
			draft, err := lib.CreateDraft(
				"disabled",
				"devices:\n  - name: switch\n    type: switch\n    mac: '02:00:00:00:00:01'\n"+extra,
			)
			if err != nil {
				t.Fatal(err)
			}
			rec := httptest.NewRecorder()
			server.handleLibraryDraftByName(
				rec,
				draftRequest(
					http.MethodPut,
					"/api/v1/library/drafts/disabled/behaviors",
					`{"timelines":[{"name":"changes","repeatCount":1,"phases":[{"name":"change","durationMs":1000,"actions":[{"device":"switch","type":"stp_topology_change"}]}]}]}`,
					draft.Revision,
				),
			)
			stored, err := lib.ReadDraft("disabled")
			if err != nil {
				t.Fatal(err)
			}
			if rec.Code != http.StatusBadRequest || stored.Revision != draft.Revision ||
				stored.Content != draft.Content {
				t.Fatalf("unrunnable action accepted: %d %s", rec.Code, rec.Body.String())
			}
		})
	}
}
