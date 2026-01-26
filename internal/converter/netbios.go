package converter

import (
	"regexp"
	"strconv"
	"strings"
)

// parseNetBiosStatus parses NetBiosStatus block.
func (p *Parser) parseNetBiosStatus() *NetbiosConfig {
	p.pos++ // Skip opening line
	netbios := &NetbiosConfig{
		Enabled: true,
	}

	for p.pos < len(p.lines) {
		line := strings.TrimSpace(p.lines[p.pos])
		if line == ")" {
			p.pos++

			break
		}

		if strings.HasPrefix(line, "//") || strings.HasPrefix(line, "#") {
			p.pos++

			continue
		}

		switch {
		case strings.HasPrefix(line, "MsBrowse"):
			netbios.MsBrowse = true
		case strings.HasPrefix(line, "Bnode"):
			netbios.NodeType = "B"
		case strings.HasPrefix(line, "Pnode"):
			netbios.NodeType = "P"
		case strings.HasPrefix(line, "Mnode"):
			netbios.NodeType = "M"
		case strings.HasPrefix(line, "NetBiosName("):
			name := p.parseNetBiosName(line)
			if name != nil {
				netbios.Names = append(netbios.Names, *name)
			}
		}

		p.pos++
	}

	return netbios
}

// parseNetBiosName parses NetBiosName("name" suffix groupflag).
func (p *Parser) parseNetBiosName(line string) *NetbiosName {
	// Extract quoted name
	re := regexp.MustCompile(`"([^"]+)"`)
	matches := re.FindAllStringSubmatch(line, -1)
	if len(matches) == 0 {
		return nil
	}
	name := matches[0][1]

	// Remove quoted name to parse remaining tokens
	remaining := re.ReplaceAllString(line, "")
	remaining = strings.ReplaceAll(remaining, "NetBiosName", "")
	remaining = strings.ReplaceAll(remaining, "(", "")
	remaining = strings.ReplaceAll(remaining, ")", "")
	tokens := strings.Fields(remaining)

	entry := &NetbiosName{Name: name}

	for _, token := range tokens {
		switch token {
		case "Machine":
			entry.Suffix = "0"
		case "MsBrowse":
			entry.Suffix = "1"
		case "MsgServ":
			entry.Suffix = "3"
		case "MbSubnet":
			entry.Suffix = "29"
		case "MbElect":
			entry.Suffix = "30"
		case "LanMan":
			entry.Suffix = "32"
		case "Group":
			entry.Group = true
		case "Unique":
			entry.Group = false
		default:
			if _, err := strconv.Atoi(token); err == nil {
				entry.Suffix = token
			}
		}
	}

	return entry
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
