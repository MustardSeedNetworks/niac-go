package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/library"
	"github.com/MustardSeedNetworks/niac-go/internal/scenario"
)

func libraryRequest(
	t *testing.T, server *Server, mux http.Handler, token, method, path string, body []byte, withCSRF bool,
) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	if withCSRF {
		req.Header.Set("X-Csrf-Token", testCSRFToken(t, server, token))
	}
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	return rec
}

func decodeEntries(t *testing.T, rec *httptest.ResponseRecorder) map[string]library.NetworkEntry {
	t.Helper()
	var entries []library.NetworkEntry
	if err := json.Unmarshal(rec.Body.Bytes(), &entries); err != nil {
		t.Fatalf("decode list: %v\n%s", err, rec.Body.String())
	}
	byName := make(map[string]library.NetworkEntry, len(entries))
	for _, entry := range entries {
		byName[entry.Name] = entry
	}

	return byName
}

func TestLibraryNetworksListsShippedScenariosAsExamples(t *testing.T) {
	server, mux, token, _ := newPackLibraryMux(t)

	rec := libraryRequest(t, server, mux, token, http.MethodGet, "/api/v1/library/networks", nil, false)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d: %s", rec.Code, rec.Body.String())
	}
	byName := decodeEntries(t, rec)
	want := []string{"basic-network", "captured-switch", "captured-pair", "resource-pressure"}
	for _, pack := range scenario.Packs() {
		want = append(want, pack.ID)
	}
	for _, name := range want {
		if byName[name].Source != library.SourceExample {
			t.Errorf("%s: source = %q, want example (row %+v)", name, byName[name].Source, byName[name])
		}
	}
	if _, ok := byName["small-office"]; ok {
		t.Error("a removed built-in is still listed")
	}
}

func TestLibraryExamplesAreReadable(t *testing.T) {
	server, mux, token, _ := newPackLibraryMux(t)

	rec := libraryRequest(t, server, mux, token, http.MethodGet, "/api/v1/library/networks/basic-network", nil, false)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d: %s", rec.Code, rec.Body.String())
	}
	var doc library.NetworkContent
	if err := json.Unmarshal(rec.Body.Bytes(), &doc); err != nil {
		t.Fatal(err)
	}
	if doc.Source != library.SourceExample || doc.Content == "" {
		t.Errorf("want an example with content, got source=%q (%d bytes)", doc.Source, len(doc.Content))
	}
}

func TestLibraryExamplesAreReadOnly(t *testing.T) {
	server, mux, token, root := newPackLibraryMux(t)

	del := libraryRequest(t, server, mux, token, http.MethodDelete,
		"/api/v1/library/networks/basic-network", nil, true)
	if del.Code != http.StatusConflict || !bytes.Contains(del.Body.Bytes(), []byte("read_only_example")) {
		t.Errorf("DELETE example = %d %s, want 409 read_only_example", del.Code, del.Body.String())
	}

	body, err := json.Marshal(map[string]string{
		"name": "basic-network",
		"content": "devices:\n  - name: r1\n    type: router\n" +
			"    mac: \"02:00:00:00:00:01\"\n    ips: [\"192.0.2.1\"]\n",
	})
	if err != nil {
		t.Fatal(err)
	}
	over := libraryRequest(t, server, mux, token, http.MethodPost, "/api/v1/library/networks", body, true)
	if over.Code != http.StatusConflict || !bytes.Contains(over.Body.Bytes(), []byte("read_only_example")) {
		t.Errorf("POST over example = %d %s, want 409 read_only_example", over.Code, over.Body.String())
	}
	if _, statErr := os.Stat(filepath.Join(root, "networks", "basic-network.yaml")); !os.IsNotExist(statErr) {
		t.Errorf("a refused write left a file behind (stat err = %v)", statErr)
	}
}

func TestLibraryDuplicateExample(t *testing.T) {
	server, mux, token, root := newPackLibraryMux(t)

	rec := libraryRequest(t, server, mux, token, http.MethodPost, "/api/v1/library/networks",
		[]byte(`{"copyFrom":"basic-network","name":"my-office"}`), true)
	if rec.Code != http.StatusCreated {
		t.Fatalf("duplicate = %d: %s", rec.Code, rec.Body.String())
	}
	want, err := library.ExampleContent("basic-network")
	if err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(root, "networks", "my-office.yaml"))
	if err != nil || !bytes.Equal(got, want) {
		t.Fatalf("saved copy differs from the example (err=%v)", err)
	}

	list := decodeEntries(t, libraryRequest(t, server, mux, token, http.MethodGet,
		"/api/v1/library/networks", nil, false))
	if list["my-office"].Source != library.SourceUser {
		t.Errorf("copy source = %q, want user", list["my-office"].Source)
	}
	if list["basic-network"].Source != library.SourceExample {
		t.Errorf("the example must remain listed, got %q", list["basic-network"].Source)
	}
}

func TestLibraryDuplicateExampleDefaultsNameAndRefusesOverwrite(t *testing.T) {
	server, mux, token, _ := newPackLibraryMux(t)

	first := libraryRequest(t, server, mux, token, http.MethodPost, "/api/v1/library/networks",
		[]byte(`{"copyFrom":"basic-network"}`), true)
	if first.Code != http.StatusCreated || !bytes.Contains(first.Body.Bytes(), []byte("basic-network-copy")) {
		t.Fatalf("first duplicate = %d: %s", first.Code, first.Body.String())
	}
	again := libraryRequest(t, server, mux, token, http.MethodPost, "/api/v1/library/networks",
		[]byte(`{"copyFrom":"basic-network"}`), true)
	if again.Code != http.StatusConflict {
		t.Errorf("second duplicate = %d, want 409: %s", again.Code, again.Body.String())
	}
}

func TestLibraryDuplicateExampleRefusals(t *testing.T) {
	server, mux, token, _ := newPackLibraryMux(t)

	tests := []struct {
		name string
		body string
		want int
	}{
		{"unknown example", `{"copyFrom":"no-such-example"}`, http.StatusNotFound},
		{"bad name", `{"copyFrom":"basic-network","name":"../x"}`, http.StatusBadRequest},
		{"content and copyFrom", `{"copyFrom":"basic-network","content":"devices: []"}`, http.StatusBadRequest},
		{"another example's name", `{"copyFrom":"basic-network","name":"hospital"}`, http.StatusConflict},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := libraryRequest(t, server, mux, token, http.MethodPost, "/api/v1/library/networks",
				[]byte(tt.body), true)
			if rec.Code != tt.want {
				t.Errorf("status = %d, want %d: %s", rec.Code, tt.want, rec.Body.String())
			}
		})
	}
}

// Duplicate rides the networks POST route, so it inherits that route's CSRF
// gate rather than adding an unguarded mutating path.
func TestLibraryDuplicateExampleRequiresCSRF(t *testing.T) {
	server, mux, token, root := newPackLibraryMux(t)

	rec := libraryRequest(t, server, mux, token, http.MethodPost, "/api/v1/library/networks",
		[]byte(`{"copyFrom":"basic-network"}`), false)
	if rec.Code != http.StatusForbidden {
		t.Errorf("status = %d, want 403 without a CSRF token: %s", rec.Code, rec.Body.String())
	}
	if _, err := os.Stat(filepath.Join(root, "networks", "basic-network-copy.yaml")); !os.IsNotExist(err) {
		t.Errorf("a CSRF-less duplicate wrote a file (stat err = %v)", err)
	}
}

func TestLibraryDuplicatePackSavesLargeDocumentServerSide(t *testing.T) {
	if testing.Short() {
		t.Skip("generates a multi-megabyte pack")
	}
	server, mux, token, root := newPackLibraryMux(t)

	rec := libraryRequest(t, server, mux, token, http.MethodPost, "/api/v1/library/networks",
		[]byte(`{"copyFrom":"campus","name":"my-campus"}`), true)
	if rec.Code != http.StatusCreated {
		t.Fatalf("duplicate pack = %d: %s", rec.Code, rec.Body.String())
	}
	info, err := os.Stat(filepath.Join(root, "networks", "my-campus.yaml"))
	if err != nil || info.Size() < 1<<20 {
		t.Errorf("saved pack missing or implausibly small (err=%v)", err)
	}
}
