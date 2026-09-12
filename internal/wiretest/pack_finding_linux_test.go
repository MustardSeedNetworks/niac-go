//go:build linux && integration

package wiretest_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/gosnmp/gosnmp"

	"github.com/MustardSeedNetworks/niac-go/internal/api"
	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/daemon"
	"github.com/MustardSeedNetworks/niac-go/internal/fabric"
	"github.com/MustardSeedNetworks/niac-go/internal/scenario"
)

// Counter columns a consumer reads to see the findings that are counters.
const (
	oidIfDescr     = ".1.3.6.1.2.1.2.2.1.2"
	oidIfInErrors  = ".1.3.6.1.2.1.2.2.1.14"
	oidIfInDiscard = ".1.3.6.1.2.1.2.2.1.13"
	oidIfHCInOcts  = ".1.3.6.1.2.1.31.1.1.1.6"
)

// Each pack ships one finding (P2-7). Authoring it is not enough: a finding a
// consumer cannot see on the wire is not a finding, so each counter-shaped one
// is polled here through SNMP exactly as a collector would.
//
// The device under test is site-internal, reachable only because the namespace
// now routes through the edge router the way a real tester on that segment
// does.
func TestEachPackFindingIsVisibleOnTheWire(t *testing.T) {
	cases := []struct {
		pack      string
		device    string
		iface     string
		column    string
		columnOID string
	}{
		{"manufacturing", "PLT-ACC-SW01", "HundredGigabitEthernet1/0/49", "ifInErrors", oidIfInErrors},
		{"campus", "NTH-ACC-SW01", "HundredGigabitEthernet1/0/49", "ifInDiscards", oidIfInDiscard},
		{"hospital", "MED-ACC-SW02", "HundredGigabitEthernet1/0/49", "ifHCInOctets", oidIfHCInOcts},
	}

	for _, testCase := range cases {
		t.Run(testCase.pack, func(t *testing.T) {
			authored := startPack(t, testCase.pack)
			client := dialDevice(t, authored, testCase.device)
			index := interfaceIndex(t, client, testCase.iface)

			first := counterAt(t, client, testCase.columnOID, index)
			// Two polls a second apart: a finding that is true at the first
			// poll still has to keep moving, and a static value would mean the
			// fault armed and then did nothing.
			time.Sleep(time.Second)
			second := counterAt(t, client, testCase.columnOID, index)

			if second <= first {
				t.Errorf("%s %s %s did not move between polls (%d then %d); the finding is not visible to a collector",
					testCase.device, testCase.iface, testCase.column, first, second)
			}
			t.Logf("%s: %s %s %s moved %d -> %d",
				testCase.pack, testCase.device, testCase.iface, testCase.column, first, second)
		})
	}
}

// startPack generates and starts one shipped pack the way the product does, so
// the thing under test is the artifact a customer runs.
func startPack(t *testing.T, id string) *config.Config {
	t.Helper()
	requireWire(t)

	var pack scenario.Pack
	for _, candidate := range scenario.Packs() {
		if candidate.ID == id {
			pack = candidate

			break
		}
	}
	if pack.ID == "" {
		t.Fatalf("no pack with id %q; scenario.Packs() no longer ships it", id)
	}

	result, err := scenario.Generate(pack.Request)
	if err != nil {
		t.Fatalf("scenario.Generate(%s): %v", id, err)
	}
	authored, err := config.LoadYAMLBytes(result.YAML)
	if err != nil {
		t.Fatalf("loading the generated %s YAML: %v", id, err)
	}

	t.Setenv("NIAC_CONFIGS_DIR", t.TempDir())
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
		SessionID:      "wiretest-finding-" + id,
		Interface:      simIface,
		Attachment:     pack.Request.AttachmentName,
		AttachmentMode: fabric.ModeAccess,
		AccessVLAN:     accessVLAN,
		ConfigData:     string(result.YAML),
	}); startErr != nil {
		t.Fatalf("StartSimulation(%s) on %s: %v", id, simIface, startErr)
	}
	t.Cleanup(func() {
		if stopErr := d.StopSimulation(""); stopErr != nil {
			t.Errorf("StopSimulation: %v", stopErr)
		}
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = d.Shutdown(ctx)
	})

	return authored
}

func dialDevice(t *testing.T, authored *config.Config, name string) *gosnmp.GoSNMP {
	t.Helper()
	for index := range authored.Devices {
		device := &authored.Devices[index]
		if device.Name != name || len(device.IPAddresses) == 0 {
			continue
		}
		client := &gosnmp.GoSNMP{
			Target:    device.IPAddresses[0].String(),
			Port:      161,
			Community: device.SNMPConfig.Community,
			Version:   gosnmp.Version2c,
			Timeout:   3 * time.Second,
			Retries:   4,
		}
		if err := client.Connect(); err != nil {
			t.Fatalf("connecting to %s (%s): %v", name, client.Target, err)
		}
		t.Cleanup(func() { _ = client.Conn.Close() })

		return client
	}
	t.Fatalf("no device named %q with an address in the generated pack", name)

	return nil
}

// interfaceIndex resolves ifIndex by walking ifDescr, because the index is
// assigned by the agent and hard-coding one would pin a number rather than the
// interface a finding names.
func interfaceIndex(t *testing.T, client *gosnmp.GoSNMP, name string) string {
	t.Helper()
	rows, err := client.WalkAll(oidIfDescr)
	if err != nil {
		t.Fatalf("walking ifDescr on %s: %v", client.Target, err)
	}
	for _, row := range rows {
		octets, ok := row.Value.([]byte)
		if !ok || string(octets) != name {
			continue
		}

		return strings.TrimPrefix(row.Name, oidIfDescr+".")
	}
	t.Fatalf("no interface named %q on %s; the pack no longer generates it", name, client.Target)

	return ""
}

func counterAt(t *testing.T, client *gosnmp.GoSNMP, column, index string) uint64 {
	t.Helper()
	oid := fmt.Sprintf("%s.%s", column, index)
	result, err := client.Get([]string{oid})
	if err != nil {
		t.Fatalf("GET %s on %s: %v", oid, client.Target, err)
	}
	if len(result.Variables) == 0 {
		t.Fatalf("GET %s returned no varbind", oid)
	}

	return gosnmp.ToBigInt(result.Variables[0].Value).Uint64()
}
