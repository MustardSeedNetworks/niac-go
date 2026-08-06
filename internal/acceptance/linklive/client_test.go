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
	if !strings.Contains(r.URL.Query().Get("project"), "interfaces") {
		t.Fatal("topology projection omitted interfaces")
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
