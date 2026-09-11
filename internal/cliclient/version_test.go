package cliclient_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/cliclient"
)

// /__version is the daemon's readiness and deployment signal, and it carries
// no auth. Reading it through the client is what lets a caller assert the
// release it is actually talking to.
func TestVersionReadsTheUnauthenticatedBuildRoute(t *testing.T) {
	t.Parallel()

	var path string
	var sentAuth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		sentAuth = r.Header.Get("Authorization")
		_, _ = w.Write([]byte(`{"version":"0.95.51","commitFull":"abc123","uiBuildHash":"deadbeef"}`))
	}))
	defer server.Close()

	client, err := cliclient.New(cliclient.Config{BaseURL: server.URL, Token: "t"})
	if err != nil {
		t.Fatal(err)
	}
	version, err := client.Version(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if path != "/__version" {
		t.Fatalf("path = %s, want /__version", path)
	}
	if sentAuth == "" {
		t.Fatal("the client dropped its bearer token on an unauthenticated route")
	}
	if version.Version != "0.95.51" || version.UIBuildHash != "deadbeef" {
		t.Fatalf("version = %+v", version)
	}
}
