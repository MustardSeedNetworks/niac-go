//go:build linux && integration

package wiretest_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gosnmp/gosnmp"

	"github.com/MustardSeedNetworks/niac-go/internal/api"
	"github.com/MustardSeedNetworks/niac-go/internal/daemon"
	"github.com/MustardSeedNetworks/niac-go/internal/fabric"
)

// G4: the routing table a current manager reads, on the wire.
//
// NIAC published only the deprecated ipRouteTable, so a consumer asking for
// routes the modern way saw a router with none. Serving RFC 4292's
// inetCidrRouteTable is only half the job: its INDEX carries the destination,
// prefix length and next hop, and an index a manager cannot parse is the same
// as no table. So this walks the table over UDP and rebuilds each route from
// its OID, then compares that against what the scenario authored.
const (
	inetRouteTarget    = "10.254.200.60"
	inetRouteCommunity = "route_demo"
	inetRouteTable     = ".1.3.6.1.2.1.4.24.7.1"
	inetRouteIfIndex   = inetRouteTable + ".7"
	// RFC 2096's table, which is the one seed's routing collector walks.
	cidrRouteTable   = ".1.3.6.1.2.1.4.24.4.1"
	cidrRouteNextHop = cidrRouteTable + ".4"

	inetRouteSweepTimeout = 60 * time.Second
)

// authoredRoute is one row as rebuilt from the wire.
type authoredRoute struct {
	destination string
	prefixLen   int
	nextHop     string
}

func startInetRouteScenario(t *testing.T) {
	t.Helper()
	requireWire(t)

	root := t.TempDir()
	configPath := filepath.Join(root, "inet-route.yaml")
	// Three routes with different shapes: a connected subnet (no next hop), a
	// static default, and a static to a remote network. A table that only ever
	// emitted one shape would pass a weaker fixture.
	//
	// The two gateways exist because the fabric refuses a route whose next hop
	// is not a configured peer -- an authored route that points nowhere is a
	// scenario defect, and the validator says so before the replay starts.
	//
	// Everything sits on the transit /24 the test end of the veth carries
	// (wire_linux_test.go: clientCIDR), above the addresses the other suites
	// use, so the kernel can reach the agent and nothing collides.
	body := fmt.Sprintf(`networks:
  - name: route-lan
    subnet: 10.254.200.0/24
attachments:
  - name: tester
    connect: route-lan
devices:
  - name: ROUTE-R1
    type: router
    mac: "02:00:00:00:c0:01"
    snmp_agent:
      enabled: true
      community: %s
      sysname: ROUTE-R1
    interfaces:
      - name: Ethernet1
        type: ethernet
        network: route-lan
        address: %s/24
    routes:
      - destination: 0.0.0.0/0
        via: Ethernet1
        next_hop: 10.254.200.61
      - destination: 192.0.2.0/24
        via: Ethernet1
        next_hop: 10.254.200.62
  - name: ROUTE-GW1
    type: router
    mac: "02:00:00:00:c0:fe"
    interfaces:
      - name: Ethernet1
        type: ethernet
        network: route-lan
        address: 10.254.200.61/24
  - name: ROUTE-GW2
    type: router
    mac: "02:00:00:00:c0:fd"
    interfaces:
      - name: Ethernet1
        type: ethernet
        network: route-lan
        address: 10.254.200.62/24
`, inetRouteCommunity, inetRouteTarget)
	if err := os.WriteFile(configPath, []byte(body), 0o600); err != nil {
		t.Fatalf("write the scenario: %v", err)
	}
	t.Setenv("NIAC_CONFIGS_DIR", root)

	d, err := daemon.NewDaemon(daemon.Config{
		StoragePath: "disabled",
		AttachmentPolicies: []fabric.PhysicalAttachmentPolicy{{
			Interface: simIface, Mode: fabric.ModeAccess, AccessVLAN: accessVLAN,
		}},
	})
	if err != nil {
		t.Fatalf("daemon.NewDaemon: %v", err)
	}
	if startErr := d.StartSimulation(api.SimulationRequest{
		SessionID: "wiretest-inet-route", Interface: simIface, Attachment: "tester",
		AttachmentMode: fabric.ModeAccess, AccessVLAN: accessVLAN, ConfigPath: configPath,
	}); startErr != nil {
		t.Fatalf("StartSimulation on %s: %v", simIface, startErr)
	}
	t.Cleanup(func() {
		if stopErr := d.StopSimulation(""); stopErr != nil {
			t.Errorf("StopSimulation: %v", stopErr)
		}
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = d.Shutdown(ctx)
	})
}

func TestInetCidrRouteTableIsWalkableAndMatchesTheAuthoredRoutes(t *testing.T) {
	startInetRouteScenario(t)
	client := dialInetRouteHost(t)

	rows := walkColumn(t, client, inetRouteIfIndex)
	if len(rows) == 0 {
		t.Fatal("inetCidrRouteTable returned no rows; a modern manager sees no routes")
	}

	got := make(map[authoredRoute]int, len(rows))
	for _, pdu := range rows {
		route := parseInetRouteIndex(t, pdu.Name)
		got[route] = gosnmp.ToBigInt(pdu.Value).Sign()
	}

	// The two static routes as authored, and the connected route the
	// interface address implies. A connected route has no next hop, which
	// RFC 4292 spells as the unspecified address.
	for _, want := range []authoredRoute{
		{destination: "0.0.0.0", prefixLen: 0, nextHop: "10.254.200.61"},
		{destination: "192.0.2.0", prefixLen: 24, nextHop: "10.254.200.62"},
		{destination: "10.254.200.0", prefixLen: 24, nextHop: "0.0.0.0"},
	} {
		if _, ok := got[want]; !ok {
			t.Errorf("no row for %s/%d via %s; rebuilt rows: %v",
				want.destination, want.prefixLen, want.nextHop, got)
		}
	}
}

// parseInetRouteIndex rebuilds a route from the OID a manager received. This
// is the assertion that matters: if the index were malformed the values would
// still arrive and mean nothing.
//
// The index after the column is destType, destLen, dest…, pfxLen, policyLen,
// policy…, hopType, hopLen, hop… — every part of it load-bearing.
func parseInetRouteIndex(t *testing.T, oid string) authoredRoute {
	t.Helper()
	suffix := strings.TrimPrefix(oid, inetRouteIfIndex+".")
	if suffix == oid {
		t.Fatalf("%s is not under %s", oid, inetRouteIfIndex)
	}
	parts := strings.Split(suffix, ".")

	take := func(at int) (string, int) {
		t.Helper()
		if at >= len(parts) {
			t.Fatalf("index %s ends before an address", suffix)
		}
		length, err := strconv.Atoi(parts[at])
		if err != nil || at+length >= len(parts)+1 {
			t.Fatalf("index %s has a bad address length at %d", suffix, at)
		}
		return strings.Join(parts[at+1:at+1+length], "."), at + 1 + length
	}

	// destType is one sub-identifier, then the destination.
	destination, next := take(1)
	prefixLen, err := strconv.Atoi(parts[next])
	if err != nil {
		t.Fatalf("index %s has a bad prefix length", suffix)
	}
	// The policy OID, then the next-hop type, then the next hop.
	policyLen, err := strconv.Atoi(parts[next+1])
	if err != nil {
		t.Fatalf("index %s has a bad policy length", suffix)
	}
	nextHop, _ := take(next + 1 + policyLen + 2)

	return authoredRoute{destination: destination, prefixLen: prefixLen, nextHop: nextHop}
}

func dialInetRouteHost(t *testing.T) *gosnmp.GoSNMP {
	t.Helper()
	client := &gosnmp.GoSNMP{
		Target: inetRouteTarget, Port: 161, Community: inetRouteCommunity,
		Version: gosnmp.Version2c, Timeout: 5 * time.Second, Retries: 3,
		MaxRepetitions: 20,
	}
	if err := client.Connect(); err != nil {
		t.Fatalf("connect to %s: %v", inetRouteTarget, err)
	}
	t.Cleanup(func() { _ = client.Conn.Close() })

	return client
}

// walkColumn sweeps one column, because every row carries the whole
// index and one column is enough to enumerate the table.
func walkColumn(t *testing.T, client *gosnmp.GoSNMP, root string) []gosnmp.SnmpPDU {
	t.Helper()
	done := make(chan struct{})
	var results []gosnmp.SnmpPDU
	var err error
	go func() {
		defer close(done)
		results, err = client.BulkWalkAll(root)
	}()
	select {
	case <-done:
	case <-time.After(inetRouteSweepTimeout):
		t.Fatalf("walking %s did not finish within %s", root, inetRouteSweepTimeout)
	}
	if err != nil {
		t.Fatalf("walking %s: %v", root, err)
	}

	return results
}

// The RFC 2096 table on the wire. NIAC served .21 and .24.7 and nothing at
// .24.4, which is where seed's routing collector looks -- so a replayed router
// reported no routes to the one consumer that matters, and no test on either
// side could see it.
//
// This asserts the next hop *column*, because that is the field path analysis
// needs and the one a mis-shaped index drops silently: seed answers a wrong
// index by discarding the row without an error.
func TestIPCidrRouteTableServesTheConsumersTableOnTheWire(t *testing.T) {
	startInetRouteScenario(t)
	client := dialInetRouteHost(t)

	rows := walkColumn(t, client, cidrRouteNextHop)
	if len(rows) == 0 {
		t.Fatal("ipCidrRouteTable returned no rows; seed's collector sees no routes")
	}

	nextHops := make(map[string]bool, len(rows))
	for _, pdu := range rows {
		// Thirteen index fields is what seed's parseRouteOID requires.
		index := strings.TrimPrefix(pdu.Name, cidrRouteNextHop+".")
		if fields := len(strings.Split(index, ".")); fields != 13 {
			t.Errorf("%s has %d index fields, want 13", pdu.Name, fields)
		}
		if address, ok := pdu.Value.(string); ok {
			nextHops[address] = true
		}
	}
	for _, want := range []string{"10.254.200.61", "10.254.200.62", "0.0.0.0"} {
		if !nextHops[want] {
			t.Errorf("no route with next hop %s; got %v", want, nextHops)
		}
	}
}
