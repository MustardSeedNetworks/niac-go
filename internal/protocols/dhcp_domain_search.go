package protocols

import (
	"fmt"

	"github.com/MustardSeedNetworks/niac-go/internal/safeconv"
)

// DHCP option constraints (RFC 2132, RFC 3397).
const (
	MaxDHCPOptionLen     = 255 // Maximum DHCP option length
	MaxDomainSearchCount = 10  // Reasonable limit for simulation
	MaxDomainLen         = 253 // RFC 1035: maximum domain name length
)

// encodeDomainSearchList encodes a domain search list in DNS label format (RFC 1035)
// Used for DHCP Option 119 (Domain Search)
// Returns error if constraints are violated.
func (h *DHCPHandler) encodeDomainSearchList(domains []string) ([]byte, error) {
	if len(domains) > MaxDomainSearchCount {
		return nil, fmt.Errorf("%w: %d > %d", ErrTooManyDomainSearchEntries, len(domains), MaxDomainSearchCount)
	}

	result := make([]byte, 0, MaxDHCPOptionLen)

	for _, domain := range domains {
		// Validate domain length
		if len(domain) > MaxDomainLen {
			return nil, fmt.Errorf("%w: %d > %d (domain: %s)", ErrDomainTooLong, len(domain), MaxDomainLen, domain)
		}

		// Split domain into labels (e.g., "example.com" -> ["example", "com"])
		labels := make([]byte, 0, len(domain)+domainLabelBufferPad)

		for _, label := range splitDomain(domain) {
			if len(label) == 0 || len(label) > maxDomainLabelLen {
				continue // Invalid label
			}
			// Add label length byte followed by label bytes
			labels = append(labels, safeconv.Byte(len(label)))
			labels = append(labels, []byte(label)...)
		}
		// Add null terminator (0x00)
		labels = append(labels, 0)

		// Check total size before adding
		if len(result)+len(labels) > MaxDHCPOptionLen {
			return nil, fmt.Errorf("%w: %d bytes", ErrDomainSearchListExceedsMaxLen, MaxDHCPOptionLen)
		}

		result = append(result, labels...)
	}

	return result, nil
}

// splitDomain splits a domain name into labels.
func splitDomain(domain string) []string {
	if domain == "" {
		return nil
	}
	// Remove trailing dot if present
	if domain[len(domain)-1] == '.' {
		domain = domain[:len(domain)-1]
	}

	labels := []string{}
	start := 0

	for i := range len(domain) {
		if domain[i] == '.' {
			if i > start {
				labels = append(labels, domain[start:i])
			}

			start = i + 1
		}
	}
	// Add last label
	if start < len(domain) {
		labels = append(labels, domain[start:])
	}

	return labels
}
