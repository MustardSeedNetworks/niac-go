package converter

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

func mergeSnmpAgent(base *SnmpAgent, incoming *SnmpAgent) *SnmpAgent {
	if base == nil {
		return incoming
	}
	if incoming == nil {
		return base
	}
	if base.Enabled == nil {
		base.Enabled = incoming.Enabled
	}
	if base.Community == "" {
		base.Community = incoming.Community
	}
	if base.SysName == "" {
		base.SysName = incoming.SysName
	}
	if base.SysDescr == "" {
		base.SysDescr = incoming.SysDescr
	}
	if base.SysContact == "" {
		base.SysContact = incoming.SysContact
	}
	if base.SysLocation == "" {
		base.SysLocation = incoming.SysLocation
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

// parseTTL parses TTL(ttl ip mask).
func (p *Parser) parseTTL(line string) *TTLConfig {
	re := regexp.MustCompile(`\(([^)]*)\)`)
	match := re.FindStringSubmatch(line)
	if len(match) < minRegexMatchParts {
		return nil
	}
	parts := strings.Fields(match[1])
	if len(parts) < ttlFieldCount {
		return nil
	}
	var ttl int
	if _, err := fmt.Sscanf(parts[0], "%d", &ttl); err != nil {
		return nil
	}

	return &TTLConfig{
		TTL:  ttl,
		IP:   parts[1],
		Mask: parts[2],
	}
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

		if p.parseRAField(line, ra) {
			continue
		}

		p.pos++
	}

	return ra
}

// parseRAField parses a single router advertisement field line.
// Returns true if the field was a PrefixInformation (already advanced pos).
func (p *Parser) parseRAField(line string, ra *Icmpv6RouterAdvertisement) bool {
	fieldParsers := []struct {
		prefix string
		format string
		target *int
	}{
		{"Period(", "Period(%d)", &ra.Period},
		{"CurHopLimit(", "CurHopLimit(%d)", &ra.CurHopLimit},
		{"Managed(", "Managed(%d)", &ra.Managed},
		{"Other(", "Other(%d)", &ra.Other},
		{"Lifetime(", "Lifetime(%d)", &ra.Lifetime},
		{"ReachableTime(", "ReachableTime(%d)", &ra.ReachableTime},
		{"RetransTimer(", "RetransTimer(%d)", &ra.RetransTimer},
		{"MTU(", "MTU(%d)", &ra.MTU},
	}

	for _, fp := range fieldParsers {
		if strings.HasPrefix(line, fp.prefix) {
			_, _ = fmt.Sscanf(line, fp.format, fp.target)

			return false
		}
	}

	if strings.HasPrefix(line, "PrefixInformation(") {
		prefix := p.parseIcmpv6PrefixInformation()
		if prefix != nil {
			ra.PrefixInfo = append(ra.PrefixInfo, *prefix)
		}

		return true
	}

	return false
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

// collectQuotedDirective joins lines until at least minQuotes quoted strings are present.

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

// extractString extracts a quoted string from a directive.
