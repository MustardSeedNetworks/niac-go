package converter

import (
	"fmt"
	"regexp"
	"strings"
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

// mergeSnmpAgent merges SNMP agent settings.
func mergeSnmpAgent(base *SnmpAgent, incoming *SnmpAgent) *SnmpAgent {
	if base == nil {
		return incoming
	}
	if incoming == nil {
		return base
	}
	if base.WalkFile == "" {
		base.WalkFile = incoming.WalkFile
	}
	base.WalkFiles = append(base.WalkFiles, incoming.WalkFiles...)
	base.AddMibs = append(base.AddMibs, incoming.AddMibs...)
	base.CommunityIncludes = append(base.CommunityIncludes, incoming.CommunityIncludes...)
	base.AccessList = append(base.AccessList, incoming.AccessList...)
	if base.SnmpAddr == "" {
		base.SnmpAddr = incoming.SnmpAddr
	}
	if base.Dot1DFdbTable == nil {
		base.Dot1DFdbTable = incoming.Dot1DFdbTable
	}
	if base.Dot1QFdbTable == nil {
		base.Dot1QFdbTable = incoming.Dot1QFdbTable
	}
	if base.Traps == nil {
		base.Traps = incoming.Traps
	}

	return base
}

// mergeNetbios merges NetBIOS configuration.
func mergeNetbios(base *NetbiosConfig, incoming *NetbiosConfig) *NetbiosConfig {
	if base == nil {
		return incoming
	}
	if incoming == nil {
		return base
	}
	if base.Name == "" {
		base.Name = incoming.Name
	}
	if base.Workgroup == "" {
		base.Workgroup = incoming.Workgroup
	}
	if base.NodeType == "" {
		base.NodeType = incoming.NodeType
	}
	if len(base.Services) == 0 {
		base.Services = incoming.Services
	}
	if base.TTL == 0 {
		base.TTL = incoming.TTL
	}
	base.Names = append(base.Names, incoming.Names...)
	base.MsBrowse = base.MsBrowse || incoming.MsBrowse
	base.Enabled = base.Enabled || incoming.Enabled

	return base
}
