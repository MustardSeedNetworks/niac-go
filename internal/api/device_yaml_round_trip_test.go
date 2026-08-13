package api

import (
	"strings"
	"testing"
)

// The device editor reads a device as YAML and writes it back. Parsing that
// document with a second, partial reader loses everything it does not know
// about: an operator who edits one line finds the addresses, agent and
// interfaces gone, with no error to say so. The read and write paths have to
// agree, so the write path uses the product's own loader.
func TestEditingADeviceKeepsWhatWasNotEdited(t *testing.T) {
	const document = `
name: MED-ACC-SW02
type: switch
mac: "00:00:0c:33:06:02"
ips:
  - 10.51.200.22
snmp_agent:
  community: NetAllyDemo
  sysname: MED-ACC-SW02
  sysdescr: Cisco C9350-48HX MED multigigabit access 2
interfaces:
  - name: Vlan200
    type: l2vlan
    address: 10.51.200.22/24
    speed: 100000
    admin_status: up
    oper_status: up
`

	device, err := parseDeviceFromYAML(document, "MED-ACC-SW02")
	if err != nil {
		t.Fatalf("parseDeviceFromYAML: %v", err)
	}
	if device.Type != "switch" {
		t.Errorf("type = %q, want switch", device.Type)
	}
	if len(device.IPAddresses) != 1 || device.IPAddresses[0].String() != "10.51.200.22" {
		t.Errorf("addresses = %v, want [10.51.200.22]", device.IPAddresses)
	}
	if device.SNMPConfig.Community != "NetAllyDemo" {
		t.Errorf("SNMP community = %q, want NetAllyDemo", device.SNMPConfig.Community)
	}
	if device.SNMPConfig.SysName != "MED-ACC-SW02" {
		t.Errorf("sysName = %q, want MED-ACC-SW02", device.SNMPConfig.SysName)
	}
	if len(device.Interfaces) != 1 || device.Interfaces[0].Name != "Vlan200" {
		t.Errorf("interfaces = %+v, want one Vlan200", device.Interfaces)
	}
}

// The hostname the request names is what the device is called, whatever the
// document says - that is how a clone or a rename works.
func TestTheRequestHostnameWins(t *testing.T) {
	device, err := parseDeviceFromYAML(
		"name: OLD-NAME\ntype: switch\nmac: \"00:00:0c:33:06:02\"\n", "NEW-NAME")
	if err != nil {
		t.Fatalf("parseDeviceFromYAML: %v", err)
	}
	if device.Name != "NEW-NAME" {
		t.Errorf("name = %q, want NEW-NAME", device.Name)
	}
}

// A document that is not a device must be refused rather than quietly yielding
// an empty one.
func TestAMalformedDeviceIsRefused(t *testing.T) {
	if _, err := parseDeviceFromYAML("type: [not, a, string]\n", "X"); err == nil {
		t.Error("a malformed device parsed without error")
	}
}

// A device with no MAC cannot be simulated, and the product's loader has always
// said so. The old reader accepted one and handed back something unusable; the
// editor now hears about it while the operator is still looking at the document.
func TestADeviceWithoutAMACIsRefused(t *testing.T) {
	_, err := parseDeviceFromYAML("name: X\ntype: switch\n", "X")
	if err == nil {
		t.Fatal("a device with no MAC parsed without error")
	}
	if !strings.Contains(err.Error(), "MAC") {
		t.Errorf("error = %v, want it to name the missing MAC", err)
	}
}

// The reader still refuses what the security checks reject, unchanged.
func TestValidationStillGuardsTheInput(t *testing.T) {
	deep := strings.Repeat("a:\n  ", 200) + "  b: c\n"
	if _, err := parseDeviceFromYAML(deep, "X"); err == nil {
		t.Error("a deeply nested document parsed without error")
	}
}
