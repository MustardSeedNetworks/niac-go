package converter

import (
	"fmt"
	"regexp"
	"strings"
)

// parseSnmpAgentInclude handles Include directive for SNMP agent.
func (p *Parser) parseSnmpAgentInclude(line string, agent *SnmpAgent) {
	walk := p.extractString(line)
	if walk == "" {
		return
	}

	if agent.WalkFile == "" {
		agent.WalkFile = walk
	}
	agent.WalkFiles = append(agent.WalkFiles, walk)
}

// parseSnmpAgentCommunityInclude handles CommunityInclude directive.
func (p *Parser) parseSnmpAgentCommunityInclude(line string, agent *SnmpAgent) {
	community, walk := p.parseCommunityInclude(line)
	if community == "" || walk == "" {
		return
	}
	agent.CommunityIncludes = append(agent.CommunityIncludes, CommunityInclude{
		Community: community,
		WalkFile:  walk,
	})
}

// parseSnmpAgentField parses a single SNMP agent field.
func (p *Parser) parseSnmpAgentField(line string, agent *SnmpAgent) {
	switch {
	case strings.HasPrefix(line, "Include("):
		p.parseSnmpAgentInclude(line, agent)
	case strings.HasPrefix(line, "CommunityInclude("):
		p.parseSnmpAgentCommunityInclude(line, agent)
	case strings.HasPrefix(line, "SnmpAddr("):
		agent.SnmpAddr = p.extractValue(line)
	case strings.HasPrefix(line, "Dot1D_FdbTable("):
		if cfg := p.parseFdbTable(line, true); cfg != nil {
			agent.Dot1DFdbTable = cfg
		}
	case strings.HasPrefix(line, "Dot1Q_FdbTable("):
		if cfg := p.parseFdbTable(line, false); cfg != nil {
			agent.Dot1QFdbTable = cfg
		}
	case strings.HasPrefix(line, "AddMib("):
		mibLine := p.collectQuotedDirective(line, addMibQuotedArgs)
		if mib := p.parseAddMib(mibLine); mib != nil {
			agent.AddMibs = append(agent.AddMibs, *mib)
		}
	}
}

// parseSnmpAgent parses an SnmpAgent block.
func (p *Parser) parseSnmpAgent() *SnmpAgent {
	p.pos++ // Skip opening line
	agent := &SnmpAgent{}

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

		p.parseSnmpAgentField(line, agent)
		p.pos++
	}

	return agent
}

// parseSnmpAccessList parses SnmpAccessList block.
func (p *Parser) parseSnmpAccessList() []string {
	p.pos++ // Skip opening line
	var accessList []string

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
		if strings.HasPrefix(line, "IpAddr(") {
			ip := p.extractValue(line)
			if ip != "" {
				accessList = append(accessList, ip)
			}
		}
		p.pos++
	}

	return accessList
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
func (p *Parser) parseFdbTable(line string, dot1d bool) *FdbTableConfig {
	re := regexp.MustCompile(`\(([^)]*)\)`)
	match := re.FindStringSubmatch(line)
	if len(match) < minRegexMatchParts {
		return nil
	}
	parts := strings.Fields(match[1])
	if len(parts) == 0 {
		return nil
	}
	cfg := &FdbTableConfig{}
	if _, err := fmt.Sscanf(parts[0], "%d", &cfg.Port); err != nil {
		return nil
	}
	if len(parts) > 1 {
		_, _ = fmt.Sscanf(parts[1], "%d", &cfg.VLAN)
	} else if !dot1d {
		return nil
	}

	return cfg
}
