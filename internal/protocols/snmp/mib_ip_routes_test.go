package snmp

import (
	"net"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/gosnmp/gosnmp"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
)

func TestConfiguredRouteUsesWalkInterfaceIndex(t *testing.T) {
	device := createTestDevice()
	device.Type = "router"
	device.Interfaces = []config.Interface{{
		Name: "GigabitEthernet1/0/48", Network: "transit", Address: "10.254.200.1/24",
	}}
	device.Routes = []config.Route{{
		Destination: "10.240.0.0/16", Via: "GigabitEthernet1/0/48", NextHop: "10.254.200.2",
	}}
	agent := NewAgent(device, 0)

	if err := agent.SetOID(ifName+".48", &OIDValue{
		Type: gosnmp.OctetString, Value: "GigabitEthernet1/0/48",
	}); err != nil {
		t.Fatalf("SetOID(): %v", err)
	}
	agent.registerConfiguredRoutes(device)

	entry := agent.mib.Get(ipRouteIfIndex + ".10.240.0.0")
	if entry == nil || entry.Value != 48 {
		t.Fatalf("ipRouteIfIndex = %#v, want 48", entry)
	}
	mask := agent.mib.Get(ipRouteMask + ".10.240.0.0")
	if mask == nil || !net.ParseIP(mask.Value.(string)).Equal(net.IPv4(255, 255, 0, 0)) {
		t.Fatalf("ipRouteMask = %#v, want 255.255.0.0", mask)
	}
	nextHop := agent.mib.Get(ipRouteNextHop + ".10.240.0.0")
	if nextHop == nil || nextHop.Value != "10.254.200.2" {
		t.Fatalf("ipRouteNextHop = %#v, want 10.254.200.2", nextHop)
	}
}

// ipRouteTable is IPv4: an IPv6 static route writes no row under its address.
func TestConfiguredIPv6RouteHasNoIPRouteTableRow(t *testing.T) {
	device := createTestDevice()
	device.Type = "router"
	device.Interfaces = []config.Interface{{
		Name: "GigabitEthernet1/0/48", Network: "transit", Address: "10.254.200.1/24",
		AddressV6: "fd00:254:200::1/64",
	}}
	device.Routes = []config.Route{{
		Destination: "fd00:240::/48", Via: "GigabitEthernet1/0/48", NextHop: "fd00:254:200::2",
	}}
	agent := NewAgent(device, 0)
	if err := agent.SetOID(ifName+".48", &OIDValue{
		Type: gosnmp.OctetString, Value: "GigabitEthernet1/0/48",
	}); err != nil {
		t.Fatalf("SetOID(): %v", err)
	}

	agent.registerConfiguredRoutes(device)

	if entry := agent.mib.Get(ipRouteDest + ".fd00:240::"); entry != nil {
		t.Fatalf("ipRouteDest.fd00:240:: = %#v, want no row", entry)
	}
}

// A replayed router keeps the routes its capture recorded unless the scenario
// authors its own; it must never answer with an empty routing table (#2684).
func TestWalkRouteTables(t *testing.T) {
	const walk = `.1.3.6.1.2.1.2.2.1.2.500 = STRING: "Vl500"
.1.3.6.1.2.1.31.1.1.1.1.500 = STRING: "Vl500"
.1.3.6.1.2.1.4.3.0 = Counter32: 123456
.1.3.6.1.2.1.4.20.1.1.10.0.38.1 = IpAddress: 10.0.38.1
.1.3.6.1.2.1.4.21.1.1.10.0.115.0 = IpAddress: 10.0.115.0
.1.3.6.1.2.1.4.21.1.7.10.0.115.0 = IpAddress: 10.0.38.2
.1.3.6.1.2.1.4.22.1.2.500.10.0.38.2 = Hex-STRING: 00 0C CE 88 23 C8
`
	cases := []struct {
		name       string
		author     func(*config.Device)
		wantKept   []string
		wantAbsent []string
		wantRoute  string
	}{
		{
			name:   "no authored addresses or routes keeps the capture's tables",
			author: func(*config.Device) {},
			wantKept: []string{
				"1.3.6.1.2.1.4.20.1.1.10.0.38.1",
				"1.3.6.1.2.1.4.21.1.1.10.0.115.0",
				"1.3.6.1.2.1.4.21.1.7.10.0.115.0",
				"1.3.6.1.2.1.4.22.1.2.500.10.0.38.2",
			},
		},
		{
			name: "authored routes replace them",
			author: func(device *config.Device) {
				device.Interfaces = []config.Interface{{Name: "Vl500", Network: "transit", Address: "10.254.200.1/24"}}
				device.Routes = []config.Route{{Destination: "10.240.0.0/16", Via: "Vl500", NextHop: "10.254.200.2"}}
			},
			wantAbsent: []string{
				"1.3.6.1.2.1.4.20.1.1.10.0.38.1",
				"1.3.6.1.2.1.4.21.1.1.10.0.115.0",
				"1.3.6.1.2.1.4.22.1.2.500.10.0.38.2",
			},
			wantRoute: "10.240.0.0",
		},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			agent, captured := loadRouterWalk(t, walk, testCase.author)
			assertCapturedRows(t, agent, captured, testCase.wantKept)
			assertReplacedRows(t, agent, testCase.wantAbsent)
			if testCase.wantRoute != "" {
				if _, err := agent.HandleGet(ipRouteDest + "." + testCase.wantRoute); err != nil {
					t.Errorf("authored route %s: %v", testCase.wantRoute, err)
				}
			}
			if got, err := agent.HandleGet(ipInReceives); err != nil || got.Value == uint32(123456) {
				t.Errorf("ipInReceives = %v (%v), want this run's counter, not the capture's", got, err)
			}
		})
	}
}

func loadRouterWalk(t *testing.T, walk string, author func(*config.Device)) (*Agent, map[string]*OIDValue) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "router.walk")
	if err := os.WriteFile(path, []byte(walk), 0o600); err != nil {
		t.Fatal(err)
	}
	source, err := ParseWalkFile(path)
	if err != nil {
		t.Fatal(err)
	}
	captured := map[string]*OIDValue{}
	for _, entry := range source {
		captured[strings.TrimPrefix(entry.OID, ".")] = &OIDValue{Type: entry.Type, Value: entry.Value}
	}
	device := createTestDevice()
	author(device)
	agent := NewAgent(device, 0)
	if loadErr := agent.LoadWalkFile(path); loadErr != nil {
		t.Fatal(loadErr)
	}
	return agent, captured
}

func assertCapturedRows(t *testing.T, agent *Agent, captured map[string]*OIDValue, oids []string) {
	t.Helper()
	for _, oid := range oids {
		got, err := agent.HandleGet(oid)
		if err != nil {
			t.Fatalf("HandleGet(%s): %v", oid, err)
		}
		if want := captured[oid]; got.Type != want.Type || !reflect.DeepEqual(got.Value, want.Value) {
			t.Errorf("%s = %v %v, want the capture's %v %v", oid, got.Type, got.Value, want.Type, want.Value)
		}
	}
}

func assertReplacedRows(t *testing.T, agent *Agent, oids []string) {
	t.Helper()
	for _, oid := range oids {
		if got, err := agent.HandleGet(oid); err == nil {
			t.Errorf("%s = %v, want the capture's row replaced", oid, got.Value)
		}
	}
}
