package devicecli

import (
	"fmt"
	"strings"

	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
)

func renderInterfaces(interfaces []devicestate.Interface) string {
	var output strings.Builder
	output.WriteString("Interface              IP-Address      Status    Protocol\n")
	for _, iface := range interfaces {
		_, _ = fmt.Fprintf(
			&output, "%-22s %-15s %-9s %s\n",
			iface.Name, iface.Address.Addr(), upDown(iface.AdminUp), upDown(iface.OperUp),
		)
	}
	return strings.TrimSuffix(output.String(), "\n")
}

func renderRoutes(routes []devicestate.Route) string {
	var output strings.Builder
	for _, route := range routes {
		if route.Connected {
			_, _ = fmt.Fprintf(
				&output, "C %s is directly connected, %s\n", route.Destination, route.Via,
			)
			continue
		}
		_, _ = fmt.Fprintf(
			&output, "S %s [1/0] via %s, %s\n", route.Destination, route.NextHop, route.Via,
		)
	}
	return strings.TrimSuffix(output.String(), "\n")
}

func renderConfiguration(snapshot devicestate.Snapshot) string {
	var output strings.Builder
	_, _ = fmt.Fprintf(&output, "hostname %s\n!\n", snapshot.Identity.Hostname)
	for _, iface := range snapshot.Network.Interfaces {
		_, _ = fmt.Fprintf(&output, "interface %s\n", iface.Name)
		if iface.Description != "" {
			_, _ = fmt.Fprintf(&output, " description %s\n", iface.Description)
		}
		if iface.Address.IsValid() {
			_, _ = fmt.Fprintf(&output, " ip address %s\n", iface.Address)
		}
		if !iface.AdminUp {
			output.WriteString(" shutdown\n")
		}
		output.WriteString("!\n")
	}
	for _, vlan := range snapshot.Network.VLANs {
		_, _ = fmt.Fprintf(&output, "vlan %d\n", vlan.ID)
		if vlan.Name != "" {
			_, _ = fmt.Fprintf(&output, " name %s\n", vlan.Name)
		}
		output.WriteString("!\n")
	}
	for _, router := range snapshot.Network.Routers {
		_, _ = fmt.Fprintf(&output, "router %s %s\n", router.Protocol, router.ProcessID)
		for _, network := range router.Networks {
			_, _ = fmt.Fprintf(
				&output, " network %s %s area %s\n",
				network.Address, network.Wildcard, network.Area,
			)
		}
		output.WriteString("!\n")
	}
	for _, route := range snapshot.Network.Routes {
		if !route.Connected {
			_, _ = fmt.Fprintf(
				&output, "ip route %s %s %s\n", route.Destination, route.NextHop, route.Via,
			)
		}
	}
	return strings.TrimSuffix(output.String(), "\n")
}

func renderEvents(events []devicestate.Event) string {
	var output strings.Builder
	output.WriteString("Version  Event                  Target\n")
	for _, event := range events {
		_, _ = fmt.Fprintf(&output, "%-8d %-22s %s\n", event.Version, event.Kind, event.Target)
	}
	return strings.TrimSuffix(output.String(), "\n")
}

func upDown(up bool) string {
	if up {
		return "up"
	}
	return "down"
}
