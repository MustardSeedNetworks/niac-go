package devicecli_test

import (
	"strings"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/devicecli"
	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
)

func TestSessionModeTransitionsAndPrompt(t *testing.T) {
	session, _ := newSession()
	if got := session.Prompt(); got != "edge-1>" {
		t.Fatalf("prompt = %q, want edge-1>", got)
	}
	assertCommand(t, session, "enable", "")
	assertCommand(t, session, "configure terminal", "Enter configuration commands, one per line.")
	assertCommand(t, session, "interface Gi0/1", "")
	if got := session.Prompt(); got != "edge-1(config-if)#" {
		t.Fatalf("prompt = %q, want interface configuration prompt", got)
	}
	assertCommand(t, session, "end", "")
	if got := session.Prompt(); got != "edge-1#" {
		t.Fatalf("prompt = %q, want privileged prompt", got)
	}
}

func TestSessionMutatesSharedHostnameAndInterfaceState(t *testing.T) {
	session, state := newSession()
	assertCommand(t, session, "enable", "")
	assertCommand(t, session, "configure terminal", "Enter configuration commands, one per line.")
	assertCommand(t, session, "hostname branch-1", "")
	assertCommand(t, session, "interface Gi0/1", "")
	assertCommand(t, session, "shutdown", "")

	snapshot := state.Snapshot()
	if snapshot.Identity.Hostname != "branch-1" || snapshot.Network.Interfaces[0].AdminUp {
		t.Fatalf("state = %#v", snapshot)
	}
	if got := session.Prompt(); got != "branch-1(config-if)#" {
		t.Fatalf("prompt = %q, want updated hostname", got)
	}
}

func TestSessionShowOutputComesFromState(t *testing.T) {
	session, _ := newSession()
	assertCommand(t, session, "enable", "")

	response := session.Execute("show ip interface brief")
	for _, want := range []string{"Gi0/1", "10.0.0.1", "up", "up"} {
		if !strings.Contains(response.Output, want) {
			t.Fatalf("output %q does not contain %q", response.Output, want)
		}
	}
}

func TestSessionShowRoutesDistinguishesConnectedAndStaticRoutes(t *testing.T) {
	session, state := newSession()
	state.ReplaceNetwork(devicestate.Network{
		Interfaces: state.Snapshot().Network.Interfaces,
		Routes: []devicestate.Route{
			{Destination: mustPrefix("10.0.0.0/24"), Via: "Gi0/1", Connected: true},
			{
				Destination: mustPrefix("10.2.0.0/16"), Via: "Gi0/1",
				NextHop: mustAddr("10.0.0.2"),
			},
		},
	})
	assertCommand(t, session, "enable", "")

	response := session.Execute("show ip route")
	for _, want := range []string{
		"C 10.0.0.0/24 is directly connected, Gi0/1",
		"S 10.2.0.0/16 [1/0] via 10.0.0.2, Gi0/1",
	} {
		if !strings.Contains(response.Output, want) {
			t.Fatalf("output %q does not contain %q", response.Output, want)
		}
	}
}

func TestSessionRejectsUnknownInterfaceWithoutChangingMode(t *testing.T) {
	session, _ := newSession()
	assertCommand(t, session, "enable", "")
	assertCommand(t, session, "configure terminal", "Enter configuration commands, one per line.")

	response := session.Execute("interface Gi0/99")
	if response.Output != "% Interface not found" || session.Mode() != devicecli.ModeGlobalConfig {
		t.Fatalf("response = %#v mode = %q", response, session.Mode())
	}
}

func TestSessionRendersRunningAndStartupConfiguration(t *testing.T) {
	session, _ := newSession()
	assertCommand(t, session, "enable", "")
	assertCommand(t, session, "configure terminal", "Enter configuration commands, one per line.")
	assertCommand(t, session, "hostname branch-1", "")
	assertCommand(t, session, "interface Gi0/1", "")
	assertCommand(t, session, "shutdown", "")
	assertCommand(t, session, "end", "")

	running := session.Execute("show running-config").Output
	for _, want := range []string{"hostname branch-1", "interface Gi0/1", " shutdown"} {
		if !strings.Contains(running, want) {
			t.Fatalf("running config %q does not contain %q", running, want)
		}
	}
	startup := session.Execute("show startup-config").Output
	if !strings.Contains(startup, "hostname edge-1") || strings.Contains(startup, " shutdown") {
		t.Fatalf("startup config = %q", startup)
	}
}

func TestSessionSaveReloadAndEraseLifecycle(t *testing.T) {
	session, state := newSession()
	assertCommand(t, session, "enable", "")
	assertCommand(t, session, "configure terminal", "Enter configuration commands, one per line.")
	assertCommand(t, session, "hostname branch-1", "")
	assertCommand(t, session, "interface Gi0/1", "")
	assertCommand(t, session, "shutdown", "")
	assertCommand(t, session, "end", "")
	assertCommand(t, session, "write memory", "Building configuration...\n[OK]")
	assertCommand(t, session, "configure terminal", "Enter configuration commands, one per line.")
	assertCommand(t, session, "interface Gi0/1", "")
	assertCommand(t, session, "no shutdown", "")
	assertCommand(t, session, "end", "")

	if response := session.Execute("reload"); !response.Close {
		t.Fatalf("reload response = %#v, want closed session", response)
	}
	if state.Snapshot().Network.Interfaces[0].AdminUp {
		t.Fatal("reload did not restore saved shutdown state")
	}

	session = devicecli.NewSession(state)
	assertCommand(t, session, "enable", "")
	assertCommand(t, session, "write erase", "[OK]")
	if response := session.Execute("reload"); !response.Close {
		t.Fatalf("reload response = %#v, want closed session", response)
	}
	snapshot := state.Snapshot()
	if snapshot.Identity.Hostname != "edge-1" || !snapshot.Network.Interfaces[0].AdminUp {
		t.Fatalf("reset state = %#v", snapshot)
	}
}

func TestSessionCheckpointAndRollback(t *testing.T) {
	session, state := newSession()
	assertCommand(t, session, "enable", "")
	assertCommand(t, session, "checkpoint lesson-start", "[OK]")
	assertCommand(t, session, "configure terminal", "Enter configuration commands, one per line.")
	assertCommand(t, session, "hostname temporary", "")
	assertCommand(t, session, "end", "")
	assertCommand(t, session, "rollback checkpoint lesson-start", "[OK]")

	if got := state.Snapshot().Identity.Hostname; got != "edge-1" {
		t.Fatalf("hostname = %q, want edge-1", got)
	}
	assertCommand(t, session, "rollback checkpoint missing", "% Checkpoint not found")
}

func TestSessionVLANAndRouterModesMutateSharedState(t *testing.T) {
	session, state := newSession()
	assertCommand(t, session, "enable", "")
	assertCommand(t, session, "configure terminal", "Enter configuration commands, one per line.")
	assertCommand(t, session, "vlan 200", "")
	if session.Mode() != devicecli.ModeVLANConfig || session.Prompt() != "edge-1(config-vlan)#" {
		t.Fatalf("VLAN mode = %q prompt = %q", session.Mode(), session.Prompt())
	}
	assertCommand(t, session, "name USERS", "")
	assertCommand(t, session, "exit", "")
	assertCommand(t, session, "router ospf 10", "")
	if session.Mode() != devicecli.ModeRouterConfig || session.Prompt() != "edge-1(config-router)#" {
		t.Fatalf("router mode = %q prompt = %q", session.Mode(), session.Prompt())
	}
	assertCommand(t, session, "network 10.0.0.0 0.0.0.255 area 0", "")

	snapshot := state.Snapshot()
	if len(snapshot.Network.VLANs) != 1 || snapshot.Network.VLANs[0].Name != "USERS" {
		t.Fatalf("VLAN state = %#v", snapshot.Network.VLANs)
	}
	if len(snapshot.Network.Routers) != 1 || len(snapshot.Network.Routers[0].Networks) != 1 {
		t.Fatalf("router state = %#v", snapshot.Network.Routers)
	}
	assertCommand(t, session, "end", "")
	running := session.Execute("show running-config").Output
	for _, want := range []string{
		"vlan 200\n name USERS",
		"router ospf 10\n network 10.0.0.0 0.0.0.255 area 0",
	} {
		if !strings.Contains(running, want) {
			t.Fatalf("running config %q does not contain %q", running, want)
		}
	}
}

func TestSessionHelpAndCompletionFollowCurrentMode(t *testing.T) {
	session, _ := newSession()
	if output := session.Execute("?").Output; !strings.Contains(output, "enable") ||
		strings.Contains(output, "interface") {
		t.Fatalf("user help = %q", output)
	}
	assertCommand(t, session, "enable", "")
	assertCommand(t, session, "configure terminal", "Enter configuration commands, one per line.")
	completion := session.Complete("int")
	if len(completion) != 1 || completion[0] != "interface" {
		t.Fatalf("completion = %#v, want interface", completion)
	}
	assertCommand(t, session, "interface Gi0/1", "")
	if output := session.Execute("?").Output; !strings.Contains(output, "shutdown") {
		t.Fatalf("interface help = %q", output)
	}
}

func TestSessionInterfaceAddressAndDescriptionMutateSharedState(t *testing.T) {
	session, state := newSession()
	assertCommand(t, session, "enable", "")
	assertCommand(t, session, "configure terminal", "Enter configuration commands, one per line.")
	assertCommand(t, session, "interface Gi0/1", "")
	assertCommand(t, session, "description Branch uplink", "")
	assertCommand(t, session, "ip address 10.0.1.1/24", "")

	iface := state.Snapshot().Network.Interfaces[0]
	if iface.Description != "Branch uplink" || iface.Address != mustPrefix("10.0.1.1/24") {
		t.Fatalf("interface state = %#v", iface)
	}
	assertCommand(t, session, "no ip address", "")
	if state.Snapshot().Network.Interfaces[0].Address.IsValid() {
		t.Fatal("no ip address did not clear the interface address")
	}
}

func TestSessionStaticRouteMutatesSharedState(t *testing.T) {
	session, state := newSession()
	assertCommand(t, session, "enable", "")
	assertCommand(t, session, "configure terminal", "Enter configuration commands, one per line.")
	assertCommand(t, session, "ip route 10.2.0.0/16 10.0.0.2 Gi0/1", "")

	routes := state.Snapshot().Network.Routes
	if len(routes) != 2 || routes[1].Destination != mustPrefix("10.2.0.0/16") || routes[1].Via != "Gi0/1" {
		t.Fatalf("routes = %#v", routes)
	}
	assertCommand(t, session, "end", "")
	if output := session.Execute("show running-config").Output; !strings.Contains(
		output,
		"ip route 10.2.0.0/16 10.0.0.2 Gi0/1",
	) {
		t.Fatalf("running config = %q", output)
	}
}

func TestSessionShowsOrderedConfigurationEvents(t *testing.T) {
	session, _ := newSession()
	assertCommand(t, session, "enable", "")
	assertCommand(t, session, "configure terminal", "Enter configuration commands, one per line.")
	assertCommand(t, session, "hostname branch-1", "")
	assertCommand(t, session, "end", "")

	output := session.Execute("show configuration events").Output
	if !strings.Contains(output, "network.installed") || !strings.Contains(output, "identity.updated") ||
		!strings.Contains(output, "branch-1") {
		t.Fatalf("event output = %q", output)
	}
}

func TestSessionRejectsMalformedAndUnsupportedCommandsWithoutMutatingState(t *testing.T) {
	session, state := newSession()
	baseline := state.Snapshot()
	commands := []string{"show everything", "enable now"}
	for _, command := range commands {
		if output := session.Execute(command).Output; output != "% Invalid input detected" {
			t.Fatalf("%q output = %q", command, output)
		}
	}
	assertCommand(t, session, "enable", "")
	assertCommand(t, session, "configure terminal", "Enter configuration commands, one per line.")
	for _, command := range []string{
		"hostname -invalid", "vlan 4095", "router bgp 1",
		"ip route not-a-prefix 10.0.0.2 Gi0/1",
	} {
		if output := session.Execute(command).Output; output == "" {
			t.Fatalf("%q unexpectedly succeeded", command)
		}
	}
	if snapshot := state.Snapshot(); snapshot.Identity != baseline.Identity ||
		len(snapshot.Network.VLANs) != 0 || len(snapshot.Network.Routers) != 0 ||
		len(snapshot.Network.Routes) != len(baseline.Network.Routes) {
		t.Fatalf("rejected commands mutated state: %#v", snapshot)
	}
}

func TestConfigurationModesRejectTrailingTokensOnTransitions(t *testing.T) {
	tests := []struct {
		name  string
		enter []string
	}{
		{name: "global", enter: []string{"enable", "configure terminal"}},
		{name: "interface", enter: []string{"enable", "configure terminal", "interface Gi0/1"}},
		{name: "vlan", enter: []string{"enable", "configure terminal", "vlan 10"}},
		{name: "router", enter: []string{"enable", "configure terminal", "router ospf 1"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, command := range []string{"exit garbage", "end garbage"} {
				session, _ := newSession()
				for _, enter := range tt.enter {
					assertCommand(t, session, enter, expectedSetupOutput(enter))
				}
				mode := session.Mode()
				assertCommand(t, session, command, "% Invalid input detected")
				if session.Mode() != mode {
					t.Fatalf("%q changed mode from %q to %q", command, mode, session.Mode())
				}
			}
		})
	}
}

func TestInterfaceShutdownRemovesAndRestoresConnectedRouteFromCarrierState(t *testing.T) {
	session, state := newSession()
	for _, command := range []string{"enable", "configure terminal", "interface Gi0/1"} {
		assertCommand(t, session, command, expectedSetupOutput(command))
	}
	assertCommand(t, session, "shutdown", "")
	snapshot := state.Snapshot()
	if snapshot.Network.Interfaces[0].OperUp || len(snapshot.Network.Routes) != 0 {
		t.Fatalf("shutdown state = %#v", snapshot.Network)
	}
	assertCommand(t, session, "no shutdown", "")
	snapshot = state.Snapshot()
	if !snapshot.Network.Interfaces[0].OperUp || len(snapshot.Network.Routes) != 1 ||
		!snapshot.Network.Routes[0].Connected {
		t.Fatalf("recovered state = %#v", snapshot.Network)
	}
}

func TestRouterConfigurationCanonicalizesIdentifiers(t *testing.T) {
	session, state := newSession()
	for _, command := range []string{"enable", "configure terminal", "router ospf +1"} {
		assertCommand(t, session, command, expectedSetupOutput(command))
	}
	assertCommand(t, session, "network 10.0.0.0 0.0.0.255 area 0001", "")
	routers := state.Snapshot().Network.Routers
	if len(routers) != 1 || routers[0].ProcessID != "1" || routers[0].Networks[0].Area != "1" {
		t.Fatalf("routers = %#v", routers)
	}
	assertCommand(t, session, "network 10.1.0.0 0.0.0.255 area garbage", "% Invalid input detected")
	if len(state.Snapshot().Network.Routers[0].Networks) != 1 {
		t.Fatal("invalid OSPF area mutated state")
	}
}

func expectedSetupOutput(command string) string {
	if command == "configure terminal" {
		return "Enter configuration commands, one per line."
	}
	return ""
}

func newSession() (*devicecli.Session, *devicestate.Store) {
	state := devicestate.NewStore(devicestate.Identity{Hostname: "edge-1"})
	state.ReplaceNetwork(devicestate.Network{
		Interfaces: []devicestate.Interface{{
			Name: "Gi0/1", Address: mustPrefix("10.0.0.1/24"), AdminUp: true, OperUp: true,
		}},
		Routes: []devicestate.Route{{
			Destination: mustPrefix("10.0.0.0/24"), Via: "Gi0/1", Connected: true,
		}},
	})
	return devicecli.NewSession(state), state
}

func assertCommand(t *testing.T, session *devicecli.Session, command, want string) {
	t.Helper()
	if got := session.Execute(command).Output; got != want {
		t.Fatalf("%q output = %q, want %q", command, got, want)
	}
}
