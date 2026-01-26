package protocols

import (
	"encoding/binary"
	"fmt"
	"net"
	"os"

	"github.com/google/gopacket/layers"

	"github.com/krisarmstrong/niac-go/internal/safeconv"
)

// buildDHCPOptions constructs the DHCP options for a response.
// Caller must hold h.mu (at least RLock).
func (h *DHCPHandler) buildDHCPOptions(
	msgType uint8,
	serverIP net.IP,
	clientMAC net.HardwareAddr,
) []layers.DHCPOption {
	options := []layers.DHCPOption{
		{Type: layers.DHCPOptMessageType, Length: 1, Data: []byte{msgType}},
		{Type: layers.DHCPOptServerID, Length: dhcpIPv4Len, Data: []byte(serverIP.To4())},
		{Type: layers.DHCPOptLeaseTime, Length: dhcpIPv4Len, Data: encodeUint32(uint32(DefaultLeaseTime.Seconds()))},
		{Type: layers.DHCPOptSubnetMask, Length: dhcpIPv4Len, Data: []byte(h.subnetMask.To4())},
	}

	options = h.appendNetworkOptions(options)
	options = h.appendTimingOptions(options)
	options = h.appendBootOptions(options)
	options = h.appendMiscOptions(options, clientMAC)

	options = append(options, layers.DHCPOption{Type: layers.DHCPOptEnd})

	return options
}

// appendNetworkOptions adds gateway, DNS, and domain options.
// Caller must hold h.mu (at least RLock).
func (h *DHCPHandler) appendNetworkOptions(options []layers.DHCPOption) []layers.DHCPOption {
	if h.gateway != nil {
		options = append(options, layers.DHCPOption{
			Type: layers.DHCPOptRouter, Length: dhcpIPv4Len, Data: []byte(h.gateway.To4()),
		})
	}

	if len(h.dnsServers) > 0 {
		dnsData := make([]byte, 0, len(h.dnsServers)*dhcpIPv4Len)
		for _, dns := range h.dnsServers {
			dnsData = append(dnsData, []byte(dns.To4())...)
		}

		dnsLen := min(len(dnsData), dhcpMaxByte)
		options = append(options, layers.DHCPOption{
			Type: layers.DHCPOptDNS, Length: safeconv.Uint8(dnsLen), Data: dnsData[:dnsLen],
		})
	}

	if h.domainName != "" {
		domainLen := min(len(h.domainName), dhcpMaxByte)
		options = append(options, layers.DHCPOption{
			Type: layers.DHCPOptDomainName, Length: safeconv.Uint8(domainLen), Data: []byte(h.domainName[:domainLen]),
		})
	}

	return options
}

// appendTimingOptions adds T1, T2, and NTP server options.
// Caller must hold h.mu (at least RLock).
func (h *DHCPHandler) appendTimingOptions(options []layers.DHCPOption) []layers.DHCPOption {
	// Add renewal time (T1) - 50% of lease time
	options = append(options, layers.DHCPOption{
		Type: layers.DHCPOptT1, Length: dhcpIPv4Len,
		Data: encodeUint32(uint32(DefaultLeaseTime.Seconds() / t1RenewalDivisor)),
	})

	// Add rebinding time (T2) - 87.5% of lease time
	options = append(options, layers.DHCPOption{
		Type: layers.DHCPOptT2, Length: dhcpIPv4Len,
		Data: encodeUint32(uint32(DefaultLeaseTime.Seconds() * t2RebindNumerator / t2RebindDivisor)),
	})

	// Add NTP servers if configured (Option 42)
	if len(h.ntpServers) > 0 {
		ntpData := make([]byte, 0, len(h.ntpServers)*dhcpIPv4Len)
		for _, ntp := range h.ntpServers {
			ntpData = append(ntpData, []byte(ntp.To4())...)
		}

		ntpLen := min(len(ntpData), dhcpMaxByte)
		options = append(options, layers.DHCPOption{
			Type: DHCPOptNTP, Length: safeconv.Uint8(ntpLen), Data: ntpData[:ntpLen],
		})
	}

	return options
}

// appendBootOptions adds TFTP, bootfile, and domain search options.
// Caller must hold h.mu (at least RLock).
func (h *DHCPHandler) appendBootOptions(options []layers.DHCPOption) []layers.DHCPOption {
	if h.tftpServerName != "" {
		tftpLen := min(len(h.tftpServerName), dhcpMaxByte)
		options = append(options, layers.DHCPOption{
			Type: DHCPOptTFTPServer, Length: safeconv.Uint8(tftpLen), Data: []byte(h.tftpServerName[:tftpLen]),
		})
	}

	if h.bootfileName != "" {
		bootLen := min(len(h.bootfileName), dhcpMaxByte)
		options = append(options, layers.DHCPOption{
			Type: DHCPOptBootfileName, Length: safeconv.Uint8(bootLen), Data: []byte(h.bootfileName[:bootLen]),
		})
	}

	if len(h.domainSearch) > 0 {
		searchData, err := encodeDomainSearchList(h.domainSearch)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: DHCP domain search encoding failed: %v\n", err)
		} else if len(searchData) > 0 {
			searchLen := min(len(searchData), dhcpMaxByte)
			options = append(options, layers.DHCPOption{
				Type: DHCPOptDomainSearch, Length: safeconv.Uint8(searchLen), Data: searchData[:searchLen],
			})
		}
	}

	return options
}

// appendMiscOptions adds vendor info and hostname options.
// Caller must hold h.mu (at least RLock).
func (h *DHCPHandler) appendMiscOptions(options []layers.DHCPOption, clientMAC net.HardwareAddr) []layers.DHCPOption {
	if len(h.vendorSpecificInfo) > 0 {
		vendorLen := min(len(h.vendorSpecificInfo), dhcpMaxByte)
		options = append(options, layers.DHCPOption{
			Type: layers.DHCPOptVendorOption, Length: safeconv.Uint8(vendorLen), Data: h.vendorSpecificInfo[:vendorLen],
		})
	}

	if lease, ok := h.leases[clientMAC.String()]; ok && lease.Hostname != "" {
		hostnameLen := min(len(lease.Hostname), dhcpMaxByte)
		options = append(options, layers.DHCPOption{
			Type: layers.DHCPOptHostname, Length: safeconv.Uint8(hostnameLen), Data: []byte(lease.Hostname[:hostnameLen]),
		})
	}

	return options
}

// encodeUint32 encodes a uint32 as big-endian bytes.
func encodeUint32(val uint32) []byte {
	b := make([]byte, dhcpIPv4Len)
	binary.BigEndian.PutUint32(b, val)

	return b
}

// encodeDomainSearchList encodes a domain search list in DNS label format (RFC 1035)
// Used for DHCP Option 119 (Domain Search)
// Returns error if constraints are violated.
func encodeDomainSearchList(domains []string) ([]byte, error) {
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
			labels = append(labels, byte(len(label)))
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
