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

func TestDraftOneShotActions(t *testing.T) {
	for _, test := range []struct {
		name, actions string
		valid         bool
	}{
		{"reboot", `{"device":"switch","type":"reboot"}`, true},
		{"stp", `{"device":"switch","type":"stp_topology_change"}`, true},
		{"unknown kind", `{"device":"switch","type":"unknown"}`, false},
		{"unknown device", `{"device":"absent","type":"reboot"}`, false},
		{"duplicate", `{"device":"switch","type":"reboot"},{"device":"switch","type":"reboot"}`, false},
		{"value", `{"device":"switch","type":"reboot","value":1}`, false},
		{"interface", `{"device":"switch","type":"reboot","interface":"eth0"}`, false},
		{"reset", `{"device":"switch","type":"reboot","reset":true}`, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			server, _ := newTestServer(t)
			lib := attachDraftLibrary(t, server)
			draft, err := lib.CreateDraft(
				"actions",
				"devices:\n  - name: switch\n    type: switch\n    mac: '02:00:00:00:00:01'\n    snmp_agent: {community: reader}\n    stp: {enabled: true}\n",
			)
			if err != nil {
				t.Fatal(err)
			}
			body := fmt.Sprintf(
				`{"timelines":[{"name":"actions","repeatCount":2,"phases":[{"name":"change","durationMs":1000,"reset":true,"actions":[%s]}]}]}`,
				test.actions,
			)
			rec := httptest.NewRecorder()
			server.handleLibraryDraftByName(
				rec,
				draftRequest(http.MethodPut, "/api/v1/library/drafts/actions/behaviors", body, draft.Revision),
			)
			stored, err := lib.ReadDraft("actions")
			if err != nil {
				t.Fatal(err)
			}
			if !test.valid {
				if rec.Code != http.StatusBadRequest || stored.Revision != draft.Revision ||
					stored.Content != draft.Content {
					t.Fatalf("invalid action accepted/changed draft: %d %s", rec.Code, rec.Body.String())
				}
				return
			}
			if rec.Code != http.StatusOK || stored.Revision == draft.Revision {
				t.Fatalf("save failed: %d %s", rec.Code, rec.Body.String())
			}
			kind := map[string]devicestate.DeviceActionType{
				"reboot": devicestate.ActionReboot, "stp": devicestate.ActionSTPTopologyChange,
			}[test.name]
			assertDraftOneShot(t, stored.Content, kind)
		})
	}
}

func assertDraftOneShot(t *testing.T, content string, kind devicestate.DeviceActionType) {
	t.Helper()
	cfg, err := config.LoadYAMLBytes([]byte(content))
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.BehaviorTimelines) != 1 || len(cfg.BehaviorTimelines[0].Phases) != 1 {
		t.Fatal("lost timeline/phase")
	}
	timeline := cfg.BehaviorTimelines[0]
	phase := timeline.Phases[0]
	want := []config.BehaviorAction{{Device: "switch", Type: kind}}
	if timeline.RepeatCount != 2 || !phase.Reset || !reflect.DeepEqual(phase.Actions, want) ||
		len(phase.Faults)+len(phase.Traffic) != 0 {
		t.Fatalf("saved timeline=%+v", timeline)
	}
}
