package library_test

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/library"
)

// Routing MIB coverage in the starter walks (plan row P5-2).
//
// A consumer that discovers routers reads BGP4-MIB and OSPF-MIB, and until
// these walks landed the whole starter set answered noSuchObject for both: a
// tester pointed at NIAC saw a network with no routing at all. Three walks
// carry the content, from three different devices, so one being re-captured
// or dropped cannot silently take the coverage to zero.
const (
	bgp4MIB = ".1.3.6.1.2.1.15."
	ospfMIB = ".1.3.6.1.2.1.14."
	// bgpPeerTable. The scalars above it (bgpVersion, bgpLocalAs,
	// bgpIdentifier) are present on a device with BGP compiled in and no
	// session, so a walk carrying only those is not routing content.
	bgpPeerTable = ".1.3.6.1.2.1.15.3.1."

	wantRoutingWalks = 3
)

func TestStarterWalksCarryRoutingMIBs(t *testing.T) {
	root := t.TempDir()
	if _, err := library.Open(root); err != nil {
		t.Fatalf("bootstrap library: %v", err)
	}
	walksDir := filepath.Join(root, string(library.KindWalks))
	entries, err := os.ReadDir(walksDir)
	if err != nil {
		t.Fatalf("read the bootstrapped walks: %v", err)
	}

	var withPeers, withOSPF []string
	for _, entry := range entries {
		if filepath.Ext(entry.Name()) != ".walk" {
			continue
		}
		counts := countPrefixes(t, filepath.Join(walksDir, entry.Name()),
			bgpPeerTable, ospfMIB)
		if counts[bgpPeerTable] > 0 {
			withPeers = append(withPeers, entry.Name())
		}
		if counts[ospfMIB] > 0 {
			withOSPF = append(withOSPF, entry.Name())
		}
	}

	if len(withPeers) < wantRoutingWalks {
		t.Errorf("%d starter walk(s) carry a BGP peer table, want at least %d: %v",
			len(withPeers), wantRoutingWalks, withPeers)
	}
	if len(withOSPF) < wantRoutingWalks {
		t.Errorf("%d starter walk(s) carry OSPF-MIB rows, want at least %d: %v",
			len(withOSPF), wantRoutingWalks, withOSPF)
	}
}

// countPrefixes reads one walk once and counts rows under each prefix; the
// walks are megabytes, so a pass per prefix would be the slowest thing in the
// package.
func countPrefixes(t *testing.T, path string, prefixes ...string) map[string]int {
	t.Helper()
	file, err := os.Open(path)
	if err != nil {
		t.Fatalf("open %s: %v", path, err)
	}
	defer file.Close()

	counts := make(map[string]int, len(prefixes))
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		for _, prefix := range prefixes {
			if strings.HasPrefix(line, prefix) {
				counts[prefix]++
			}
		}
	}
	if err = scanner.Err(); err != nil {
		t.Fatalf("read %s: %v", path, err)
	}

	return counts
}
