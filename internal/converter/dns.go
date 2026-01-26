package converter

import (
	"fmt"
	"regexp"
	"strings"
)

// parseDNS parses a DNS block.
func (p *Parser) parseDNS() *DNSServer {
	p.pos++ // Skip opening line
	dns := &DNSServer{
		ForwardRecords: make([]DNSRecord, 0),
		ReverseRecords: make([]DNSRecord, 0),
	}

	for p.pos < len(p.lines) {
		line := strings.TrimSpace(p.lines[p.pos])

		if line == ")" {
			p.pos++

			break
		}

		// Skip comments
		if strings.HasPrefix(line, "//") || strings.HasPrefix(line, "#") {
			p.pos++

			continue
		}

		switch {
		case strings.HasPrefix(line, "Forward2("), strings.HasPrefix(line, "Forward("):
			record := p.parseDNSRecord(line, true)
			if record != nil {
				dns.ForwardRecords = append(dns.ForwardRecords, *record)
			}
		case strings.HasPrefix(line, "Reverse2("), strings.HasPrefix(line, "Reverse("):
			record := p.parseDNSRecord(line, false)
			if record != nil {
				dns.ReverseRecords = append(dns.ReverseRecords, *record)
			}
		}

		p.pos++
	}

	return dns
}

// parseDNSRecordTTLAndRCode parses optional TTL and RCode fields from DNS record parts.
func (p *Parser) parseDNSRecordTTLAndRCode(parts []string, record *DNSRecord) {
	if len(parts) >= dnsPartsWithTTL {
		_, _ = fmt.Sscanf(parts[2], "%d", &record.TTL)
	}

	if len(parts) >= dnsPartsWithRCode {
		_, _ = fmt.Sscanf(parts[3], "%d", &record.RCode)
	}
}

// parseDNSRecord parses a Forward() or Reverse() DNS record.
func (p *Parser) parseDNSRecord(line string, isForward bool) *DNSRecord {
	// Forward("hostname" IP TTL)
	// Forward2("hostname" IP TTL RCODE)
	// Reverse(IP "hostname" TTL)
	// Reverse2(IP "hostname" TTL RCODE)
	re := regexp.MustCompile(`\((.*?)\)`)
	match := re.FindStringSubmatch(line)
	if len(match) < minRegexMatchParts {
		return nil
	}

	parts := strings.Fields(match[1])
	if len(parts) < routerFieldCount {
		return nil
	}

	record := &DNSRecord{}

	if isForward {
		// Forward("hostname" IP TTL)
		record.Name = strings.Trim(parts[0], "\"")
		record.IP = parts[1]
	} else {
		// Reverse(IP "hostname" TTL)
		record.IP = parts[0]
		record.Name = strings.Trim(parts[1], "\"")
	}

	p.parseDNSRecordTTLAndRCode(parts, record)

	return record
}
