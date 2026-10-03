//go:build linux && integration

package wiretest_test

import (
	"testing"
	"time"

	"github.com/gosnmp/gosnmp"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
)

// P5-9: every managed pack device answers SNMPv3 authPriv as the published
// user, from the pack's own attachment port. The walk is asserted by content
// (sysName names the device), a wrong passphrase must be refused, and an
// endpoint appliance, which stays v2c-only, must not answer v3 at all: an
// engine that accepted anything, or answered for every device, would pass
// the walk alone.
const (
	packV3User     = "netops"
	packV3AuthPass = "NetAllyDemoAuth"
	packV3PrivPass = "NetAllyDemoPriv"
	systemGroupOID = ".1.3.6.1.2.1.1"
	sysNameOID     = ".1.3.6.1.2.1.1.5.0"
)

func TestPackDeviceAnswersSNMPv3OnTheWire(t *testing.T) {
	authored, _ := startPack(t, "hospital")
	switchName := authored.Attachments[0].At.Device

	client := dialDeviceV3(t, authored, switchName, packV3AuthPass, 4)
	var sysName string
	walked := 0
	err := client.BulkWalk(systemGroupOID, func(pdu gosnmp.SnmpPDU) error {
		walked++
		if pdu.Name == sysNameOID {
			if value, ok := pdu.Value.([]byte); ok {
				sysName = string(value)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("SNMPv3 authPriv walk of %s on %s: %v", systemGroupOID, switchName, err)
	}
	if sysName != switchName {
		t.Fatalf(
			"SNMPv3 walk of %s returned sysName %q, want %q (%d varbinds)",
			switchName,
			sysName,
			switchName,
			walked,
		)
	}
	t.Logf("%s: SNMPv3 authPriv (SHA-256/AES) walk of %s returned %d varbinds, sysName %q",
		switchName, systemGroupOID, walked, sysName)

	assertWrongDigestReport(t, client, switchName)

	upsName := firstDeviceWithRole(t, authored, "ups")
	_, err = dialDeviceV3(t, authored, upsName, packV3AuthPass, 0).Get([]string{sysNameOID})
	if err == nil {
		t.Errorf("endpoint appliance %s answered SNMPv3; the pack authors v3 on managed devices only", upsName)
	}
	t.Logf("%s: no SNMPv3 engine, as authored: %v", upsName, err)
}

// assertWrongDigestReport sends a GET signed with the wrong passphrase over
// the discovered session and expects the usmStatsWrongDigests Report (RFC 3414
// §3.2 step 6, #2370). gosnmp's manager discards an unauthenticated Report, so
// the request goes out raw on the client's socket.
func assertWrongDigestReport(t *testing.T, client *gosnmp.GoSNMP, name string) {
	t.Helper()
	discovered, ok := client.SecurityParameters.(*gosnmp.UsmSecurityParameters)
	if !ok || discovered.AuthoritativeEngineID == "" {
		t.Fatalf("%s: no discovered SNMPv3 engine to send the wrong digest to", name)
	}
	usm := &gosnmp.UsmSecurityParameters{
		AuthoritativeEngineID:    discovered.AuthoritativeEngineID,
		AuthoritativeEngineBoots: discovered.AuthoritativeEngineBoots,
		AuthoritativeEngineTime:  discovered.AuthoritativeEngineTime,
		UserName:                 packV3User,
		AuthenticationProtocol:   gosnmp.SHA256,
		AuthenticationPassphrase: "NotTheDemoPass",
		PrivacyProtocol:          gosnmp.AES,
		PrivacyPassphrase:        packV3PrivPass,
	}
	if err := usm.InitSecurityKeys(); err != nil {
		t.Fatalf("wrong-passphrase keys: %v", err)
	}
	request := &gosnmp.SnmpPacket{
		Version:            gosnmp.Version3,
		MsgFlags:           gosnmp.AuthPriv | gosnmp.Reportable,
		SecurityModel:      gosnmp.UserSecurityModel,
		SecurityParameters: usm,
		ContextEngineID:    discovered.AuthoritativeEngineID,
		PDUType:            gosnmp.GetRequest,
		MsgID:              2370,
		RequestID:          2370,
		MsgMaxSize:         65507,
		Variables:          []gosnmp.SnmpPDU{{Name: sysNameOID, Type: gosnmp.Null}},
	}
	if err := usm.InitPacket(request); err != nil {
		t.Fatalf("wrong-passphrase salt: %v", err)
	}
	wire, err := request.MarshalMsg()
	if err != nil {
		t.Fatalf("marshal wrong-passphrase GET: %v", err)
	}
	if _, err = client.Conn.Write(wire); err != nil {
		t.Fatalf("send wrong-passphrase GET to %s: %v", name, err)
	}
	if err = client.Conn.SetReadDeadline(time.Now().Add(client.Timeout)); err != nil {
		t.Fatal(err)
	}
	buf := make([]byte, 65535)
	n, err := client.Conn.Read(buf)
	if err != nil {
		t.Fatalf("%s sent no Report for a wrong passphrase: %v", name, err)
	}

	decoder := &gosnmp.GoSNMP{
		Version:            gosnmp.Version3,
		SecurityModel:      gosnmp.UserSecurityModel,
		SecurityParameters: &gosnmp.UsmSecurityParameters{UserName: "decoder"},
	}
	report, err := decoder.SnmpDecodePacket(buf[:n])
	if err != nil {
		t.Fatalf("decode %s's reply to a wrong passphrase: %v", name, err)
	}
	const usmStatsWrongDigests = ".1.3.6.1.6.3.15.1.1.5.0"
	if report.PDUType != gosnmp.Report || len(report.Variables) != 1 ||
		report.Variables[0].Name != usmStatsWrongDigests || report.MsgID != request.MsgID {
		t.Fatalf("%s answered a wrong passphrase with %v msgID %d %+v, want a Report of %s for msgID %d",
			name, report.PDUType, report.MsgID, report.Variables, usmStatsWrongDigests, request.MsgID)
	}
	t.Logf("%s: wrong passphrase refused with Report %s = %v",
		name, usmStatsWrongDigests, report.Variables[0].Value)
}

func dialDeviceV3(t *testing.T, authored *config.Config, name, authPass string, retries int) *gosnmp.GoSNMP {
	t.Helper()
	for index := range authored.Devices {
		device := &authored.Devices[index]
		if device.Name != name || len(device.IPAddresses) == 0 {
			continue
		}
		client := &gosnmp.GoSNMP{
			Target:        device.IPAddresses[0].String(),
			Port:          161,
			Version:       gosnmp.Version3,
			SecurityModel: gosnmp.UserSecurityModel,
			MsgFlags:      gosnmp.AuthPriv,
			SecurityParameters: &gosnmp.UsmSecurityParameters{
				UserName:                 packV3User,
				AuthenticationProtocol:   gosnmp.SHA256,
				AuthenticationPassphrase: authPass,
				PrivacyProtocol:          gosnmp.AES,
				PrivacyPassphrase:        packV3PrivPass,
			},
			Timeout: 3 * time.Second,
			Retries: retries,
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
