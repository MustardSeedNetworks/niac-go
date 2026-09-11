//go:build linux && integration

package wiretest_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gosnmp/gosnmp"

	"github.com/MustardSeedNetworks/niac-go/internal/api"
	"github.com/MustardSeedNetworks/niac-go/internal/daemon"
	"github.com/MustardSeedNetworks/niac-go/internal/fabric"
)

// P5-2: the routing MIBs a consumer reads off a router, on the wire.
//
// The starter set answered noSuchObject for BGP4-MIB and OSPF-MIB until three
// captures carrying them landed, and a file in the tree is not the same claim
// as an agent answering for it: the walk loader drops rows it cannot resolve,
// and the contract classifier overwrites others. This walks both subtrees over
// UDP from the far end of the veth and requires real rows in each.
const (
	routingWalk      = "cisco-c3750-02-r64.walk"
	routingTarget    = "10.254.200.3"
	routingCommunity = "wire_routing"

	bgpPeerTableOID = ".1.3.6.1.2.1.15.3.1"
	ospfMIBOID      = ".1.3.6.1.2.1.14"
	// bgpPeerState (bgpPeerTable column 2). A peer row without a state is a
	// table NIAC built rather than one the capture holds.
	bgpPeerStateOID = ".1.3.6.1.2.1.15.3.1.2"

	// internal/library/starter/walks/cisco-c3750-02-r64.walk: 72 bgpPeerTable
	// rows across 3 peers, and 233 OSPF rows. Asserting the exact counts would
	// fail on a re-capture that is still correct, so these are floors that a
	// silently emptied subtree cannot clear.
	wantBGPPeerRows = 24
	wantOSPFRows    = 50
	wantBGPPeers    = 3

	routingSweepTimeout = 2 * time.Minute
)

func startRoutingCapture(t *testing.T) {
	t.Helper()
	requireWire(t)

	root := t.TempDir()
	networks := filepath.Join(root, "networks")
	walks := filepath.Join(root, "walks")
	for _, dir := range []string{networks, walks} {
		if err := os.MkdirAll(dir, 0o750); err != nil {
			t.Fatalf("create %s: %v", dir, err)
		}
	}
	configPath := filepath.Join(networks, "router-wire.yaml")
	copyFile(t, filepath.Join("testdata", "router-wire.yaml"), configPath)
	copyFile(t,
		filepath.Join("..", "library", "starter", "walks", routingWalk),
		filepath.Join(walks, routingWalk))

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
		SessionID:      "wiretest-routing",
		Interface:      simIface,
		Attachment:     "tester",
		AttachmentMode: fabric.ModeAccess,
		AccessVLAN:     accessVLAN,
		ConfigPath:     configPath,
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

func TestRouterWalkServesBGPAndOSPFOnTheWire(t *testing.T) {
	startRoutingCapture(t)
	client := dialRoutingHost(t)

	bgp := walkSubtree(t, client, bgpPeerTableOID)
	if len(bgp) < wantBGPPeerRows {
		t.Errorf("bgpPeerTable returned %d rows, want at least %d", len(bgp), wantBGPPeerRows)
	}
	peers := 0
	for _, pdu := range bgp {
		if strings.HasPrefix(pdu.Name, bgpPeerStateOID+".") {
			peers++
		}
	}
	if peers < wantBGPPeers {
		t.Errorf("bgpPeerState returned %d peers, want at least %d", peers, wantBGPPeers)
	}

	ospf := walkSubtree(t, client, ospfMIBOID)
	if len(ospf) < wantOSPFRows {
		t.Errorf("OSPF-MIB returned %d rows, want at least %d", len(ospf), wantOSPFRows)
	}
}

func dialRoutingHost(t *testing.T) *gosnmp.GoSNMP {
	t.Helper()
	client := &gosnmp.GoSNMP{
		Target: routingTarget, Port: 161, Community: routingCommunity,
		Version: gosnmp.Version2c, Timeout: 5 * time.Second, Retries: 3,
		MaxRepetitions: 20,
	}
	if err := client.Connect(); err != nil {
		t.Fatalf("connect to %s: %v", routingTarget, err)
	}
	t.Cleanup(func() { _ = client.Conn.Close() })

	return client
}

func walkSubtree(t *testing.T, client *gosnmp.GoSNMP, root string) []gosnmp.SnmpPDU {
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
	case <-time.After(routingSweepTimeout):
		t.Fatalf("walking %s did not finish within %s", root, routingSweepTimeout)
	}
	if err != nil {
		t.Fatalf("walking %s: %v", root, err)
	}

	return results
}
