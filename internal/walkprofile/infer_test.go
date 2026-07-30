package walkprofile_test

import (
	"testing"

	"github.com/gosnmp/gosnmp"

	"github.com/MustardSeedNetworks/niac-go/internal/protocols/snmp"
	"github.com/MustardSeedNetworks/niac-go/internal/walkprofile"
)

func TestInferMatchesSupportedSwitchAndCapabilities(t *testing.T) {
	entries := []snmp.WalkEntry{
		{OID: ".1.3.6.1.2.1.1.1.0", Type: gosnmp.OctetString, Value: "Cisco IOS XE Software"},
		{OID: ".1.3.6.1.2.1.1.2.0", Type: gosnmp.ObjectIdentifier, Value: "1.3.6.1.4.1.9.1.2238"},
		{OID: ".1.3.6.1.2.1.2.2.1.2.1", Type: gosnmp.OctetString, Value: "GigabitEthernet1/0/1"},
		{OID: ".1.0.8802.1.1.2.1.4.1.1.9.1.1", Type: gosnmp.OctetString, Value: "peer"},
	}
	review := walkprofile.Infer("captured/c9300.walk", entries)
	if review.Profile.DeviceType != "switch" || review.Profile.Vendor != "cisco" ||
		review.Profile.SysObjectID != "1.3.6.1.4.1.9.1.2238" {
		t.Fatalf("profile = %+v", review.Profile)
	}
	if review.Profile.InterfaceCount != 1 || len(review.Profile.SupportedSNMPData) != 3 {
		t.Fatalf("evidence = %+v", review.Profile)
	}
}

func TestInferUsesConservativeFallback(t *testing.T) {
	review := walkprofile.Infer("captured/unknown.walk", []snmp.WalkEntry{
		{OID: ".1.3.6.1.2.1.1.1.0", Type: gosnmp.OctetString, Value: "Acme edge firewall"},
	})
	if review.Profile.DeviceType != "firewall" || review.Profile.Vendor != "generic" ||
		review.Profile.Role != "captured-firewall" {
		t.Fatalf("profile = %+v", review.Profile)
	}
}
