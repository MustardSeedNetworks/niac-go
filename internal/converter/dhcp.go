package converter

import (
	"strings"
)

// dhcpParseState holds state during DHCP parsing.
type dhcpParseState struct {
	dhcp         *DhcpServer
	currentLease *DhcpLease
}

// parseDhcpClientIP parses YourClientIpAddr and starts a new lease.
func (p *Parser) parseDhcpClientIP(line string) *DhcpLease {
	ip := p.extractValue(line)
	if ip == "" {
		// No closing paren on same line, extract everything after (
		if _, after, found := strings.Cut(line, "("); found {
			ip = strings.TrimSpace(after)
		}
	}

	return &DhcpLease{ClientIP: ip}
}

// parseDhcpMacAddrMask handles MacAddrMask and finalizes the current lease.
// Returns true if the caller should continue (skip pos increment).
func (p *Parser) parseDhcpMacAddrMask(line string, state *dhcpParseState) bool {
	if state.currentLease == nil {
		return false
	}

	state.currentLease.MacAddrMask = p.formatMAC(p.extractValue(line))
	state.dhcp.ClientLeases = append(state.dhcp.ClientLeases, *state.currentLease)
	state.currentLease = nil

	// Skip the closing paren of YourClientIpAddr block on next line
	p.pos++
	if p.pos < len(p.lines) && strings.TrimSpace(p.lines[p.pos]) == ")" {
		p.pos++ // Skip the closing paren
	}

	return true
}

// parseDhcpServerField parses server-level DHCP fields (not lease-specific).
func (p *Parser) parseDhcpServerField(line string, dhcp *DhcpServer) {
	switch {
	case strings.HasPrefix(line, "SubnetMask"):
		dhcp.SubnetMask = p.extractValue(line)
	case strings.HasPrefix(line, "Router"):
		if value := p.extractValue(line); value != "" {
			dhcp.Router = strings.Fields(value)[0]
		}
	case strings.HasPrefix(line, "DomainNameServer"):
		dhcp.DomainNameServer = p.extractValue(line)
	case strings.HasPrefix(line, "NextServerIpAddr"):
		dhcp.NextServerIP = p.extractValue(line)
	case strings.HasPrefix(line, "ServerIdentifier"):
		dhcp.ServerIdentifier = p.extractValue(line)
	}
}

// parseDhcp parses a Dhcp block.
func (p *Parser) parseDhcp() *DhcpServer {
	p.pos++ // Skip opening line
	state := &dhcpParseState{
		dhcp: &DhcpServer{
			ClientLeases: make([]DhcpLease, 0),
		},
	}

	for p.pos < len(p.lines) {
		line := strings.TrimSpace(p.lines[p.pos])

		if line == ")" {
			// Save current lease if exists
			if state.currentLease != nil {
				state.dhcp.ClientLeases = append(state.dhcp.ClientLeases, *state.currentLease)
			}
			p.pos++

			break
		}

		// Skip comments
		if strings.HasPrefix(line, "//") || strings.HasPrefix(line, "#") {
			p.pos++

			continue
		}

		// Handle lease-specific fields
		switch {
		case strings.HasPrefix(line, "YourClientIpAddr"):
			state.currentLease = p.parseDhcpClientIP(line)
		case strings.HasPrefix(line, "MacAddrValue"):
			if state.currentLease != nil {
				state.currentLease.MacAddrValue = p.formatMAC(p.extractValue(line))
			}
		case strings.HasPrefix(line, "MacAddrMask"):
			if p.parseDhcpMacAddrMask(line, state) {
				continue
			}
		default:
			// Handle server-level fields
			p.parseDhcpServerField(line, state.dhcp)
		}

		p.pos++
	}

	return state.dhcp
}
