package api

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDraftMaskPayloadRoundTrip(t *testing.T) {
	for _, prefix := range []int{0, 16, 32} {
		server, _ := newTestServer(t)
		library := attachDraftLibrary(t, server)
		draft, err := library.CreateDraft("mask", `devices:
  - name: host
    mac: '02:00:00:00:00:01'
    interfaces: [{name: eth0, address: 192.0.2.1/24}]
`)
		if err != nil {
			t.Fatal(err)
		}
		body := fmt.Sprintf(
			`{"timelines":[{"name":"mask","repeatCount":1,"phases":[{"name":"fault","durationMs":10,"reset":true,"faults":[{"device":"host","interface":"eth0","type":"bad_mask","prefixBits":%d}]}]}]}`,
			prefix,
		)
		recorder := httptest.NewRecorder()
		server.handleLibraryDraftByName(
			recorder,
			draftRequest(
				http.MethodPut,
				"/api/v1/library/drafts/mask/behaviors",
				body,
				draft.Revision,
			),
		)
		if recorder.Code != http.StatusOK {
			t.Fatalf("draft save: %d %s", recorder.Code, recorder.Body.String())
		}
		stored, readErr := library.ReadDraft("mask")
		if readErr != nil {
			t.Fatal(readErr)
		}
		if !strings.Contains(stored.Content, fmt.Sprintf("prefix_bits: %d", prefix)) ||
			strings.Contains(stored.Content, "value:") {
			t.Fatalf("lost mask payload: %s", stored.Content)
		}
	}
}
