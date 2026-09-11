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
	inetRouteTarget    = "10.254.201.1"
	inetRouteCommunity = "route_demo"
	inetRouteTable     = ".1.3.6.1.2.1.4.24.7.1"
	inetRouteIfIndex   = inetRouteTable + ".7"

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
	body := fmt.Sprintf(`networks:
  - name: route-lan
    subnet: 10.254.201.0/24
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
        next_hop: 10.254.201.254
      - destination: 192.0.2.0/24
        via: Ethernet1
        next_hop: 10.254.201.253
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

	rows := walkInetRouteColumn(t, client)
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
		{destination: "0.0.0.0", prefixLen: 0, nextHop: "10.254.201.254"},
		{destination: "192.0.2.0", prefixLen: 24, nextHop: "10.254.201.253"},
		{destination: "10.254.201.0", prefixLen: 24, nextHop: "0.0.0.0"},
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
// The index after the column is
// destType, destLen, dest…, pfxLen, policyLen, policy…, hopType, hopLen, hop…
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

// walkInetRouteColumn sweeps one column, because every row carries the whole
// index and one column is enough to enumerate the table.
func walkInetRouteColumn(t *testing.T, client *gosnmp.GoSNMP) []gosnmp.SnmpPDU {
	t.Helper()
	done := make(chan struct{})
	var results []gosnmp.SnmpPDU
	var err error
	go func() {
		defer close(done)
		results, err = client.BulkWalkAll(inetRouteIfIndex)
	}()
	select {
	case <-done:
	case <-time.After(inetRouteSweepTimeout):
		t.Fatalf("walking %s did not finish within %s", inetRouteIfIndex, inetRouteSweepTimeout)
	}
	if err != nil {
		t.Fatalf("walking %s: %v", inetRouteIfIndex, err)
	}

	return results
}
