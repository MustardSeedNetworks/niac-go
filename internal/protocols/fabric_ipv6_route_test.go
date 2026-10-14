package protocols

import (
	"bytes"
	"net"
	"net/netip"
	"slices"
	"testing"

	"github.com/gopacket/gopacket/layers"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
	"github.com/MustardSeedNetworks/niac-go/internal/fabric"
	"github.com/MustardSeedNetworks/niac-go/internal/logging"
)

// An IPv6 network no attachment-router interface numbers answers through an
// IPv6 static route on that router (niac-go#2411): here the core routes the
// remote /64 to the neighbour switch over med-mgmt.

const (
	v6NeighbourMgmt   = "fd00:51:c8::22"
	v6NeighbourRemote = "fd00:51:dc::22"
	v6RemotePrefix    = "fd00:51:dc::/64"
)

func dualStackRoutedStack(t *testing.T, routes ...config.Route) *Stack {
	t.Helper()
	cfg := dualStackPlacementConfig()
	cfg.Networks = append(cfg.Networks, config.Network{
		Name: "med-remote", Subnet: "10.51.220.0/24", SubnetV6: v6RemotePrefix, VirtualVLAN: 220,
	})
	for index := range cfg.Devices {
		device := &cfg.Devices[index]
		switch device.Name {
		case placementCore:
			device.Routes = append(device.Routes, routes...)
		case placementNeighbour:
			setAddressV6(device, "Vlan200", v6NeighbourMgmt+"/64")
			device.Interfaces = append(device.Interfaces, config.Interface{
				Name: "Vlan220", Network: "med-remote",
				Address: "10.51.220.22/24", AddressV6: v6NeighbourRemote + "/64",
			})
		}
	}
	stack := NewStackWithTransport(idleTransport{}, cfg, logging.NewDebugConfig(0))
	stack.ConfigureFabric(compilePlacement(t, cfg))
	return stack
}

func remoteRoute(destination string) config.Route {
	return config.Route{Destination: destination, Via: "Vlan200", NextHop: v6NeighbourMgmt}
}

func TestFabricIPv6EchoFollowsTheAttachmentRoutersStaticRoutes(t *testing.T) {
	cases := []struct {
		name   string
		routes []config.Route
		answer bool
	}{
		{"remote prefix route", []config.Route{remoteRoute(v6RemotePrefix)}, true},
		{"default route", []config.Route{remoteRoute("::/0")}, true},
		{"route to another prefix", []config.Route{remoteRoute("fd00:51:dd::/64")}, false},
		{"no route", nil, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			stack := dualStackRoutedStack(t, tc.routes...)
			core := stackDevice(t, stack, placementCore).MACAddress
			pkt := icmpv6EchoFrame(t, placementClient(1), core, v6Client, v6NeighbourRemote)

			stack.decodePacket(pkt)

			got := queuedICMPv6(t, stack, layers.LayerTypeICMPv6Echo)
			if !tc.answer {
				assertIPv6EchoDropped(t, stack, pkt, got, fabricRejectionNoRoute)
				return
			}
			if len(got) != 1 || !bytes.Equal(got[0].src, core) || !got[0].srcIP.Equal(net.ParseIP(v6NeighbourRemote)) {
				t.Fatalf(
					"echo to %s answered %v, want one reply from %s at %s",
					v6NeighbourRemote,
					got,
					v6NeighbourRemote,
					core,
				)
			}
			if forwarded := stack.GetStats().FabricForwarded; forwarded != 1 {
				t.Fatalf("FabricForwarded = %d, want 1", forwarded)
			}
		})
	}
}

func TestFabricIPv6StaticRouteNeedsBothEndsUp(t *testing.T) {
	for _, shut := range []string{placementCore, placementNeighbour} {
		t.Run(shut, func(t *testing.T) {
			stack := dualStackRoutedStack(t, remoteRoute(v6RemotePrefix))
			state := stack.deviceStates[stackDevice(t, stack, shut)]
			err := state.UpdateInterface("Vlan200", func(iface devicestate.Interface) (devicestate.Interface, error) {
				iface.AdminUp, iface.OperUp = false, false
				return iface, nil
			})
			if err != nil {
				t.Fatalf("shut %s Vlan200: %v", shut, err)
			}

			stack.decodePacket(icmpv6EchoFrame(
				t, placementClient(1), stackDevice(t, stack, placementCore).MACAddress, v6Client, v6NeighbourRemote,
			))

			if got := queuedICMPv6(t, stack, layers.LayerTypeICMPv6Echo); len(got) != 0 {
				t.Fatalf("echo over a static route with %s Vlan200 shut answered %v", shut, got)
			}
		})
	}
}

// The device route table and its CLI are IPv4; the IPv6 route stays in the
// compiled topology the session reports.
func TestFabricIPv6StaticRouteIsReportedOutsideTheIPv4RouteTable(t *testing.T) {
	stack := dualStackRoutedStack(t, remoteRoute(v6RemotePrefix))
	want := fabric.Route{
		Device: placementCore, Destination: netip.MustParsePrefix(v6RemotePrefix),
		Via: "Vlan200", NextHop: netip.MustParseAddr(v6NeighbourMgmt),
	}

	topology, ok := stack.RuntimeFabricTopology()
	if !ok {
		t.Fatal("RuntimeFabricTopology() ok = false")
	}
	if !slices.Contains(topology.Routes, want) {
		t.Fatalf("RuntimeFabricTopology().Routes = %#v, want %#v", topology.Routes, want)
	}

	for _, route := range stack.deviceStates[stackDevice(t, stack, placementCore)].Snapshot().Network.Routes {
		if route.Destination.Addr().Is6() {
			t.Fatalf("device route table carries IPv6 route %#v", route)
		}
	}
}
