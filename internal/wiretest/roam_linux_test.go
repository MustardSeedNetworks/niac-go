//go:build linux && integration

package wiretest_test

import (
	"net"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/MustardSeedNetworks/niac-go/internal/api"
	"github.com/MustardSeedNetworks/niac-go/internal/daemon"
	"github.com/MustardSeedNetworks/niac-go/internal/fabric"
)

// TestAuthoredRoamEmitsSyslogFromBothAPsOnTheWire: a station roaming from one
// AP to another and back is reported by each AP for its own half -- the AP it
// joined logs the association, the AP it left logs the roam -- over the wire,
// from each AP's own address.
func TestAuthoredRoamEmitsSyslogFromBothAPsOnTheWire(t *testing.T) {
	requireWire(t)
	collector, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.ParseIP(faultClientAddr), Port: 514})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = collector.Close() })
	data, err := os.ReadFile(filepath.Join("testdata", "roam-timeline.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("NIAC_CONFIGS_DIR", t.TempDir())
	d, err := daemon.NewDaemon(daemon.Config{
		StoragePath: "disabled",
		AttachmentPolicies: []fabric.PhysicalAttachmentPolicy{{
			Interface: simIface, Mode: fabric.ModeAccess, AccessVLAN: accessVLAN,
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { shutdownSyslogDaemon(t, d) })
	if err = d.StartSimulation(api.SimulationRequest{
		SessionID: "wiretest-roam", Interface: simIface, Attachment: "tester",
		AttachmentMode: fabric.ModeAccess, AccessVLAN: accessVLAN, ConfigData: string(data),
	}); err != nil {
		t.Fatal(err)
	}
	if err = collector.SetReadDeadline(time.Now().Add(20 * time.Second)); err != nil {
		t.Fatal(err)
	}

	// The two APs dispatch independently, so only each AP's own order is fixed.
	got := map[string][]string{}
	buffer := make([]byte, 2048)
	for range 4 {
		count, source, readErr := collector.ReadFromUDP(buffer)
		if readErr != nil {
			t.Fatalf("receive roam syslog (have %v): %v", got, readErr)
		}
		fields := strings.Fields(string(buffer[:count]))
		if len(fields) != 10 || fields[0] != "<134>1" || fields[9] != `target="02:c0:17:a4:03:6b"` {
			t.Fatalf("unexpected syslog from %s: %q", source, buffer[:count])
		}
		got[fields[2]+"@"+source.IP.String()] = append(got[fields[2]+"@"+source.IP.String()], fields[5])
	}

	want := map[string][]string{
		"roam-ap-01@10.254.200.11": {"STATION_ROAMED", "STATION_ASSOCIATED"},
		"roam-ap-02@10.254.200.12": {"STATION_ASSOCIATED", "STATION_ROAMED"},
	}
	for ap, sequence := range want {
		if !slices.Equal(got[ap], sequence) {
			t.Errorf("%s sent %v, want %v", ap, got[ap], sequence)
		}
	}
}
