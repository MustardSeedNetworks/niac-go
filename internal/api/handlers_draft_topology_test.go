package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/api/ratelimit"
	"github.com/MustardSeedNetworks/niac-go/internal/config"
)

const topologyDraftYAML = `devices:
  - name: core-1
    type: switch
    mac: "02:00:00:00:00:01"
    ips: ["192.0.2.1"]
    interfaces:
      - name: Ethernet1/1
        type: ethernet
  - name: dist-1
    type: switch
    mac: "02:00:00:00:00:02"
    ips: ["192.0.2.2"]
    interfaces:
      - name: Ethernet1/49
        type: ethernet
`

func TestDraftTopologyMutationPersistsReciprocalLinkWithoutApplyingRuntime(t *testing.T) {
	server, _ := newTestServer(t)
	lib := attachDraftLibrary(t, server)
	draft, err := lib.CreateDraft("topology", topologyDraftYAML)
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}
	applied := false
	server.cfg.ApplyConfig = func(_ *config.Config) error {
		applied = true
		return nil
	}

	body := `{
      "operation":"connect",
      "link":{
        "source":{"device":"core-1","interface":"Ethernet1/1"},
        "target":{"device":"dist-1","interface":"Ethernet1/49"},
        "properties":{"vlans":[200,210],"native_vlan":200}
      }
    }`
	rec := httptest.NewRecorder()
	server.handleLibraryDraftByName(rec, draftRequest(
		http.MethodPatch, "/api/v1/library/drafts/topology/topology", body, draft.Revision,
	))
	if rec.Code != http.StatusOK {
		t.Fatalf("PATCH status = %d, body=%s", rec.Code, rec.Body.String())
	}
	updated := decodeDraftResponse(t, rec)
	if updated.Revision == draft.Revision {
		t.Fatal("topology mutation did not advance draft revision")
	}
	cfg, loadErr := config.LoadYAMLBytes([]byte(updated.Content))
	if loadErr != nil {
		t.Fatalf("mutated draft is invalid: %v", loadErr)
	}
	if len(cfg.Devices[0].TrunkPorts) != 1 || len(cfg.Devices[1].TrunkPorts) != 1 {
		t.Fatalf(
			"trunk counts = %d, %d",
			len(cfg.Devices[0].TrunkPorts),
			len(cfg.Devices[1].TrunkPorts),
		)
	}
	if cfg.Devices[0].TrunkPorts[0].RemoteDevice != "dist-1" ||
		cfg.Devices[1].TrunkPorts[0].RemoteDevice != "core-1" {
		t.Fatalf(
			"link is not reciprocal: %+v %+v",
			cfg.Devices[0].TrunkPorts,
			cfg.Devices[1].TrunkPorts,
		)
	}
	if applied {
		t.Fatal("draft topology mutation applied the running configuration")
	}
}

func TestDraftTopologyMutationRequiresRevisionAndPreservesDraftOnFailure(t *testing.T) {
	server, _ := newTestServer(t)
	lib := attachDraftLibrary(t, server)
	draft, err := lib.CreateDraft("topology", topologyDraftYAML)
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}
	body := `{"operation":"connect","link":{"source":{"device":"core-1","interface":"missing"},"target":{"device":"dist-1","interface":"Ethernet1/49"},"properties":{}}}`

	for _, test := range []struct {
		name     string
		revision string
		want     int
	}{
		{name: "missing revision", want: http.StatusPreconditionRequired},
		{name: "stale revision", revision: "stale", want: http.StatusPreconditionFailed},
		{name: "missing interface", revision: draft.Revision, want: http.StatusNotFound},
	} {
		t.Run(test.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			server.handleLibraryDraftByName(rec, draftRequest(
				http.MethodPatch, "/api/v1/library/drafts/topology/topology", body, test.revision,
			))
			if rec.Code != test.want {
				t.Fatalf("status = %d, want %d; body=%s", rec.Code, test.want, rec.Body.String())
			}
		})
	}

	stored, readErr := lib.ReadDraft("topology")
	if readErr != nil {
		t.Fatalf("ReadDraft: %v", readErr)
	}
	if stored.Revision != draft.Revision || stored.Content != topologyDraftYAML {
		t.Fatal("failed mutation changed the draft")
	}
}

func TestDraftTopologyRouteUsesDraftWritePolicy(t *testing.T) {
	policy := fetchRouteManifest(t)["/api/v1/library/drafts/"]
	if !policy.CSRF || !policy.RateLimited || policy.Admin {
		t.Fatalf("draft topology route policy = %+v, want csrf+rateLimited without admin", policy)
	}

	server, _, rwToken := newTestServerWithAuth(t)
	lib := attachDraftLibrary(t, server)
	server.writeLimiter = ratelimit.NewRateLimiter(WriteRateLimit, WriteBurst)
	draft, err := lib.CreateDraft("topology", topologyDraftYAML)
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}
	mux := http.NewServeMux()
	server.registerAPIRoutes(mux)
	body := `{"operation":"move_device","position":{"device":"core-1","x":10,"y":20}}`
	req := draftRequest(
		http.MethodPatch,
		"/api/v1/library/drafts/topology/topology",
		body,
		draft.Revision,
	)
	req.Header.Set("Authorization", "Bearer "+rwToken)
	req.Header.Set("X-Csrf-Token", testCSRFToken(t, server, rwToken))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("authorized PATCH status = %d, body=%s", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("ETag"); got == "" {
		t.Fatal("topology mutation response did not include an ETag")
	}
}
