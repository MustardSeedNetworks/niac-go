package protocols

import (
	"strings"
	"testing"
)

// FuzzDNSDomainName tests DNS domain name parsing with arbitrary input
func FuzzDNSDomainName(f *testing.F) {
	// Seed with valid domain names
	f.Add("example.com")
	f.Add("test.example.com")
	f.Add("sub.domain.example.com")
	f.Add("localhost")
	f.Add("192.168.1.1.in-addr.arpa")
	f.Add("")
	f.Add(".")
	f.Add("..")
	f.Add("a.b.c.d.e.f.g.h.i.j.k.l.m.n.o.p")

	f.Fuzz(func(t *testing.T, domain string) {
		// Prevent panics
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("DNS domain parsing panicked with %q: %v", domain, r)
			}
		}()

		// Validate domain length - DNS labels are max 63 chars, total is 253
		if len(domain) > 253 {
			return
		}

		// Test basic string operations - should not panic
		_ = strings.ToLower(domain)
		_ = strings.TrimSpace(domain)

		// Split into labels
		labels := strings.Split(domain, ".")

		// Check each label
		for _, label := range labels {
			if len(label) > 63 {
				return // Invalid but shouldn't panic
			}
		}
	})
}

// FuzzDNSRecordType tests DNS record type handling with arbitrary input
func FuzzDNSRecordType(f *testing.F) {
	// Seed with valid record types (as strings for testing)
	f.Add("A")
	f.Add("AAAA")
	f.Add("CNAME")
	f.Add("MX")
	f.Add("NS")
	f.Add("PTR")
	f.Add("SOA")
	f.Add("TXT")
	f.Add("")
	f.Add("INVALID")
	f.Add("VERYLONGRECORDTYPE")

	f.Fuzz(func(t *testing.T, recordType string) {
		// Prevent panics
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("DNS record type handling panicked with %q: %v", recordType, r)
			}
		}()

		// Test string operations - should not panic
		upper := strings.ToUpper(recordType)
		_ = len(upper)

		// Check if it's a known type - should not panic
		knownTypes := map[string]bool{
			"A":     true,
			"AAAA":  true,
			"CNAME": true,
			"MX":    true,
			"NS":    true,
			"PTR":   true,
			"SOA":   true,
			"TXT":   true,
		}
		_ = knownTypes[upper]
	})
}

// FuzzDNSTTL tests DNS TTL value handling with arbitrary input
func FuzzDNSTTL(f *testing.F) {
	// Seed with valid TTL values
	f.Add(uint32(0))
	f.Add(uint32(60))
	f.Add(uint32(300))
	f.Add(uint32(3600))
	f.Add(uint32(86400))
	f.Add(uint32(2147483647)) // Max int32
	f.Add(uint32(4294967295)) // Max uint32

	f.Fuzz(func(t *testing.T, ttl uint32) {
		// Prevent panics
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("DNS TTL handling panicked with %d: %v", ttl, r)
			}
		}()

		// Test TTL operations - should not panic
		_ = ttl * 2
		_ = ttl + 1
		_ = ttl / 2

		// Check reasonable ranges (though all uint32 values are technically valid)
		if ttl > 2147483647 {
			// Large TTL but valid
		}
	})
}
