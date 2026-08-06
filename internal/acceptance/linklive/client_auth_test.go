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

func TestClientRefreshesConfiguredAccessTokenOnHTTP426(t *testing.T) {
	const cachedToken = "header.eyJvcmciOiJvcmctNyJ9.signature"
	apiCalls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v2/auth/refresh":
			if got := r.Header.Get("Authorization"); got != "Access "+cachedToken {
				t.Fatalf("refresh Authorization = %q", got)
			}
			var body map[string]map[string]string
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatal(err)
			}
			if got := body["claims"]["org"]; got != "org-7" {
				t.Fatalf("refresh org claim = %q", got)
			}
			_ = json.NewEncoder(w).Encode(map[string]string{"accessToken": "renewed-token"})
		case "/v1/admin/analysis":
			apiCalls++
			if apiCalls == 1 {
				w.WriteHeader(http.StatusUpgradeRequired)
				return
			}
			if got := r.Header.Get("Authorization"); got != "Access renewed-token" {
				t.Fatalf("retry Authorization = %q", got)
			}
			_, _ = w.Write([]byte(`[]`))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)

	client, err := linklive.New(linklive.Config{
		IdentityURL: server.URL, APIURL: server.URL,
		AccessToken: cachedToken, AllowInsecure: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = client.Analyses(context.Background()); err != nil {
		t.Fatal(err)
	}
	if apiCalls != 2 {
		t.Fatalf("API calls = %d, want 2", apiCalls)
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
