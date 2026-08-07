package api

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/library"
	"github.com/MustardSeedNetworks/niac-go/internal/scenario"
	"github.com/MustardSeedNetworks/niac-go/internal/walkprofile"
)

const capturedProfileFixture = `.1.3.6.1.2.1.1.1.0 = STRING: "Cisco IOS XE Software"
.1.3.6.1.2.1.1.2.0 = OID: .1.3.6.1.4.1.9.1.2238
.1.3.6.1.2.1.1.4.0 = STRING: "admin@private.example"
.1.3.6.1.2.1.1.5.0 = STRING: "private-access-01"
.1.3.6.1.2.1.1.6.0 = STRING: "Private data center"
.1.3.6.1.2.1.2.2.1.2.1 = STRING: "GigabitEthernet1/0/1"
.1.3.6.1.2.1.2.2.1.3.1 = INTEGER: 6
`

func walkProfileTestServer(t *testing.T) *Server {
	t.Helper()
	contentLibrary, err := library.Open(t.TempDir())
	if err != nil {
		t.Fatalf("library.Open() error = %v", err)
	}
	return &Server{library: contentLibrary, logger: slog.Default()}
}

func TestWalkImportSanitizesAndReturnsReview(t *testing.T) {
	server := walkProfileTestServer(t)
	body, err := json.Marshal(
		walkImportRequest{Name: "office-switch.walk", Content: capturedProfileFixture},
	)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	recorder := httptest.NewRecorder()
	server.handleWalkImport(
		recorder,
		httptest.NewRequest(http.MethodPost, "/api/v1/walk/import", bytes.NewReader(body)),
	)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", recorder.Code, recorder.Body.String())
	}
	var review walkprofile.Review
	if err = json.NewDecoder(recorder.Body).Decode(&review); err != nil {
		t.Fatalf("decode review: %v", err)
	}
	if review.WalkName != "captured/office-switch.walk" || review.Profile.DeviceType != "switch" ||
		review.Profile.InterfaceCount != 1 {
		t.Fatalf("review = %+v", review)
	}
	saved, err := server.library.ReadFile(library.KindWalks, review.WalkName)
	if err != nil {
		t.Fatalf("read saved walk: %v", err)
	}
	secrets := []string{"private-access-01", "admin@private.example", "Private data center"}
	for _, secret := range secrets {
		if strings.Contains(string(saved), secret) {
			t.Errorf("saved walk contains private identity %q", secret)
		}
	}
}

func TestWalkImportNormalizesSymbolicOIDsBeforeInferenceAndStorage(t *testing.T) {
	server := walkProfileTestServer(t)
	content := `SNMPv2-MIB::sysDescr.0 = STRING: "Cisco IOS XE Switch"
SNMPv2-MIB::sysObjectID.0 = OID: .1.3.6.1.4.1.9.1.2238
SNMPv2-MIB::sysName.0 = STRING: "private-access-01"
IF-MIB::ifDescr.1 = STRING: "GigabitEthernet1/0/1"
IF-MIB::ifType.1 = INTEGER: 6
VENDOR-MIB::privateObject.0 = STRING: "preserved"
`
	body, err := json.Marshal(walkImportRequest{Name: "symbolic.walk", Content: content})
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	recorder := httptest.NewRecorder()
	server.handleWalkImport(
		recorder,
		httptest.NewRequest(http.MethodPost, "/api/v1/walk/import", bytes.NewReader(body)),
	)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", recorder.Code, recorder.Body.String())
	}
	var review walkprofile.Review
	if err = json.NewDecoder(recorder.Body).Decode(&review); err != nil {
		t.Fatalf("decode review: %v", err)
	}
	if review.Profile.DeviceType != "switch" || review.Profile.Vendor != "cisco" ||
		review.Profile.InterfaceCount != 1 {
		t.Fatalf("review = %+v", review)
	}
	saved, err := server.library.ReadFile(library.KindWalks, review.WalkName)
	if err != nil {
		t.Fatalf("read saved walk: %v", err)
	}
	text := string(saved)
	for _, expected := range []string{
		".1.3.6.1.2.1.1.1.0", ".1.3.6.1.2.1.2.2.1.2.1", "VENDOR-MIB::privateObject.0",
	} {
		if !strings.Contains(text, expected) {
			t.Errorf("saved walk missing %q", expected)
		}
	}
}

func TestCapturedProfileRequiresReviewThenJoinsCatalog(t *testing.T) {
	server := walkProfileTestServer(t)
	sanitizedBody, err := json.Marshal(
		walkImportRequest{Name: "reviewed.walk", Content: capturedProfileFixture},
	)
	if err != nil {
		t.Fatalf("marshal import: %v", err)
	}
	server.handleWalkImport(
		httptest.NewRecorder(),
		httptest.NewRequest(http.MethodPost, "/api/v1/walk/import", bytes.NewReader(sanitizedBody)),
	)

	create := createCapturedProfileRequest{
		Role: "office-access", DeviceType: "switch", Vendor: "cisco", Model: "Catalyst 9300-48P",
		Platform: "Cisco IOS XE", Software: "17.15", WalkName: "captured/reviewed.walk",
	}
	body, err := json.Marshal(create)
	if err != nil {
		t.Fatalf("marshal create: %v", err)
	}
	recorder := httptest.NewRecorder()
	server.handleCapturedProfileCreate(
		recorder,
		httptest.NewRequest(
			http.MethodPost,
			"/api/v1/scenario/profiles/captured",
			bytes.NewReader(body),
		),
	)
	if recorder.Code != http.StatusCreated {
		t.Fatalf("status = %d, body=%s", recorder.Code, recorder.Body.String())
	}
	profilesRecorder := httptest.NewRecorder()
	server.handleScenarioProfiles(
		profilesRecorder,
		httptest.NewRequest(http.MethodGet, "/api/v1/scenario/profiles", nil),
	)
	var profiles []scenario.DeviceProfile
	if err = json.NewDecoder(profilesRecorder.Body).Decode(&profiles); err != nil {
		t.Fatalf("decode profiles: %v", err)
	}
	if profiles[len(profiles)-1].Role != "office-access" ||
		profiles[len(profiles)-1].WalkName != "captured/reviewed.walk" {
		t.Fatalf("captured profile missing from catalog: %+v", profiles)
	}
}

func TestWalkImportRejectsInvalidAndResumesExistingReview(t *testing.T) {
	server := walkProfileTestServer(t)
	for index, request := range []walkImportRequest{
		{Name: "../escape.walk", Content: capturedProfileFixture},
		{Name: "empty.walk", Content: "not a walk"},
		{Name: "duplicate.walk", Content: capturedProfileFixture},
		{Name: "duplicate.walk", Content: capturedProfileFixture},
		{Name: "duplicate.walk", Content: strings.Replace(
			capturedProfileFixture, "Cisco IOS XE Software", "Different platform", 1,
		)},
	} {
		body, err := json.Marshal(request)
		if err != nil {
			t.Fatalf("marshal request: %v", err)
		}
		recorder := httptest.NewRecorder()
		server.handleWalkImport(
			recorder,
			httptest.NewRequest(http.MethodPost, "/api/v1/walk/import", bytes.NewReader(body)),
		)
		statuses := []int{
			http.StatusBadRequest, http.StatusUnprocessableEntity, http.StatusOK, http.StatusOK,
			http.StatusConflict,
		}
		want := statuses[index]
		if recorder.Code != want {
			t.Errorf(
				"request %d status = %d, want %d; body=%s",
				index,
				recorder.Code,
				want,
				recorder.Body.String(),
			)
		}
	}
}

func TestWalkProfileRoutesCarryBoundedAuthoringPolicy(t *testing.T) {
	routes := fetchRouteManifest(t)
	importRoute := routes["/api/v1/walk/import"]
	if !importRoute.CSRF || !importRoute.RateLimited ||
		importRoute.MaxBodyBytes != MaxWalkImportBodySize {
		t.Fatalf("walk import policy = %+v", importRoute)
	}
	captureRoute := routes["/api/v1/walk/capture-profile"]
	if !captureRoute.CSRF ||
		!captureRoute.RateLimited ||
		captureRoute.Admin {
		t.Fatalf("walk capture policy = %+v", captureRoute)
	}
}

func TestWalkImportEnvelopeAllowsWorstCaseJSONEscaping(t *testing.T) {
	if MaxWalkImportBodySize < MaxWalkImportSize*6 {
		t.Fatalf("body cap %d cannot hold worst-case escaped walk", MaxWalkImportBodySize)
	}
}
