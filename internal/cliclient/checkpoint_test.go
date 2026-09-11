package cliclient_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/cliclient"
)

// The acceptance harness resets a scenario between assertions, so the client
// has to reach the session-scoped checkpoint routes with the same bearer and
// CSRF discipline every other mutation uses.
func TestCheckpointRoundTripUsesSessionRoutesAndCSRF(t *testing.T) {
	t.Parallel()

	var saved, restored string
	var savedToken, restoredToken string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/csrf-token":
			_, _ = w.Write([]byte(`{"token":"csrf-1"}`))
		case "/api/v1/sessions/clinic/checkpoints":
			body, _ := io.ReadAll(r.Body)
			saved = string(body)
			savedToken = r.Header.Get("X-Csrf-Token")
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"checkpoint":"healthy","devices":3}`))
		case "/api/v1/sessions/clinic/checkpoints/restore":
			body, _ := io.ReadAll(r.Body)
			restored = string(body)
			restoredToken = r.Header.Get("X-Csrf-Token")
			_, _ = w.Write([]byte(`{"checkpoint":"healthy","restored":true}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client, err := cliclient.New(cliclient.Config{BaseURL: server.URL})
	if err != nil {
		t.Fatal(err)
	}
	devices, err := client.SaveCheckpoint(context.Background(), "clinic", "healthy")
	if err != nil {
		t.Fatalf("SaveCheckpoint: %v", err)
	}
	if devices != 3 {
		t.Fatalf("SaveCheckpoint() = %d devices, want 3", devices)
	}
	if err = client.RestoreCheckpoint(context.Background(), "clinic", "healthy"); err != nil {
		t.Fatalf("RestoreCheckpoint: %v", err)
	}

	for name, got := range map[string]string{"save": saved, "restore": restored} {
		if got != `{"name":"healthy"}` {
			t.Errorf("%s body = %s, want {\"name\":\"healthy\"}", name, got)
		}
	}
	for name, got := range map[string]string{"save": savedToken, "restore": restoredToken} {
		if got != "csrf-1" {
			t.Errorf("%s CSRF token = %q, want csrf-1", name, got)
		}
	}
}

// A session ID reaches the URL path, so one carrying a slash would silently
// address a different route.
func TestCheckpointEscapesTheSessionID(t *testing.T) {
	t.Parallel()

	var path string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/csrf-token" {
			_, _ = w.Write([]byte(`{"token":"csrf"}`))
			return
		}
		path = r.URL.EscapedPath()
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"devices":1}`))
	}))
	defer server.Close()

	client, err := cliclient.New(cliclient.Config{BaseURL: server.URL})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = client.SaveCheckpoint(context.Background(), "a/b", "healthy"); err != nil {
		t.Fatal(err)
	}
	if path != "/api/v1/sessions/a%2Fb/checkpoints" {
		t.Fatalf("path = %s, want /api/v1/sessions/a%%2Fb/checkpoints", path)
	}
}

// The harness injects a fault between the checkpoint and the reset; without
// this the mutation half of the sequence would have to be hand-rolled.
func TestSetDeviceFaultPostsTheAuthoredShape(t *testing.T) {
	t.Parallel()

	var body string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/csrf-token" {
			_, _ = w.Write([]byte(`{"token":"csrf"}`))
			return
		}
		raw, _ := io.ReadAll(r.Body)
		body = string(raw)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer server.Close()

	client, err := cliclient.New(cliclient.Config{BaseURL: server.URL})
	if err != nil {
		t.Fatal(err)
	}
	err = client.SetDeviceFault(context.Background(), cliclient.DeviceFaultRequest{
		Device: "edge-1", Type: "latency", Value: 250,
	})
	if err != nil {
		t.Fatal(err)
	}
	if body != `{"device":"edge-1","errorType":"latency","value":250}` {
		t.Fatalf("body = %s", body)
	}
}
