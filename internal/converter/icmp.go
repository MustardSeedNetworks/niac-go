package converter

import (
	"fmt"
	"regexp"
	"strings"
)

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
