package cliclient_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/cliclient"
)

func TestSelectSimulationSendsCSRFToken(t *testing.T) {
	t.Parallel()

	var selected bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/csrf-token":
			_, _ = w.Write([]byte(`{"token":"csrf-1"}`))
		case "/api/v1/simulation":
			selected = r.Method == http.MethodPut && r.Header.Get("X-Csrf-Token") == "csrf-1"
			_, _ = w.Write([]byte(`{"running":true,"sessionId":"clinic"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client, err := cliclient.New(cliclient.Config{BaseURL: server.URL})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = client.SelectSimulation(context.Background(), "clinic"); err != nil {
		t.Fatal(err)
	}
	if !selected {
		t.Fatal("selection request omitted its method or CSRF token")
	}
}

func TestPreflightSimulationDoesNotTruncateSuccessfulTopology(t *testing.T) {
	t.Parallel()

	largeName := strings.Repeat("n", maxSuccessfulResponseTestSize)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/csrf-token" {
			_, _ = w.Write([]byte(`{"token":"csrf"}`))
			return
		}
		_, _ = w.Write([]byte(`{"safe":true,"topology":{"networks":[{"name":"` + largeName + `"}]}}`))
	}))
	defer server.Close()

	client, err := cliclient.New(cliclient.Config{BaseURL: server.URL})
	if err != nil {
		t.Fatal(err)
	}
	report, err := client.PreflightSimulation(context.Background(), cliclient.SimulationRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Topology.Networks[0].Name) != maxSuccessfulResponseTestSize {
		t.Fatal("successful topology response was truncated")
	}
}

const maxSuccessfulResponseTestSize = 128 << 10

func TestSimulationMutationFailuresRemainActionable(t *testing.T) {
	t.Parallel()

	client, err := cliclient.New(cliclient.Config{BaseURL: "http://127.0.0.1:1"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.SelectSimulation(context.Background(), "clinic")
	if !errors.Is(err, cliclient.ErrDaemonUnreachable) {
		t.Errorf("unreachable error = %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/csrf-token" {
			_, _ = w.Write([]byte(`{"token":"csrf"}`))
			return
		}
		w.WriteHeader(http.StatusForbidden)
	}))
	defer server.Close()
	client, err = cliclient.New(cliclient.Config{BaseURL: server.URL, Token: "read-only"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.SelectSimulation(context.Background(), "clinic")
	if !errors.Is(err, cliclient.ErrForbidden) {
		t.Errorf("forbidden error = %v", err)
	}
}

func TestSelectSimulationRefreshesRejectedCSRFTokenOnce(t *testing.T) {
	t.Parallel()

	tokenFetches := 0
	selectionAttempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/csrf-token" {
			tokenFetches++
			_, _ = w.Write([]byte(`{"token":"csrf-` + string(rune('0'+tokenFetches)) + `"}`))
			return
		}
		selectionAttempts++
		if selectionAttempts == 1 {
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte(`{"error":{"code":"csrf_token_invalid"}}`))
			return
		}
		_, _ = w.Write([]byte(`{"running":true,"sessionId":"clinic"}`))
	}))
	defer server.Close()

	client, err := cliclient.New(cliclient.Config{BaseURL: server.URL})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = client.SelectSimulation(context.Background(), "clinic"); err != nil {
		t.Fatal(err)
	}
	if tokenFetches != 2 || selectionAttempts != 2 {
		t.Fatalf("token fetches = %d, attempts = %d; want 2 and 2", tokenFetches, selectionAttempts)
	}
}

func TestSelectSimulationRefreshesExpiredCSRFTokenOnce(t *testing.T) {
	t.Parallel()

	assertCSRFRefresh(t, "csrf_token_expired")
}

func assertCSRFRefresh(t *testing.T, code string) {
	t.Helper()
	tokenFetches := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/csrf-token" {
			tokenFetches++
			_, _ = w.Write([]byte(`{"token":"csrf"}`))
			return
		}
		if tokenFetches == 1 {
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte(`{"error":{"code":"` + code + `"}}`))
			return
		}
		_, _ = w.Write([]byte(`{"running":true,"sessionId":"clinic"}`))
	}))
	defer server.Close()

	client, err := cliclient.New(cliclient.Config{BaseURL: server.URL})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = client.SelectSimulation(context.Background(), "clinic"); err != nil {
		t.Fatal(err)
	}
	if tokenFetches != 2 {
		t.Fatalf("token fetches = %d, want 2", tokenFetches)
	}
}

func TestStartSimulationReturnsNewSessionWhenAnotherIsSelected(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/csrf-token" {
			_, _ = w.Write([]byte(`{"token":"csrf"}`))
			return
		}
		_, _ = w.Write([]byte(`{"running":true,"sessionId":"selected","sessions":[` +
			`{"running":true,"sessionId":"selected"},{"running":true,"sessionId":"new"}]}`))
	}))
	defer server.Close()

	client, err := cliclient.New(cliclient.Config{BaseURL: server.URL})
	if err != nil {
		t.Fatal(err)
	}
	status, err := client.StartSimulation(context.Background(), cliclient.SimulationRequest{SessionID: "new"})
	if err != nil {
		t.Fatal(err)
	}
	if status.SessionID != "new" {
		t.Fatalf("session = %q, want new", status.SessionID)
	}
}

func TestStartSimulationReturnsDefaultSessionWhenIDIsOmitted(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/csrf-token" {
			_, _ = w.Write([]byte(`{"token":"csrf"}`))
			return
		}
		_, _ = w.Write([]byte(`{"running":true,"sessionId":"selected","sessions":[` +
			`{"running":true,"sessionId":"selected"},{"running":true,"sessionId":"default"}]}`))
	}))
	defer server.Close()

	client, err := cliclient.New(cliclient.Config{BaseURL: server.URL})
	if err != nil {
		t.Fatal(err)
	}
	status, err := client.StartSimulation(context.Background(), cliclient.SimulationRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if status.SessionID != "default" {
		t.Fatalf("session = %q, want default", status.SessionID)
	}
}

func TestSimulationLifecycleUsesDaemonRoutes(t *testing.T) {
	t.Parallel()

	want := []string{
		"POST /api/v1/simulation/preflight",
		"POST /api/v1/simulation",
		"DELETE /api/v1/sessions/clinic",
	}
	got := make([]string, 0, len(want))
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/csrf-token" {
			_, _ = w.Write([]byte(`{"token":"csrf"}`))
			return
		}
		got = append(got, r.Method+" "+r.URL.Path)
		if r.URL.Path == "/api/v1/simulation/preflight" {
			_, _ = w.Write([]byte(`{"safe":true}`))
			return
		}
		if r.Method == http.MethodDelete {
			_, _ = w.Write([]byte(`{"status":"stopped"}`))
			return
		}
		_, _ = w.Write([]byte(`{"running":true,"sessionId":"clinic"}`))
	}))
	defer server.Close()

	client, err := cliclient.New(cliclient.Config{BaseURL: server.URL})
	if err != nil {
		t.Fatal(err)
	}
	request := cliclient.SimulationRequest{SessionID: "clinic", Interface: "eth0", ConfigPath: "clinic.yaml"}
	if _, err = client.PreflightSimulation(context.Background(), request); err != nil {
		t.Fatal(err)
	}
	if _, err = client.StartSimulation(context.Background(), request); err != nil {
		t.Fatal(err)
	}
	if err = client.StopSimulation(context.Background(), "clinic"); err != nil {
		t.Fatal(err)
	}
	for index := range want {
		if got[index] != want[index] {
			t.Errorf("request %d = %q, want %q", index, got[index], want[index])
		}
	}
}
