package protocols

import (
	"net"
	"testing"
)

// TestDNSWildcardCatchAll: a "*" forward record resolves any otherwise-unmatched
// name (so a simulated resolver can stand in for the whole internet), while exact
// records still win and absence of a wildcard still yields no answer (NXDOMAIN).
func TestDNSWildcardCatchAll(t *testing.T) {
	h := &DNSHandler{
		domain: "demo.lab",
		records: map[string][]dnsRecord{
			"gw.demo.lab":   {{ip: net.ParseIP("10.10.200.1")}},
			dnsWildcardName: {{ip: net.ParseIP("8.8.8.8")}},
		},
	}

	// Exact match wins over the wildcard.
	if r := h.lookupHost("gw.demo.lab", nil); len(r) != 1 || !r[0].ip.Equal(net.ParseIP("10.10.200.1")) {
		t.Errorf("exact match failed: %v", r)
	}

	// Unmatched internet name falls back to the wildcard.
	for _, name := range []string{"www.google.com", "connectivitycheck.gstatic.com", "anything.example"} {
		r := h.lookupHost(name, nil)
		if len(r) != 1 || !r[0].ip.Equal(net.ParseIP("8.8.8.8")) {
			t.Errorf("wildcard fallback for %q failed: %v", name, r)
		}
	}

	// Without a wildcard, an unmatched name still resolves to nothing.
	noWild := &DNSHandler{
		domain:  "demo.lab",
		records: map[string][]dnsRecord{"host.demo.lab": {{ip: net.ParseIP("10.0.0.9")}}},
	}
	if r := noWild.lookupHost("nope.example", nil); r != nil {
		t.Errorf("expected no answer without a wildcard, got %v", r)
	}
}
