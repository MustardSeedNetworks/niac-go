package converter

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

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

// parseIcmp parses Icmp block.
func (p *Parser) parseIcmp() *IcmpConfig {
	p.pos++ // Skip opening line
	icmp := &IcmpConfig{Enabled: true}

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

		if strings.HasPrefix(line, "AddressMaskReply(") {
			icmp.AddressMaskReply = p.extractValue(line)
		} else if strings.HasPrefix(line, "RouterAdvertisement(") {
			icmp.RouterAdvertisement = p.parseIcmpRouterAdvertisement()

			continue
		}

		p.pos++
	}

	return icmp
}

// parseIcmpRouter parses a single Router directive and returns an IcmpRouter.
func (p *Parser) parseIcmpRouter(line string) *IcmpRouter {
	re := regexp.MustCompile(`\(([^)]*)\)`)
	match := re.FindStringSubmatch(line)
	if len(match) < minRegexMatchParts {
		return nil
	}

	parts := strings.Fields(match[1])
	if len(parts) < routerFieldCount {
		return nil
	}

	var pref int
	_, _ = fmt.Sscanf(parts[1], "%d", &pref)

	return &IcmpRouter{
		Address:    parts[0],
		Preference: pref,
	}
}

// parseIcmpRAField parses a single field in the RouterAdvertisement block.
func (p *Parser) parseIcmpRAField(line string, ra *IcmpRouterAdvertisement) {
	switch {
	case strings.HasPrefix(line, "Period("):
		var period int
		if _, err := fmt.Sscanf(line, "Period(%d)", &period); err == nil {
			ra.Period = period
		}
	case strings.HasPrefix(line, "Lifetime("):
		var lifetime int
		if _, err := fmt.Sscanf(line, "Lifetime(%d)", &lifetime); err == nil {
			ra.Lifetime = lifetime
		}
	case strings.HasPrefix(line, "Router("):
		if router := p.parseIcmpRouter(line); router != nil {
			ra.Routers = append(ra.Routers, *router)
		}
	}
}

// parseIcmpRouterAdvertisement parses RouterAdvertisement block for IPv4.
func (p *Parser) parseIcmpRouterAdvertisement() *IcmpRouterAdvertisement {
	p.pos++ // Skip opening line
	ra := &IcmpRouterAdvertisement{}

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

		p.parseIcmpRAField(line, ra)
		p.pos++
	}

	return ra
}

// parseIcmp6 parses Icmp6 block.
func (p *Parser) parseIcmp6() *Icmpv6Config {
	p.pos++ // Skip opening line
	icmp6 := &Icmpv6Config{Enabled: true}

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

		if strings.HasPrefix(line, "RouterAdvertisement(") {
			ra := p.parseIcmpv6RouterAdvertisement()
			icmp6.RouterAdvertisement = ra

			continue
		}

		p.pos++
	}

	return icmp6
}

// parseIcmpv6RouterAdvertisement parses IPv6 RouterAdvertisement block.
func (p *Parser) parseIcmpv6RouterAdvertisement() *Icmpv6RouterAdvertisement {
	p.pos++ // Skip opening line
	ra := &Icmpv6RouterAdvertisement{}

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
		case strings.HasPrefix(line, "Period("):
			var v int
			_, _ = fmt.Sscanf(line, "Period(%d)", &v)
			ra.Period = v
		case strings.HasPrefix(line, "CurHopLimit("):
			var v int
			_, _ = fmt.Sscanf(line, "CurHopLimit(%d)", &v)
			ra.CurHopLimit = v
		case strings.HasPrefix(line, "Managed("):
			var v int
			_, _ = fmt.Sscanf(line, "Managed(%d)", &v)
			ra.Managed = v
		case strings.HasPrefix(line, "Other("):
			var v int
			_, _ = fmt.Sscanf(line, "Other(%d)", &v)
			ra.Other = v
		case strings.HasPrefix(line, "Lifetime("):
			var v int
			_, _ = fmt.Sscanf(line, "Lifetime(%d)", &v)
			ra.Lifetime = v
		case strings.HasPrefix(line, "ReachableTime("):
			var v int
			_, _ = fmt.Sscanf(line, "ReachableTime(%d)", &v)
			ra.ReachableTime = v
		case strings.HasPrefix(line, "RetransTimer("):
			var v int
			_, _ = fmt.Sscanf(line, "RetransTimer(%d)", &v)
			ra.RetransTimer = v
		case strings.HasPrefix(line, "MTU("):
			var v int
			_, _ = fmt.Sscanf(line, "MTU(%d)", &v)
			ra.MTU = v
		case strings.HasPrefix(line, "PrefixInformation("):
			prefix := p.parseIcmpv6PrefixInformation()
			if prefix != nil {
				ra.PrefixInfo = append(ra.PrefixInfo, *prefix)
			}

			continue
		}

		p.pos++
	}

	return ra
}

// parseIcmpv6PrefixInformation parses PrefixInformation block.
func (p *Parser) parseIcmpv6PrefixInformation() *Icmpv6PrefixInfo {
	p.pos++ // Skip opening line
	prefix := &Icmpv6PrefixInfo{}

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
		case strings.HasPrefix(line, "PrefixLength("):
			var v int
			_, _ = fmt.Sscanf(line, "PrefixLength(%d)", &v)
			prefix.PrefixLength = v
		case strings.HasPrefix(line, "Onlink("):
			var v int
			_, _ = fmt.Sscanf(line, "Onlink(%d)", &v)
			prefix.Onlink = v
		case strings.HasPrefix(line, "Auto("):
			var v int
			_, _ = fmt.Sscanf(line, "Auto(%d)", &v)
			prefix.Auto = v
		case strings.HasPrefix(line, "ValidLifetime("):
			var v int
			_, _ = fmt.Sscanf(line, "ValidLifetime(%d)", &v)
			prefix.ValidLifetime = v
		case strings.HasPrefix(line, "PreferredLifetime("):
			var v int
			_, _ = fmt.Sscanf(line, "PreferredLifetime(%d)", &v)
			prefix.PreferredLifetime = v
		case strings.HasPrefix(line, "Prefix("):
			prefix.Prefix = p.extractValue(line)
		}
		p.pos++
	}

	return prefix
}
