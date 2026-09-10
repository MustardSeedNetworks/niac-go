//go:build linux && integration

package wiretest_test

import (
	"reflect"
	"testing"
	"time"

	"github.com/gosnmp/gosnmp"

	"github.com/MustardSeedNetworks/niac-go/internal/api"
	"github.com/MustardSeedNetworks/niac-go/internal/daemon"
	"github.com/MustardSeedNetworks/niac-go/internal/fabric"
	"github.com/MustardSeedNetworks/niac-go/internal/templates"
)

func TestResourcePressureTemplateOnTheWire(t *testing.T) {
	requireWire(t)
	template, err := templates.Get("resource-pressure")
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
		SessionID: "resource-pressure", Interface: simIface, Attachment: "tester",
		AttachmentMode: fabric.ModeAccess, AccessVLAN: accessVLAN, ConfigData: template.Content,
	}); err != nil {
		t.Fatal(err)
	}
	active, peer := dialResourceHost(t, "10.254.200.11"), dialResourceHost(t, "10.254.200.12")
	baseline := []int64{18, 22, 1048576, 16777216, 4194304, 67108864, 4096, 4096}
	assertResourceWireValues(t, active, baseline)
	assertResourceWireValues(t, peer, baseline)
	t.Log("both servers returned healthy CPU, RAM and disk values over SNMP")
	pressure := []int64{95, 95, 3774873, 63753420, 4194304, 67108864, 4096, 4096}
	awaitResourceWireValues(t, active, pressure)
	assertResourceWireValues(t, peer, baseline)
	t.Log("APP01 returned CPU 95%, RAM 90%, disk 95%; APP02 and capacities unchanged")
	awaitResourceWireValues(t, active, baseline)
	assertResourceWireValues(t, peer, baseline)
	t.Log("APP01 reset to exact baseline; APP02 remained healthy")
}

func dialResourceHost(t *testing.T, address string) *gosnmp.GoSNMP {
	t.Helper()
	client := &gosnmp.GoSNMP{
		Target: address, Port: 161, Community: "resource_demo", Version: gosnmp.Version2c,
		Timeout: time.Second, Retries: 2,
	}
	if err := client.Connect(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = client.Conn.Close() })
	return client
}

func resourceWireValues(t *testing.T, client *gosnmp.GoSNMP) []int64 {
	t.Helper()
	oids := []string{
		"1.3.6.1.2.1.25.3.3.1.2.1", "1.3.6.1.2.1.25.3.3.1.2.2",
		"1.3.6.1.2.1.25.2.3.1.6.10", "1.3.6.1.2.1.25.2.3.1.6.20",
		"1.3.6.1.2.1.25.2.3.1.5.10", "1.3.6.1.2.1.25.2.3.1.5.20",
		"1.3.6.1.2.1.25.2.3.1.4.10", "1.3.6.1.2.1.25.2.3.1.4.20",
	}
	packet, err := client.Get(oids)
	if err != nil {
		t.Fatal(err)
	}
	if packet.Error != gosnmp.NoError || len(packet.Variables) != len(oids) {
		t.Fatalf("invalid SNMP response: %#v", packet)
	}
	values := make([]int64, len(oids))
	for i, pdu := range packet.Variables {
		if pdu.Type != gosnmp.Integer || pdu.Name != "."+oids[i] {
			t.Fatalf("unexpected resource PDU: %#v", pdu)
		}
		values[i] = gosnmp.ToBigInt(pdu.Value).Int64()
	}
	return values
}

func assertResourceWireValues(t *testing.T, client *gosnmp.GoSNMP, want []int64) {
	t.Helper()
	if got := resourceWireValues(t, client); !reflect.DeepEqual(got, want) {
		t.Fatalf("%s resources = %v, want %v", client.Target, got, want)
	}
}

func awaitResourceWireValues(t *testing.T, client *gosnmp.GoSNMP, want []int64) {
	t.Helper()
	deadline := time.NewTimer(25 * time.Second)
	defer deadline.Stop()
	poll := time.NewTicker(100 * time.Millisecond)
	defer poll.Stop()
	for {
		got := resourceWireValues(t, client)
		if reflect.DeepEqual(got, want) {
			return
		}
		select {
		case <-deadline.C:
			t.Fatalf("%s resources = %v, never reached %v", client.Target, got, want)
		case <-poll.C:
		}
	}
}
