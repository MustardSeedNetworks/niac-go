package converter

import (
	"fmt"
	"regexp"
	"strings"
)

// Parser constants for field counts and lengths.
const (
	addMibQuotedArgs       = 3  // number of quoted args in AddMib directive
	minRegexMatchParts     = 2  // minimum parts from regex match
	ttlFieldCount          = 3  // TTL has ttl, ip, mask
	routerFieldCount       = 2  // Router has address, preference
	addMibFieldCount       = 3  // AddMib has OID, type, value
	communityIncludeFields = 2  // CommunityInclude has community, walkfile
	dnsPartsWithTTL        = 3  // DNS record with TTL
	dnsPartsWithRCode      = 4  // DNS record with RCode
	macAddressRawLen       = 12 // MAC address hex chars (XXXXXXXXXXXX)
)

// extractString extracts a quoted string from a directive.
func (p *Parser) extractString(line string) string {
	start := strings.Index(line, "\"")
	if start == -1 {
		return ""
	}
	end := strings.Index(line[start+1:], "\"")
	if end == -1 {
		return ""
	}

	return line[start+1 : start+1+end]
}

// extractValue extracts a value from parentheses (no quotes).
func (p *Parser) extractValue(line string) string {
	start := strings.Index(line, "(")
	if start == -1 {
		return ""
	}
	end := strings.Index(line[start+1:], ")")
	if end == -1 {
		return ""
	}

	return line[start+1 : start+1+end]
}

// formatMAC converts XXXXXXXXXXXX to XX:XX:XX:XX:XX:XX.
func (p *Parser) formatMAC(mac string) string {
	if len(mac) != macAddressRawLen {
		return mac
	}

	return fmt.Sprintf("%s:%s:%s:%s:%s:%s",
		mac[0:2], mac[2:4], mac[4:6], mac[6:8], mac[8:10], mac[10:12])
}

// collectQuotedDirective joins lines until at least minQuotes quoted strings are present.
func (p *Parser) collectQuotedDirective(line string, minQuotes int) string {
	combined := stripInlineComment(line)
	re := regexp.MustCompile(`"([^"]+)"`)
	for p.pos+1 < len(p.lines) && len(re.FindAllStringSubmatch(combined, -1)) < minQuotes {
		p.pos++
		next := stripInlineComment(strings.TrimSpace(p.lines[p.pos]))
		if next == "" {
			continue
		}
		combined += " " + next
	}

	return combined
}

// stripInlineComment removes inline comments (// or #) from a line while respecting quoted strings.
func stripInlineComment(line string) string {
	inQuote := false
	for i := range len(line) - 1 {
		if line[i] == '"' {
			inQuote = !inQuote

			continue
		}
		if !inQuote && line[i] == '/' && line[i+1] == '/' {
			return strings.TrimSpace(line[:i])
		}
		if !inQuote && line[i] == '#' {
			return strings.TrimSpace(line[:i])
		}
	}

	return strings.TrimSpace(line)
}
