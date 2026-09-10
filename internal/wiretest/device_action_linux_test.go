//go:build linux && integration

package wiretest_test

import (
	"errors"
	"net"
	"testing"
	"time"

	"github.com/gosnmp/gosnmp"

	"github.com/MustardSeedNetworks/niac-go/internal/capture"
	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
	"github.com/MustardSeedNetworks/niac-go/internal/logging"
	"github.com/MustardSeedNetworks/niac-go/internal/protocols"
)

func TestRebootAndSTPStateOnUDPWire(t *testing.T) {
	requireWire(t)
	collector, err := net.ListenUDP(
		"udp4",
		&net.UDPAddr{IP: net.ParseIP("10.254.200.50"), Port: 162},
	)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = collector.Close() })
	cfg := &config.Config{Devices: []config.Device{
		actionWireDevice("action-switch", "10.254.200.11", 11),
		actionWireDevice("healthy-switch", "10.254.200.12", 12),
	}}
	stack := startActionWireStack(t, cfg, nil)
	target, peer := dialActionHost(t, "10.254.200.11"), dialActionHost(t, "10.254.200.12")
	baseline, peerBaseline := readActionMIB(t, target), readActionMIB(t, peer)
	if baseline[0] < 24*60*60*100 {
		t.Fatalf("baseline does not predate reboot: %v", baseline)
	}
	if err = stack.ExecuteDeviceAction("action-switch", devicestate.ActionReboot, "reboot-1"); err != nil {
		t.Fatal(err)
	}
	trap := receiveActionTrap(t, collector)
	rebooted := readActionMIB(t, target)
	if rebooted[0] >= baseline[0] || trap.Variables[1].Value != ".1.3.6.1.6.3.1.1.5.1" ||
		gosnmp.ToBigInt(trap.Variables[0].Value).Uint64() > rebooted[0] {
		t.Fatalf("reboot trap and uptime disagree: trap=%+v mib=%v", trap.Variables, rebooted)
	}
	if err = stack.ExecuteDeviceAction("action-switch", devicestate.ActionSTPTopologyChange, "stp-1"); err != nil {
		t.Fatal(err)
	}
	changed := readActionMIB(t, target)
	if changed[1] != baseline[1]+1 {
		t.Fatalf("STP counter=%d, baseline=%d", changed[1], baseline[1])
	}
	if got := readActionMIB(t, peer); got[0] < peerBaseline[0] || got[1] != peerBaseline[1] {
		t.Fatalf("peer changed: %v, baseline=%v", got, peerBaseline)
	}
	saved := stack.ExportDeviceStates()
	stack.Stop()
	recovered := startActionWireStack(t, cfg, saved)
	for _, action := range []struct {
		kind devicestate.DeviceActionType
		id   string
	}{{devicestate.ActionReboot, "reboot-1"}, {devicestate.ActionSTPTopologyChange, "stp-1"}} {
		if err = recovered.ExecuteDeviceAction("action-switch", action.kind, action.id); err != nil {
			t.Fatal(err)
		}
	}
	if got := readActionMIB(t, target); got[0] < changed[0] || got[1] != changed[1] {
		t.Fatalf("recovery replayed an action: %v, before=%v", got, changed)
	}
	if err = collector.SetReadDeadline(time.Now().Add(250 * time.Millisecond)); err != nil {
		t.Fatal(err)
	}
	buffer := make([]byte, 2048)
	if _, _, err = collector.ReadFromUDP(buffer); err == nil {
		t.Fatal("recovery or duplicate delivery emitted another coldStart")
	} else if timeout, ok := errors.AsType[net.Error](err); !ok || !timeout.Timeout() {
		t.Fatal(err)
	}
	t.Log(
		"actual UDP: reboot coldStart and uptime reset, STP counter, healthy peer, no action replay",
	)
}

func actionWireDevice(name, address string, suffix byte) config.Device {
	return config.Device{
		Name: name, Type: "switch", MACAddress: net.HardwareAddr{2, 0, 0, 0, 0, suffix},
		IPAddresses: []net.IP{net.ParseIP(address)},
		STPConfig:   &config.STPConfig{Enabled: true},
		SNMPConfig: config.SNMPConfig{Community: "action_demo", Traps: &config.TrapConfig{
			Enabled: true, Receivers: []string{"10.254.200.50"}, Community: "action_demo",
			ColdStart: &config.TrapTriggerConfig{Enabled: true},
		}},
	}
}

func startActionWireStack(
	t *testing.T,
	cfg *config.Config,
	states map[string]devicestate.State,
) *protocols.Stack {
	t.Helper()
	engine, err := capture.New(simIface, 0)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(engine.Close)
	stack := protocols.NewStack(engine, cfg, logging.NewDebugConfig(0))
	if states != nil {
		if err = stack.RestoreDeviceStates(states); err != nil {
			t.Fatal(err)
		}
	}
	if err = stack.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(stack.Stop)
	return stack
}

func dialActionHost(t *testing.T, address string) *gosnmp.GoSNMP {
	t.Helper()
	client := &gosnmp.GoSNMP{
		Target:    address,
		Port:      161,
		Community: "action_demo",
		Version:   gosnmp.Version2c,
		Timeout:   time.Second,
	}
	if err := client.Connect(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = client.Conn.Close() })
	return client
}

func readActionMIB(t *testing.T, client *gosnmp.GoSNMP) [3]uint64 {
	t.Helper()
	oids := []string{".1.3.6.1.2.1.1.3.0", ".1.3.6.1.2.1.17.2.4.0", ".1.3.6.1.2.1.17.2.3.0"}
	packet, err := client.Get(oids)
	if err != nil || packet == nil || packet.Error != gosnmp.NoError ||
		len(packet.Variables) != len(oids) {
		t.Fatalf("SNMP GET: packet=%+v err=%v", packet, err)
	}
	var values [3]uint64
	for index, kind := range []gosnmp.Asn1BER{gosnmp.TimeTicks, gosnmp.Counter32, gosnmp.TimeTicks} {
		pdu := packet.Variables[index]
		if pdu.Name != oids[index] || pdu.Type != kind {
			t.Fatalf("unexpected action MIB row: %+v", pdu)
		}
		values[index] = gosnmp.ToBigInt(pdu.Value).Uint64()
	}
	return values
}

func receiveActionTrap(t *testing.T, collector *net.UDPConn) *gosnmp.SnmpPacket {
	t.Helper()
	if err := collector.SetReadDeadline(time.Now().Add(3 * time.Second)); err != nil {
		t.Fatal(err)
	}
	buffer := make([]byte, 2048)
	count, source, err := collector.ReadFromUDP(buffer)
	if err != nil {
		t.Fatal(err)
	}
	if source.IP.String() != "10.254.200.11" || source.Port != 162 {
		t.Fatalf("wrong reboot source: %s", source)
	}
	decoder := gosnmp.GoSNMP{Version: gosnmp.Version2c}
	packet, err := decoder.SnmpDecodePacket(buffer[:count])
	if err != nil || packet == nil || packet.PDUType != gosnmp.SNMPv2Trap ||
		len(packet.Variables) < 2 {
		t.Fatalf("invalid reboot trap: %+v err=%v", packet, err)
	}
	return packet
}
