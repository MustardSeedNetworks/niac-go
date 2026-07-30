package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/scenario"
)

func TestScenarioGenerationErrorStatusSeparatesInputFromInternalFailures(t *testing.T) {
	invalidRequest := fmt.Errorf("bad request: %w", scenario.ErrInvalidRequest)
	if got := scenarioGenerationErrorStatus(invalidRequest); got != http.StatusBadRequest {
		t.Fatalf("invalid request status = %d, want 400", got)
	}
	got := scenarioGenerationErrorStatus(errors.New("marshal failed"))
	if got != http.StatusInternalServerError {
		t.Fatalf("internal failure status = %d, want 500", got)
	}
}

func TestScenarioGenerationHandlersReturnValidatedFleet(t *testing.T) {
	server := &Server{}
	packRecorder := httptest.NewRecorder()
	server.handleScenarioPacks(
		packRecorder,
		httptest.NewRequest(http.MethodGet, "/api/v1/scenario/packs", nil),
	)
	if packRecorder.Code != http.StatusOK {
		t.Fatalf("packs status = %d, want 200", packRecorder.Code)
	}
	var packs []scenario.Pack
	if err := json.NewDecoder(packRecorder.Body).Decode(&packs); err != nil {
		t.Fatalf("decode packs: %v", err)
	}
	if len(packs) != 6 || packs[0].Manifest.DeviceCount == 0 {
		t.Fatalf("packs = %+v, want six versioned manifests", packs)
	}

	profilesRecorder := httptest.NewRecorder()
	server.handleScenarioProfiles(
		profilesRecorder,
		httptest.NewRequest(http.MethodGet, "/api/v1/scenario/profiles", nil),
	)
	if profilesRecorder.Code != http.StatusOK {
		t.Fatalf("profiles status = %d, want 200", profilesRecorder.Code)
	}
	var profiles []scenario.DeviceProfile
	if err := json.NewDecoder(profilesRecorder.Body).Decode(&profiles); err != nil {
		t.Fatalf("decode profiles: %v", err)
	}
	if len(profiles) != len(scenario.Profiles()) {
		t.Fatalf("profiles = %d, want %d", len(profiles), len(scenario.Profiles()))
	}

	body, err := json.Marshal(scenario.EnterpriseReferenceRequest())
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	generateRecorder := httptest.NewRecorder()
	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/scenario/generate",
		bytes.NewReader(body),
	)
	server.handleScenarioGenerate(generateRecorder, request)
	if generateRecorder.Code != http.StatusOK {
		t.Fatalf(
			"generate status = %d, want 200; body=%s",
			generateRecorder.Code,
			generateRecorder.Body.String(),
		)
	}
	var response scenarioGenerateResponse
	if err = json.NewDecoder(generateRecorder.Body).Decode(&response); err != nil {
		t.Fatalf("decode generated scenario: %v", err)
	}
	if response.Manifest.DeviceCount != 531 {
		t.Fatalf("device count = %d, want 531", response.Manifest.DeviceCount)
	}
	if _, err = config.LoadYAMLBytes([]byte(response.Content)); err != nil {
		t.Fatalf("generated content does not load: %v", err)
	}
}

func TestScenarioGenerateRejectsUnknownRequestFields(t *testing.T) {
	server := &Server{}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/scenario/generate",
		bytes.NewBufferString(`{"sites":[],"unknown":true}`),
	)
	server.handleScenarioGenerate(recorder, request)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestScenarioRoutesCarryTemplateAuthoringPolicy(t *testing.T) {
	routes := fetchRouteManifest(t)
	packs := routes["/api/v1/scenario/packs"]
	if packs.Feature != "config_templates" || packs.CSRF || packs.RateLimited {
		t.Fatalf("packs policy = %+v, want config_templates read", packs)
	}
	profiles := routes["/api/v1/scenario/profiles"]
	if profiles.Feature != "config_templates" || profiles.CSRF || profiles.RateLimited {
		t.Fatalf("profiles policy = %+v, want config_templates read", profiles)
	}
	captured := routes["/api/v1/scenario/profiles/captured"]
	if captured.Feature != "config_templates" || !captured.CSRF || !captured.RateLimited ||
		captured.Admin {
		t.Fatalf("captured profile policy = %+v, want config_templates+csrf+rateLimited", captured)
	}
	generate := routes["/api/v1/scenario/generate"]
	if generate.Feature != "config_templates" || !generate.CSRF || !generate.RateLimited ||
		generate.Admin {
		t.Fatalf("generate policy = %+v, want config_templates+csrf+rateLimited", generate)
	}
}
