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
	got := runRoamTimeline(t, "wiretest-roam", nil, 4)

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

// TestRoamCauseTakesTheRadioDownBeforeTheStationLeavesOnTheWire: with
// `cause: radio_down` the old AP reports its radio going down before the
// station's roam, and back up before the station returns, so a collector sees
// why the station moved.
func TestRoamCauseTakesTheRadioDownBeforeTheStationLeavesOnTheWire(t *testing.T) {
	got := runRoamTimeline(t, "wiretest-roam-cause", strings.NewReplacer(
		"    syslog:\n      enabled: true\n      receivers: [\"10.254.200.50:514\"]\n    interfaces:\n      - name: Management0\n        network: wire-transit\n        address: 10.254.200.11/24",
		"    snmp_agent:\n      community: public\n    syslog:\n      enabled: true\n      receivers: [\"10.254.200.50:514\"]\n    interfaces:\n      - name: Management0\n        network: wire-transit\n        address: 10.254.200.11/24",
		"            to: roam-ap-02\n",
		"            to: roam-ap-02\n            cause: radio_down\n",
	), 8)

	want := map[string][]string{
		"roam-ap-01@10.254.200.11": {
			"FAULT_UPDATED", "LINK_DOWN", "STATION_ROAMED",
			"FAULT_CLEARED", "LINK_UP", "STATION_ASSOCIATED",
		},
		"roam-ap-02@10.254.200.12": {"STATION_ASSOCIATED", "STATION_ROAMED"},
	}
	for ap, sequence := range want {
		if !slices.Equal(got[ap], sequence) {
			t.Errorf("%s sent %v, want %v", ap, got[ap], sequence)
		}
	}
}

// runRoamTimeline starts the roam fixture, edited by edit when it is not nil,
// and returns the message IDs each AP sent, keyed by hostname@source address.
// The two APs dispatch independently, so only each AP's own order is fixed.
func runRoamTimeline(t *testing.T, session string, edit *strings.Replacer, messages int) map[string][]string {
	t.Helper()
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
	config := string(data)
	if edit != nil {
		edited := edit.Replace(config)
		if strings.Count(edited, "cause: radio_down") != 1 || strings.Count(edited, "snmp_agent:") != 1 {
			t.Fatal("the fixture edit did not apply; roam-timeline.yaml changed shape")
		}
		config = edited
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
		SessionID: session, Interface: simIface, Attachment: "tester",
		AttachmentMode: fabric.ModeAccess, AccessVLAN: accessVLAN, ConfigData: config,
	}); err != nil {
		t.Fatal(err)
	}
	if err = collector.SetReadDeadline(time.Now().Add(20 * time.Second)); err != nil {
		t.Fatal(err)
	}

	got := map[string][]string{}
	buffer := make([]byte, 2048)
	for range messages {
		count, source, readErr := collector.ReadFromUDP(buffer)
		if readErr != nil {
			t.Fatalf("receive roam syslog (have %v): %v", got, readErr)
		}
		fields := strings.Fields(string(buffer[:count]))
		station := len(fields) == 10 && strings.HasPrefix(fields[5], "STATION_")
		if len(fields) != 10 || station && (fields[0] != "<134>1" || fields[9] != `target="02:c0:17:a4:03:6b"`) {
			t.Fatalf("unexpected syslog from %s: %q", source, buffer[:count])
		}
		key := fields[2] + "@" + source.IP.String()
		got[key] = append(got[key], fields[5])
	}
	return got
}
