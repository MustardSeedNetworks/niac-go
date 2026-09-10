//go:build linux && integration

package wiretest_test

import (
	"context"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/MustardSeedNetworks/niac-go/internal/api"
	"github.com/MustardSeedNetworks/niac-go/internal/daemon"
	"github.com/MustardSeedNetworks/niac-go/internal/fabric"
)

func TestAuthoredLinkFaultEmitsSyslogOnTheWire(t *testing.T) {
	requireWire(t)
	collector, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.ParseIP(faultClientAddr), Port: 514})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = collector.Close() })
	data, err := os.ReadFile(filepath.Join("testdata", "syslog-timeline.yaml"))
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
		SessionID: "wiretest-syslog", Interface: simIface, Attachment: "tester",
		AttachmentMode: fabric.ModeAccess, AccessVLAN: accessVLAN, ConfigData: string(data),
	}); err != nil {
		t.Fatal(err)
	}
	if err = collector.SetReadDeadline(time.Now().Add(20 * time.Second)); err != nil {
		t.Fatal(err)
	}
	var previous uint64
	for _, expected := range []syslogExpectation{
		{"<132>1", "FAULT_UPDATED", "fault.updated"},
		{"<133>1", "FAULT_CLEARED", "fault.cleared"},
	} {
		fields := receiveSyslog(t, collector, expected)
		version, parseErr := strconv.ParseUint(strings.TrimPrefix(fields[7], "version="), 10, 64)
		if parseErr != nil || version <= previous {
			t.Fatalf("event version did not increase: %q after %d", fields[7], previous)
		}
		previous = version
	}
}

type syslogExpectation struct{ priority, messageID, kind string }

func receiveSyslog(t *testing.T, collector *net.UDPConn, expected syslogExpectation) []string {
	t.Helper()
	buffer := make([]byte, 2048)
	count, source, err := collector.ReadFromUDP(buffer)
	if err != nil {
		t.Fatalf("receive %s: %v", expected.messageID, err)
	}
	message := string(buffer[:count])
	fields := strings.Fields(message)
	if source.IP.String() != "10.254.200.1" || source.Port != 514 || len(fields) != 10 ||
		fields[0] != expected.priority || fields[2] != "LAB-SYSLOG-R1" || fields[3] != "niac" ||
		fields[4] != "-" || fields[5] != expected.messageID || fields[6] != "-" ||
		fields[8] != "kind="+expected.kind || fields[9] != `target="Access1:link_down"` {
		t.Fatalf("unexpected syslog from %s: %q", source, message)
	}
	if _, err = time.Parse(time.RFC3339Nano, fields[1]); err != nil {
		t.Fatalf("invalid timestamp: %q", fields[1])
	}
	return fields
}

func shutdownSyslogDaemon(t *testing.T, d *daemon.Daemon) {
	t.Helper()
	if d.GetStatus().Running {
		if err := d.StopSimulation(""); err != nil {
			t.Error(err)
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := d.Shutdown(ctx); err != nil {
		t.Error(err)
	}
}
