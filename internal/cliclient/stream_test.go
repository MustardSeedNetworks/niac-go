package cliclient_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/cliclient"
)

// The daemon frames its stream as "id: N\ndata: {json}\n\n" and opens with a
// connect event that announces the stream rather than carrying a record. A
// reader that took that first frame for a record would print an empty line
// before every session.
func TestStreamSkipsTheConnectFrameAndReadsRecords(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		flusher, _ := w.(http.Flusher)
		fmt.Fprint(w, "event: connected\ndata: {\"stream\":\"logs\"}\n\n")
		fmt.Fprint(w, "id: 1\ndata: {\"message\":\"link up\",\"device\":\"MED-ACC-SW01\"}\n\n")
		fmt.Fprint(w, "id: 2\ndata: {\"message\":\"link down\",\"device\":\"MED-ACC-SW02\"}\n\n")
		flusher.Flush()
	}))
	defer server.Close()

	client, err := cliclient.New(cliclient.Config{BaseURL: server.URL})
	if err != nil {
		t.Fatal(err)
	}
	var seen []string
	err = client.StreamLogs(context.Background(), func(event cliclient.LogEvent) bool {
		seen = append(seen, event.Device+": "+event.Message)

		return true
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(seen) != 2 {
		t.Fatalf("records = %v, want the two log events and not the connect frame", seen)
	}
	if seen[0] != "MED-ACC-SW01: link up" {
		t.Errorf("first record = %q", seen[0])
	}
}

// A caller that has seen enough stops the stream rather than reading to the end
// of a stream that never ends.
func TestStreamStopsWhenTheCallerIsDone(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		for i := range 100 {
			fmt.Fprintf(w, "id: %d\ndata: {\"message\":\"event\"}\n\n", i)
		}
	}))
	defer server.Close()

	client, err := cliclient.New(cliclient.Config{BaseURL: server.URL})
	if err != nil {
		t.Fatal(err)
	}
	count := 0
	err = client.StreamLogs(context.Background(), func(_ cliclient.LogEvent) bool {
		count++

		return count < 3
	})
	if err != nil {
		t.Fatal(err)
	}
	if count != 3 {
		t.Errorf("read %d records after asking to stop at 3", count)
	}
}
