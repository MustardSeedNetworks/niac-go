package snmp

import (
	"bytes"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

type identityFixture struct {
	agent *Agent
	mac   net.HardwareAddr
}

func loadCapturedIdentity(t *testing.T, macText, walkContent string) identityFixture {
	t.Helper()
	walk := filepath.Join(t.TempDir(), "identity.walk")
	if err := os.WriteFile(walk, []byte(walkContent), 0o600); err != nil {
		t.Fatal(err)
	}
	mac, err := net.ParseMAC(macText)
	if err != nil {
		t.Fatal(err)
	}
	device := createTestDevice()
	device.MACAddress = mac
	agent := NewAgent(device, 0)
	if loadErr := agent.LoadWalkFile(walk); loadErr != nil {
		t.Fatal(loadErr)
	}
	return identityFixture{agent: agent, mac: mac}
}

func assertAuthoredMAC(t *testing.T, fixture identityFixture) {
	t.Helper()
	for _, oid := range []string{dot1dBaseBridgeAddress, ifPhysAddress + ".1", lldpLocChassisID} {
		value, err := fixture.agent.HandleGet(oid)
		if err != nil {
			t.Fatalf("HandleGet(%s): %v", oid, err)
		}
		got, ok := value.Value.([]byte)
		if !ok || !bytes.Equal(got, fixture.mac) {
			t.Errorf("%s = %v, want %v", oid, value.Value, fixture.mac)
		}
	}
}

func assertAuthoredPhysicalAddress(t *testing.T, fixture identityFixture, index int) {
	t.Helper()
	got := physicalAddress(t, fixture.agent, index)
	if !bytes.Equal(got, fixture.mac) {
		t.Errorf("physical address %d = %v, want %v", index, got, fixture.mac)
	}
}

func assertDistinctPhysicalAddresses(t *testing.T, first, second identityFixture, index int) {
	t.Helper()
	firstMAC := physicalAddress(t, first.agent, index)
	secondMAC := physicalAddress(t, second.agent, index)
	captured, _ := net.ParseMAC("00:0c:ce:88:23:c8")
	if bytes.Equal(firstMAC, captured) || bytes.Equal(firstMAC, secondMAC) {
		t.Errorf("physical address %d was not made device-specific: %v, %v", index, firstMAC, secondMAC)
	}
}

func physicalAddress(t *testing.T, agent *Agent, index int) []byte {
	t.Helper()
	value, err := agent.HandleGet(ifPhysAddress + "." + strconv.Itoa(index))
	if err != nil {
		t.Fatal(err)
	}
	got, ok := value.Value.([]byte)
	if !ok {
		t.Fatalf("physical address %d type = %T, want []byte", index, value.Value)
	}
	return got
}

func assertOIDString(t *testing.T, agent *Agent, oid, want string) {
	t.Helper()
	value, err := agent.HandleGet(oid)
	if err != nil {
		t.Fatal(err)
	}
	if got := oidValueString(value); got != want {
		t.Errorf("%s = %q, want %q", oid, got, want)
	}
}

func assertEmptyPhysicalAddress(t *testing.T, agent *Agent) {
	t.Helper()
	value, err := agent.HandleGet(ifPhysAddress + ".2")
	if err != nil {
		t.Fatal(err)
	}
	if got := value.Value; got != "" {
		t.Errorf("logical interface physical address = %v, want empty", got)
	}
}

func assertPhysicalAddress(t *testing.T, agent *Agent, index int, want string) {
	t.Helper()
	assertOIDBytes(t, agent, ifPhysAddress+"."+strconv.Itoa(index), want)
}

func assertOIDBytes(t *testing.T, agent *Agent, oid, want string) {
	t.Helper()
	value, err := agent.HandleGet(oid)
	if err != nil {
		t.Fatal(err)
	}
	wantBytes, err := net.ParseMAC(want)
	if err != nil {
		t.Fatal(err)
	}
	got, ok := value.Value.([]byte)
	if !ok || !bytes.Equal(got, wantBytes) {
		t.Errorf("%s = %v, want %v", oid, value.Value, wantBytes)
	}
}

func oidString(t *testing.T, agent *Agent, oid string) string {
	t.Helper()
	value, err := agent.HandleGet(oid)
	if err != nil {
		t.Fatalf("HandleGet(%s): %v", oid, err)
	}
	got, ok := value.Value.(string)
	if !ok {
		t.Fatalf("%s type = %T, want string", oid, value.Value)
	}
	return got
}
