package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/api/ratelimit"
	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/drafttopology"
	"github.com/MustardSeedNetworks/niac-go/internal/library"
	"github.com/MustardSeedNetworks/niac-go/internal/scenario"
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

func TestCapturedProfileEnrichmentAttachesReviewedWalk(t *testing.T) {
	server, _ := newTestServer(t)
	contentLibrary := attachDraftLibrary(t, server)
	writeErr := contentLibrary.WriteFile(
		library.KindWalks, "captured/access.walk", []byte(capturedProfileFixture),
	)
	if writeErr != nil {
		t.Fatalf("WriteFile() error = %v", writeErr)
	}
	profile := scenario.DeviceProfile{
		Role: "office-access", DeviceType: "switch", Vendor: "cisco", Model: "Catalyst 9300",
		Platform: "Cisco IOS XE", SysObjectID: "1.3.6.1.4.1.9.1.2238",
		WalkName: "captured/access.walk", InterfaceCount: 1,
		Interfaces: []scenario.ProfileInterface{
			{Name: "GigabitEthernet1/0/1", Type: "ethernet", MTU: 1500, Speed: 100_000_000_000},
		},
		Source: "captured",
	}
	if err := scenario.SaveCustomProfile(contentLibrary.Root(), profile); err != nil {
		t.Fatalf("SaveCustomProfile() error = %v", err)
	}
	cfg := &config.Config{}
	mutation := drafttopology.Mutation{
		Operation: drafttopology.AddDevice,
		Device: &drafttopology.DeviceMutation{
			Name: "access-1", Type: "host", Vendor: "generic", MACSuffix: 1,
			ProfileRole: "office-access",
			Interfaces:  []drafttopology.Interface{{Name: "GigabitEthernet1/0/1", Type: "ethernet"}},
		},
	}
	if err := server.enrichCapturedDraftProfile(cfg, &mutation); err != nil {
		t.Fatalf("enrichCapturedDraftProfile() error = %v", err)
	}
	if mutation.Device.Type != "switch" || mutation.Device.WalkFile != "captured/access.walk" ||
		cfg.IncludePath != contentLibrary.SubDir(library.KindWalks) {
		t.Fatalf("enriched mutation = %+v, includePath=%q", mutation.Device, cfg.IncludePath)
	}
	if len(mutation.Device.Interfaces) != 1 ||
		mutation.Device.Interfaces[0].Name != "GigabitEthernet1/0/1" ||
		mutation.Device.Interfaces[0].Speed != 100_000 ||
		mutation.Device.Interfaces[0].MTU != 1500 {
		t.Fatalf("captured interfaces = %+v", mutation.Device.Interfaces)
	}
	if err := drafttopology.Apply(cfg, mutation); err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	if cfg.Devices[0].SNMPConfig.WalkFile != "captured/access.walk" ||
		cfg.Devices[0].SNMPConfig.Community != "public" {
		t.Fatalf("SNMP config = %+v", cfg.Devices[0].SNMPConfig)
	}
}

func TestCapturedProfileEnrichmentRejectsExternalCommunityWalk(t *testing.T) {
	server, _ := newTestServer(t)
	contentLibrary := attachDraftLibrary(t, server)
	writeErr := contentLibrary.WriteFile(
		library.KindWalks, "captured/access.walk", []byte(capturedProfileFixture),
	)
	if writeErr != nil {
		t.Fatalf("WriteFile() error = %v", writeErr)
	}
	profile := scenario.DeviceProfile{
		Role: "office-access", DeviceType: "switch", Vendor: "cisco", Model: "Catalyst 9300",
		Platform: "Cisco IOS XE", WalkName: "captured/access.walk", Source: "captured",
	}
	if err := scenario.SaveCustomProfile(contentLibrary.Root(), profile); err != nil {
		t.Fatalf("SaveCustomProfile() error = %v", err)
	}
	cfg := &config.Config{IncludePath: t.TempDir(), Devices: []config.Device{{
		SNMPConfig: config.SNMPConfig{
			CommunityIncludes: []config.CommunityInclude{{WalkFile: "external.walk"}},
		},
	}}}
	mutation := drafttopology.Mutation{
		Operation: drafttopology.AddDevice,
		Device: &drafttopology.DeviceMutation{
			Name: "access-1", ProfileRole: "office-access",
		},
	}
	enrichErr := server.enrichCapturedDraftProfile(cfg, &mutation)
	if enrichErr == nil || !strings.Contains(enrichErr.Error(), "another content location") {
		t.Fatalf("enrichCapturedDraftProfile() error = %v", enrichErr)
	}
}

func TestCapturedProfileInterfacePreservesSupportedStates(t *testing.T) {
	converted := capturedProfileInterface(scenario.ProfileInterface{
		Name: "Ethernet1", Type: "ethernet", MTU: 65536, Speed: 1_000_000_000,
		AdminStatus: "testing", OperStatus: "lowerLayerDown",
	})
	if converted.AdminStatus != "testing" || converted.OperStatus != "down" ||
		converted.MTU != 65536 {
		t.Fatalf("capturedProfileInterface() = %+v", converted)
	}
}

func TestApplyCapturedProfileKeepsFallbackInterfacesWithoutInventory(t *testing.T) {
	device := &drafttopology.DeviceMutation{
		Interfaces: []drafttopology.Interface{{Name: "Ethernet1/1", Type: "ethernet"}},
	}
	applyCapturedProfile(device, scenario.DeviceProfile{
		Role: "captured-switch", DeviceType: "switch", Vendor: "generic",
		WalkName: "captured/portless.walk",
	})
	if len(device.Interfaces) != 1 || device.Interfaces[0].Name != "Ethernet1/1" {
		t.Fatalf("fallback interfaces = %+v", device.Interfaces)
	}
}

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
        "properties":{"vlans":[200,210],"nativeVlan":200}
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

func TestDraftTopologyAddsTheFirstDeviceToAnEmptyDraft(t *testing.T) {
	server, _ := newTestServer(t)
	lib := attachDraftLibrary(t, server)
	draft, err := lib.CreateDraft("empty-topology", "devices: []\n")
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}

	body := `{
      "operation":"add_device",
      "device":{
        "name":"first-1","type":"router","vendor":"cisco","macSuffix":123,
        "interfaces":[{"name":"Ethernet1/1","type":"ethernet","mtu":1500,
          "speed":1000,"duplex":"full","adminStatus":"up","operStatus":"up"}]
      }
    }`
	rec := httptest.NewRecorder()
	server.handleLibraryDraftByName(rec, draftRequest(
		http.MethodPatch, "/api/v1/library/drafts/empty-topology/topology", body, draft.Revision,
	))

	// The handler loads the draft's *current* content before applying the
	// mutation, and the loader refuses a device-less config because a runnable
	// one must describe at least one device. Treating that as a broken draft
	// made it impossible to add the first device to an empty draft at all,
	// which is exactly what authoring a network from empty starts with.
	if rec.Code != http.StatusOK {
		t.Fatalf("PATCH status = %d, want %d; body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}
	updated := decodeDraftResponse(t, rec)
	cfg, loadErr := config.LoadYAMLBytes([]byte(updated.Content))
	if loadErr != nil {
		t.Fatalf("mutated draft does not load: %v", loadErr)
	}
	if len(cfg.Devices) != 1 || cfg.Devices[0].Name != "first-1" {
		t.Fatalf("devices = %#v, want one device named first-1", cfg.Devices)
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
