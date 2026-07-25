package snmp

import (
	"errors"
	"testing"
)

const capturedIdentityWalk = `.1.3.6.1.2.1.2.2.1.6.1 = Hex-STRING: 00 0C CE 88 23 C7
.1.3.6.1.2.1.2.2.1.6.2 = ""
.1.3.6.1.2.1.2.2.1.6.3 = Hex-STRING: 00 0C CE 88 23 C8
.1.3.6.1.2.1.2.2.1.6.4 = STRING: "00:0B:FD:D4:6F:DE"
.1.3.6.1.2.1.17.1.1.0 = Hex-STRING: 00 0B FD D4 6F DE
.1.3.6.1.2.1.47.1.1.1.1.11.1 = STRING: "FHK0722J116"
.1.0.8802.1.1.2.1.3.1.0 = INTEGER: 7
.1.0.8802.1.1.2.1.3.2.0 = Hex-STRING: 00 0B FD D4 6F DE
`

const physicalSerialOID = "1.3.6.1.2.1.47.1.1.1.1.11"

func TestLoadWalkFilePreservesAuthoredPhysicalIdentity(t *testing.T) {
	first := loadCapturedIdentity(t, "00:00:0c:00:01:01", capturedIdentityWalk)
	second := loadCapturedIdentity(t, "00:00:0c:00:01:02", capturedIdentityWalk)

	assertAuthoredMAC(t, first)
	assertAuthoredMAC(t, second)
	assertEmptyPhysicalAddress(t, first.agent)
	assertDistinctPhysicalAddresses(t, first, second, 3)
	assertAuthoredPhysicalAddress(t, first, 4)
	assertOIDString(t, first.agent, lldpLocChassisIDSubtype, "4")
	if len(first.agent.mib.sorted) != 0 {
		t.Fatal("physical identity refresh built the sorted OID index early")
	}
	firstSerial := oidString(t, first.agent, physicalSerialOID+".1")
	secondSerial := oidString(t, second.agent, physicalSerialOID+".1")
	if firstSerial == "FHK0722J116" || firstSerial == secondSerial {
		t.Fatalf("serials did not preserve unique authored identity: %q, %q", firstSerial, secondSerial)
	}
}

func TestLoadWalkFileDoesNotPublishNonEthernetMACIdentity(t *testing.T) {
	fixture := loadCapturedIdentity(t, "00:00:0c:00:01:01:02:03", capturedIdentityWalk)

	assertPhysicalAddress(t, fixture.agent, 1, "00:0c:ce:88:23:c7")
	assertPhysicalAddress(t, fixture.agent, 3, "00:0c:ce:88:23:c8")
	assertOIDBytes(t, fixture.agent, dot1dBaseBridgeAddress, "00:0b:fd:d4:6f:de")
	assertOIDBytes(t, fixture.agent, lldpLocChassisID, "00:0b:fd:d4:6f:de")
}

func TestLoadWalkFileDoesNotInventLLDPIdentity(t *testing.T) {
	fixture := loadCapturedIdentity(t, "00:00:0c:00:01:01", `.1.3.6.1.2.1.1.5.0 = STRING: "captured"`)
	_, err := fixture.agent.HandleGet(lldpLocChassisID)
	if !errors.Is(err, ErrNoSuchObject) {
		t.Fatalf("LLDP chassis ID error = %v, want ErrNoSuchObject", err)
	}
}
