package converter

import (
	"fmt"
	"strings"
)

func (p *Parser) parseCapturePlayback() (*CapturePlayback, error) {
	p.pos++ // Skip opening line
	playback := &CapturePlayback{}

	for p.pos < len(p.lines) {
		line := strings.TrimSpace(p.lines[p.pos])

		if line == ")" {
			p.pos++

			break
		}

		switch {
		case strings.HasPrefix(line, "FileName("):
			playback.FileName = p.extractString(line)
		case strings.HasPrefix(line, "LoopTime("):
			var loopTime int
			n, err := fmt.Sscanf(line, "LoopTime(%d)", &loopTime)
			if err != nil || n != 1 {
				return nil, fmt.Errorf("line %d: %w: %s", p.pos+1, ErrInvalidLoopTimeFormat, line)
			}
			playback.LoopTime = loopTime
		case strings.HasPrefix(line, "ScaleTime("):
			var scaleTime float64
			n, err := fmt.Sscanf(line, "ScaleTime(%f)", &scaleTime)
			if err != nil || n != 1 {
				return nil, fmt.Errorf("line %d: %w: %s", p.pos+1, ErrInvalidScaleTimeFormat, line)
			}
			playback.ScaleTime = scaleTime
		}

		p.pos++
	}

	return playback, nil
}

// parseDeviceSimpleField handles simple device fields that don't require nested parsing.
// Returns true if the field was handled.
func (p *Parser) parseDeviceSimpleField(line string, device *Device) bool {
	switch {
	case strings.HasPrefix(line, "MacAddr("):
		device.MAC = p.formatMAC(p.extractValue(line))
	case strings.HasPrefix(line, "IpAddr("):
		p.parseDeviceIPAddr(line, device)
	case strings.HasPrefix(line, "Ip6Addr("):
		if ip := p.extractValue(line); ip != "" {
			device.IPs = append(device.IPs, ip)
		}
	case strings.HasPrefix(line, "MapToIp("):
		device.MapToIP = p.extractValue(line)
	case strings.HasPrefix(line, "Babble("):
		device.Babble = true
	case strings.HasPrefix(line, "TTL("):
		if ttlCfg := p.parseTTL(line); ttlCfg != nil {
			device.TTL = ttlCfg
		}
	case strings.HasPrefix(line, "SpanningTree("):
		p.parseDeviceSpanningTree(device)
	default:
		return false
	}

	return true
}

// parseDeviceIPAddr handles IpAddr field parsing.
func (p *Parser) parseDeviceIPAddr(line string, device *Device) {
	if ip := p.extractValue(line); ip != "" {
		device.IPs = append(device.IPs, ip)
	}
}

// parseDeviceSpanningTree handles SpanningTree field parsing.
func (p *Parser) parseDeviceSpanningTree(device *Device) {
	if device.Stp == nil {
		device.Stp = &StpConfig{Enabled: true}
	} else {
		device.Stp.Enabled = true
	}
}

// parseDeviceVlan parses the VLAN field and returns an error if invalid.
func (p *Parser) parseDeviceVlan(line string) (int, error) {
	var vlan int
	n, err := fmt.Sscanf(line, "Vlan(%d)", &vlan)
	if err != nil || n != 1 {
		return 0, fmt.Errorf("line %d: %w: %s", p.pos+1, ErrInvalidVlanFormat, line)
	}

	return vlan, nil
}

// parseDeviceNestedBlock handles device fields that require nested block parsing.
// Returns true if parsing occurred and pos should not be incremented.
func (p *Parser) parseDeviceNestedBlock(line string, device *Device) bool {
	switch {
	case strings.HasPrefix(line, "SnmpAgent("):
		agent := p.parseSnmpAgent()
		device.SnmpAgent = mergeSnmpAgent(device.SnmpAgent, agent)

		return true
	case strings.HasPrefix(line, "SnmpAccessList("):
		accessList := p.parseSnmpAccessList()
		if device.SnmpAgent == nil {
			device.SnmpAgent = &SnmpAgent{}
		}
		device.SnmpAgent.AccessList = append(device.SnmpAgent.AccessList, accessList...)

		return true
	case strings.HasPrefix(line, "NetBiosStatus("):
		device.Netbios = mergeNetbios(device.Netbios, p.parseNetBiosStatus())

		return true
	case strings.HasPrefix(line, "Icmp("):
		device.Icmp = p.parseIcmp()

		return true
	case strings.HasPrefix(line, "Icmp6("):
		device.Icmpv6 = p.parseIcmp6()

		return true
	case strings.HasPrefix(line, "Dhcp("):
		device.Dhcp = p.parseDhcp()

		return true
	case strings.HasPrefix(line, "Dns("):
		device.DNS = p.parseDNS()

		return true
	}

	return false
}

// parseDevice parses a Device block.
func (p *Parser) parseDevice(deviceNum int) (*Device, error) {
	p.pos++ // Skip opening line
	device := &Device{
		Name: fmt.Sprintf("device%d", deviceNum+1),
	}

	for p.pos < len(p.lines) {
		line := strings.TrimSpace(p.lines[p.pos])

		if line == ")" {
			p.pos++

			break
		}

		// Handle simple fields first
		if p.parseDeviceSimpleField(line, device) {
			p.pos++

			continue
		}

		// Handle VLAN separately due to error return
		if strings.HasPrefix(line, "Vlan(") {
			vlan, err := p.parseDeviceVlan(line)
			if err != nil {
				return nil, err
			}
			device.VLAN = vlan
			p.pos++

			continue
		}

		// Handle nested blocks
		if p.parseDeviceNestedBlock(line, device) {
			continue
		}

		p.pos++
	}

	return device, nil
}
