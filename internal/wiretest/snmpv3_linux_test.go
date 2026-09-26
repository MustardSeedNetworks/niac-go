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

	// The engine drops a wrong digest rather than reporting it (#2370), so any
	// error is a refusal here; a varbind back is the failure.
	_, err = dialDeviceV3(t, authored, switchName, "NotTheDemoPass", 0).Get([]string{sysNameOID})
	if err == nil {
		t.Errorf("%s answered an SNMPv3 GET authenticated with the wrong passphrase", switchName)
	}
	t.Logf("%s: wrong passphrase refused: %v", switchName, err)

	upsName := firstDeviceWithRole(t, authored, "ups")
	_, err = dialDeviceV3(t, authored, upsName, packV3AuthPass, 0).Get([]string{sysNameOID})
	if err == nil {
		t.Errorf("endpoint appliance %s answered SNMPv3; the pack authors v3 on managed devices only", upsName)
	}
	t.Logf("%s: no SNMPv3 engine, as authored: %v", upsName, err)
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
