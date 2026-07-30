package linklive_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/acceptance/linklive"
)

const testMFAStatus = 498

func TestClientAuthenticatesAndFetchesTopology(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v2/auth/login":
			assertLoginRequest(t, r)
			_ = json.NewEncoder(w).Encode(map[string]string{"accessToken": "secret-token"})
		case "/v1/admin/hosts":
			if got := r.Header.Get("Authorization"); got != "Access secret-token" {
				t.Fatalf("Authorization = %q", got)
			}
			assertTopologyQuery(t, r)
			_, _ = w.Write([]byte(`{"devices":[{"name":"COS-CORE-SW01"}]}`))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)

	client := newTestClient(t, server.URL)
	result, err := client.Topology(context.Background(), "analysis-7")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(result), "COS-CORE-SW01") {
		t.Fatalf("result = %s", result)
	}
}

func TestClientUsesConfiguredAccessTokenWithoutLogin(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v2/auth/login" {
			t.Fatal("client attempted login with a configured access token")
		}
		if got := r.Header.Get("Authorization"); got != "Access cached-token" {
			t.Fatalf("Authorization = %q", got)
		}
		_, _ = w.Write([]byte(`[]`))
	}))
	t.Cleanup(server.Close)

	client, err := linklive.New(linklive.Config{
		IdentityURL: server.URL, APIURL: server.URL,
		AccessToken: "cached-token", AllowInsecure: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = client.Analyses(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestClientLoginIncludesOrganizationClaim(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v2/auth/login" {
			_, _ = w.Write([]byte(`[]`))
			return
		}
		var body map[string]map[string]string
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if got := body["claims"]["org"]; got != "org-7" {
			t.Fatalf("org claim = %q", got)
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"accessToken": "token"})
	}))
	t.Cleanup(server.Close)

	client, err := linklive.New(linklive.Config{
		IdentityURL: server.URL, APIURL: server.URL,
		Username: "tester@example.com", Password: "test-password",
		OrganizationID: "org-7", AllowInsecure: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = client.Analyses(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func assertTopologyQuery(t *testing.T, r *http.Request) {
	t.Helper()
	var query map[string]string
	if err := json.Unmarshal([]byte(r.URL.Query().Get("query")), &query); err != nil {
		t.Fatal(err)
	}
	if query["analysisId"] != "analysis-7" {
		t.Fatalf("analysisId = %q", query["analysisId"])
	}
	if !strings.Contains(r.URL.Query().Get("project"), "connectedHosts") {
		t.Fatal("topology projection omitted connectedHosts")
	}
}

func TestClientReportsMFAWithoutLeakingCredentials(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(testMFAStatus)
		_, _ = w.Write([]byte(`{"message":"MFA required"}`))
	}))
	t.Cleanup(server.Close)

	client := newTestClient(t, server.URL)
	_, err := client.Analyses(context.Background())
	if err == nil || !strings.Contains(err.Error(), "MFA") {
		t.Fatalf("error = %v", err)
	}
	for _, secret := range []string{"tester@example.com", "test-password"} {
		if strings.Contains(err.Error(), secret) {
			t.Fatalf("error leaked credential %q", secret)
		}
	}
}

func TestNewRejectsCleartextCredentials(t *testing.T) {
	_, err := linklive.New(linklive.Config{
		IdentityURL: "http://id.example.test",
		APIURL:      "https://api.example.test",
		Username:    "tester@example.com",
		Password:    "test-password",
	})
	if err == nil || !strings.Contains(err.Error(), "HTTPS") {
		t.Fatalf("error = %v", err)
	}
}

func TestClientDoesNotFollowLoginRedirect(t *testing.T) {
	reachedRedirect := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v2/auth/login" {
			http.Redirect(w, r, "/redirect-target", http.StatusFound)
			return
		}
		reachedRedirect = true
		_, _ = w.Write([]byte(`{"accessToken":"token"}`))
	}))
	t.Cleanup(server.Close)

	client := newTestClient(t, server.URL)
	_, err := client.Analyses(context.Background())
	if err == nil || reachedRedirect {
		t.Fatalf("error = %v, followed redirect = %v", err, reachedRedirect)
	}
}

func assertLoginRequest(t *testing.T, r *http.Request) {
	t.Helper()
	username, password, ok := r.BasicAuth()
	if !ok || username != "tester@example.com" || password != "test-password" {
		t.Fatal("login did not use the configured Basic credentials")
	}
	var body map[string]map[string]string
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body["claims"]["app"] != "3pGv3iSAICObaFX7EjdojO" {
		t.Fatalf("app claim = %q", body["claims"]["app"])
	}
}

func newTestClient(t *testing.T, baseURL string) *linklive.Client {
	t.Helper()
	client, err := linklive.New(linklive.Config{
		IdentityURL:   baseURL,
		APIURL:        baseURL,
		Username:      "tester@example.com",
		Password:      "test-password",
		AllowInsecure: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	return client
}
