package devicestate_test

import (
	"net/netip"
	"reflect"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
)

func prefixStore() *devicestate.Store {
	store := devicestate.NewStore(devicestate.Identity{})
	store.ReplaceNetwork(devicestate.Network{
		Interfaces: []devicestate.Interface{
			{Name: "eth0", Address: netip.MustParsePrefix("192.0.2.10/24"), AdminUp: true, OperUp: true},
			{Name: "eth1", Address: netip.MustParsePrefix("198.51.100.10/24"), AdminUp: true, OperUp: true},
		},
		Routes: []devicestate.Route{
			{Destination: netip.MustParsePrefix("192.0.2.0/24"), Via: "eth0", Connected: true},
			{Destination: netip.MustParsePrefix("198.51.100.0/24"), Via: "eth1", Connected: true},
			{Destination: netip.MustParsePrefix("0.0.0.0/0"), Via: "eth0", NextHop: netip.MustParseAddr("192.0.2.1")},
		},
	})
	return store
}

func TestPrefixFaultProjection(t *testing.T) {
	for _, bits := range []int{0, 16, 30, 31, 32} {
		t.Run(netip.PrefixFrom(netip.MustParseAddr("192.0.2.10"), bits).String(), func(t *testing.T) {
			store := prefixStore()
			before := store.Snapshot()
			fault := devicestate.InterfacePrefixFault{
				Interface:  "eth0",
				Type:       devicestate.FaultBadMask,
				PrefixBits: bits,
			}
			if err := store.SetInterfacePrefixFault(fault); err != nil {
				t.Fatal(err)
			}
			snapshot := store.Snapshot()
			effective := snapshot.EffectiveHostNetwork()
			want := netip.PrefixFrom(before.Network.Interfaces[0].Address.Addr(), bits)
			if effective.Interfaces[0].Address != want ||
				!reflect.DeepEqual(effective.Interfaces[1], before.Network.Interfaces[1]) {
				t.Fatalf("incorrect effective interfaces: %+v", effective.Interfaces)
			}
			if !reflect.DeepEqual(snapshot.Network, before.Network) {
				t.Fatal("canonical network changed")
			}
			assertPrefixRoutes(t, effective.Routes, before.Network.Routes, want)
			effective.Interfaces[0].Name = "mutated"
			if snapshot.Network.Interfaces[0].Name != "eth0" {
				t.Fatal("projection aliases snapshot")
			}
			if err := store.ClearInterfacePrefixFault("eth0", devicestate.FaultBadMask); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(store.Snapshot().EffectiveHostNetwork(), before.Network) {
				t.Fatal("clear did not restore baseline")
			}
		})
	}
}

func assertPrefixRoutes(t *testing.T, got, before []devicestate.Route, prefix netip.Prefix) {
	t.Helper()
	want := []devicestate.Route{before[1], before[2], {
		Destination: prefix.Masked(), Via: "eth0", Connected: true,
	}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("routes = %+v, want %+v", got, want)
	}
}

func TestPrefixFaultCanonicalEditAndCheckpoint(t *testing.T) {
	store := prefixStore()
	fault := devicestate.InterfacePrefixFault{Interface: "eth0", Type: devicestate.FaultBadMask, PrefixBits: 16}
	if err := store.SetInterfacePrefixFault(fault); err != nil {
		t.Fatal(err)
	}
	store.SaveCheckpoint("armed")
	if err := store.UpdateInterface("eth0", func(iface devicestate.Interface) (devicestate.Interface, error) {
		iface.Address = netip.MustParsePrefix("203.0.113.10/24")
		return iface, nil
	}); err != nil {
		t.Fatal(err)
	}
	if got := store.Snapshot().EffectiveHostNetwork().Interfaces[0].Address.String(); got != "203.0.113.10/16" {
		t.Fatal(got)
	}
	state := store.ExportState()
	store.ClearAllFaults()
	if err := store.RestoreState(state); err != nil {
		t.Fatal(err)
	}
	if got := store.Snapshot().EffectiveHostNetwork().Interfaces[0].Address.String(); got != "203.0.113.10/16" {
		t.Fatal(got)
	}
	fault.PrefixBits = 32
	if err := store.SetInterfacePrefixFault(fault); err != nil {
		t.Fatal(err)
	}
	if err := store.RestoreCheckpoint("armed"); err != nil {
		t.Fatal(err)
	}
	if got := store.Snapshot().EffectiveHostNetwork().Interfaces[0].Address.String(); got != "192.0.2.10/16" {
		t.Fatal(got)
	}
}
