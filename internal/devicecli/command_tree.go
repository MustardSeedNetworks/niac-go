package devicecli

import (
	"fmt"
	"slices"
	"strings"
)

type commandSpec struct {
	syntax  string
	summary string
}

func commandsForMode(mode Mode) []commandSpec {
	switch mode {
	case ModeUser:
		return []commandSpec{
			{syntax: "enable", summary: "Enter privileged mode"},
			{syntax: commandExit, summary: "Close this session"},
		}
	case ModePrivileged:
		return []commandSpec{
			{syntax: "configure terminal", summary: "Enter global configuration mode"},
			{syntax: "show ip interface brief", summary: "Display interface status"},
			{syntax: "show ip route", summary: "Display the routing table"},
			{syntax: "show running-config", summary: "Display running configuration"},
			{syntax: "show startup-config", summary: "Display startup configuration"},
			{syntax: "show configuration events", summary: "Display configuration events"},
			{syntax: "write memory", summary: "Save running configuration"},
			{syntax: "write erase", summary: "Erase startup configuration"},
			{syntax: "reload", summary: "Reload startup configuration"},
			{syntax: "checkpoint <name>", summary: "Save a named checkpoint"},
			{syntax: "rollback checkpoint <name>", summary: "Restore a named checkpoint"},
			{syntax: "disable", summary: "Return to user mode"},
			{syntax: commandExit, summary: "Close this session"},
		}
	case ModeGlobalConfig:
		return []commandSpec{
			{syntax: "hostname <name>", summary: "Set the device hostname"},
			{syntax: "interface <name>", summary: "Configure an interface"},
			{syntax: "vlan <id>", summary: "Configure a VLAN"},
			{syntax: "router ospf <process-id>", summary: "Configure an OSPF process"},
			{syntax: "ip route <prefix> <next-hop> <interface>", summary: "Configure a static route"},
			{syntax: "end", summary: "Return to privileged mode"},
			{syntax: commandExit, summary: "Return to privileged mode"},
		}
	case ModeInterfaceConfig:
		return []commandSpec{
			{syntax: "description <text>", summary: "Set the interface description"},
			{syntax: "ip address <prefix>", summary: "Set the interface IPv4 prefix"},
			{syntax: "no ip address", summary: "Remove the interface IPv4 prefix"},
			{syntax: "shutdown", summary: "Administratively disable the interface"},
			{syntax: "no shutdown", summary: "Administratively enable the interface"},
			{syntax: "end", summary: "Return to privileged mode"},
			{syntax: commandExit, summary: "Return to global configuration mode"},
		}
	case ModeVLANConfig:
		return []commandSpec{
			{syntax: "name <name>", summary: "Set the VLAN name"},
			{syntax: "end", summary: "Return to privileged mode"},
			{syntax: commandExit, summary: "Return to global configuration mode"},
		}
	case ModeRouterConfig:
		return []commandSpec{
			{syntax: "network <address> <wildcard> area <id>", summary: "Add an OSPF network"},
			{syntax: "end", summary: "Return to privileged mode"},
			{syntax: commandExit, summary: "Return to global configuration mode"},
		}
	default:
		return nil
	}
}

// Complete returns command keywords matching prefix in the current mode.
func (s *Session) Complete(prefix string) []string {
	prefix = strings.TrimSpace(prefix)
	result := make([]string, 0)
	for _, command := range commandsForMode(s.mode) {
		keyword := strings.Fields(command.syntax)[0]
		if strings.HasPrefix(command.syntax, prefix) && !slices.Contains(result, keyword) {
			result = append(result, keyword)
		}
	}
	slices.Sort(result)
	return result
}

func (s *Session) help(prefix string) string {
	prefix = strings.TrimSpace(prefix)
	var output strings.Builder
	for _, command := range commandsForMode(s.mode) {
		if prefix != "" && !strings.HasPrefix(command.syntax, prefix) {
			continue
		}
		_, _ = fmt.Fprintf(&output, "%-42s %s\n", command.syntax, command.summary)
	}
	return strings.TrimSuffix(output.String(), "\n")
}
