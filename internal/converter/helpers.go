package converter

import (
	"fmt"
	"regexp"
	"strings"
)

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

// parseAddMib parses an AddMib directive.
func (p *Parser) parseAddMib(line string) *AddMib {
	// Extract quoted strings (OID, type, value)
	re := regexp.MustCompile(`"([^"]+)"`)
	matches := re.FindAllStringSubmatch(line, -1)
	if len(matches) < addMibFieldCount {
		return nil
	}

	return &AddMib{
		OID:   matches[0][1],
		Type:  matches[1][1],
		Value: matches[2][1],
	}
}

// parseCommunityInclude parses CommunityInclude("community" "walkfile").
func (p *Parser) parseCommunityInclude(line string) (string, string) {
	re := regexp.MustCompile(`"([^"]+)"`)
	matches := re.FindAllStringSubmatch(line, -1)
	if len(matches) < communityIncludeFields {
		return "", ""
	}

	return matches[0][1], matches[1][1]
}

// parseFdbTable parses Dot1D_FdbTable or Dot1Q_FdbTable directives.

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

// MaxYAMLConfigSize caps in-memory YAML config size at 16 MiB. The check
// guards against pathological alias-bomb / billion-laughs payloads that would
// otherwise be expanded by yaml.Unmarshal. Real configs in our corpus top out
// well under 1 MiB.
