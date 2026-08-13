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

// The counters the monitor prints come from the daemon's own stats endpoint,
// which has always carried them. The CLI reported zeros because it read a
// transport that carried no protocol breakdown at all.
func TestStatsCarriesThePerProtocolCounters(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/stats" {
			t.Errorf("path = %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"interface":"eth0","deviceCount":75,"stack":{
			"packetsSent":6902,"packetsReceived":25758,
			"arpRequests":5,"arpReplies":3,"icmpRequests":2,"icmpReplies":1,
			"dnsQueries":7,"dhcpRequests":4,"snmpQueries":11,"errors":0}}`))
	}))
	defer server.Close()

	stats := fetch(t, server.URL)
	if stats.Stack.SNMPQueries != 11 || stats.Stack.DNSQueries != 7 {
		t.Errorf("stack = %+v, want the served counters", stats.Stack)
	}
	// One column each for ARP and ICMP, so request and reply are summed.
	if stats.Stack.ARP() != 8 {
		t.Errorf("ARP() = %d, want 8", stats.Stack.ARP())
	}
	if stats.Stack.ICMP() != 3 {
		t.Errorf("ICMP() = %d, want 3", stats.Stack.ICMP())
	}
}

// The old failure told an operator "no NIAC simulation running" while several
// were running. Say what was actually tried instead.
func TestUnreachableDaemonNamesTheAddress(t *testing.T) {
	client, err := cliclient.New(cliclient.Config{BaseURL: "http://127.0.0.1:1", Insecure: true})
	if err != nil {
		t.Fatal(err)
	}

	_, err = client.Stats(context.Background())
	if !errors.Is(err, cliclient.ErrDaemonUnreachable) {
		t.Fatalf("error = %v, want ErrDaemonUnreachable", err)
	}
	if !strings.Contains(err.Error(), "127.0.0.1:1") {
		t.Errorf("error = %v, want it to name the address tried", err)
	}
}

// A daemon on a non-loopback address wants a token, and the operator needs to
// be told which one.
func TestTokenRefusalSaysWhatToSet(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	client, err := cliclient.New(cliclient.Config{BaseURL: server.URL})
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.Stats(context.Background())
	if !errors.Is(err, cliclient.ErrUnauthorized) {
		t.Fatalf("error = %v, want ErrUnauthorized", err)
	}
	if !strings.Contains(err.Error(), "NIAC_API_TOKEN") {
		t.Errorf("error = %v, want it to name the variable to set", err)
	}
}

// A token, when set, has to actually reach the daemon.
func TestTokenIsSent(t *testing.T) {
	var seen string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = r.Header.Get("Authorization")
		_, _ = w.Write([]byte(`{"interface":"eth0"}`))
	}))
	defer server.Close()

	client, err := cliclient.New(cliclient.Config{BaseURL: server.URL, Token: "s3cret"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = client.Stats(context.Background()); err != nil {
		t.Fatal(err)
	}
	if seen != "Bearer s3cret" {
		t.Errorf("Authorization = %q", seen)
	}
}

func fetch(t *testing.T, baseURL string) *cliclient.Stats {
	t.Helper()
	client, err := cliclient.New(cliclient.Config{BaseURL: baseURL})
	if err != nil {
		t.Fatal(err)
	}
	stats, err := client.Stats(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	return stats
}
