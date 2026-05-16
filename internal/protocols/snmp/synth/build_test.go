package synth_test

import (
	"strings"
	"testing"

	"github.com/krisarmstrong/niac-go/internal/protocols/snmp/synth"
)

func mustPick(t *testing.T, v synth.Vendor, dt synth.DeviceType) synth.Profile {
	t.Helper()
	p, err := synth.Pick(v, dt)
	if err != nil {
		t.Fatalf("synth.Pick(%s, %s): %v", v, dt, err)
	}
	return p
}

func TestPickRejectsInvalidCombo(t *testing.T) {
	// Arista doesn't ship APs. Per the design-doc resolution we
	// return an error rather than falling back to generic — the
	// daemon's 422 path relies on this.
	if _, err := synth.Pick(synth.VendorAristaEOS, synth.TypeAccessPoint); err == nil {
		t.Fatal("expected error for arista-eos + access_point, got nil")
	}
	// Cisco IOS firewall is valid (ASA).
	if _, err := synth.Pick(synth.VendorCiscoIOS, synth.TypeFirewall); err != nil {
		t.Errorf("cisco-ios + firewall should be valid: %v", err)
	}
}

func TestPickJunosAccessPointRoutesToMist(t *testing.T) {
	// Junos + access_point is auto-routed to the Mist sub-profile
	// (Mist is API-first so the walk is intentionally minimal).
	p, err := synth.Pick(synth.VendorJunos, synth.TypeAccessPoint)
	if err != nil {
		t.Fatalf("expected junos+ap to route to Mist, got error: %v", err)
	}
	if p.Vendor != synth.VendorJuniperMist {
		t.Errorf("vendor = %q, want junos-mist (auto-routed)", p.Vendor)
	}
	if !strings.Contains(p.SysDescr, "Mist") {
		t.Errorf("Mist sysDescr should mention Mist, got %q", p.SysDescr)
	}
}

func TestBuildEmitsSystemGroup(t *testing.T) {
	p := mustPick(t, synth.VendorCiscoIOS, synth.TypeRouter)
	walk := string(synth.Build(p, synth.DeviceInput{
		Hostname: "edge-1",
		IP:       "10.0.0.1",
	}, synth.BuildOptions{}))

	mustContain(t, walk, ".1.3.6.1.2.1.1.1.0 = STRING:")             // sysDescr
	mustContain(t, walk, ".1.3.6.1.2.1.1.5.0 = STRING: \"edge-1\"")  // sysName
	mustContain(t, walk, ".1.3.6.1.2.1.1.2.0 = OID: .1.3.6.1.4.1.9") // Cisco enterprise
}

func TestBuildIfTableRowCount(t *testing.T) {
	p := mustPick(t, synth.VendorCiscoIOS, synth.TypeSwitch)
	walk := string(synth.Build(p, synth.DeviceInput{Hostname: "sw-1"}, synth.BuildOptions{InterfaceCount: 5}))

	// ifNumber should match the requested count.
	mustContain(t, walk, ".1.3.6.1.2.1.2.1.0 = INTEGER: 5")
	// Five ifDescr lines.
	if c := strings.Count(walk, ".1.3.6.1.2.1.2.2.1.2."); c != 5 {
		t.Errorf("ifDescr row count = %d, want 5", c)
	}
	// First and last interface names render per Cisco convention.
	mustContain(t, walk, "GigabitEthernet1/0/0")
	mustContain(t, walk, "GigabitEthernet1/0/4")
}

func TestBuildIfTableDefaultCountWhenZero(t *testing.T) {
	p := mustPick(t, synth.VendorCiscoIOS, synth.TypeSwitch)
	walk := string(synth.Build(p, synth.DeviceInput{Hostname: "sw-1"}, synth.BuildOptions{}))
	// Cisco switch profile default is 24.
	mustContain(t, walk, ".1.3.6.1.2.1.2.1.0 = INTEGER: 24")
}

func TestBuildIfTableCapsAtMaxInterfaces(t *testing.T) {
	p := mustPick(t, synth.VendorCiscoIOS, synth.TypeSwitch)
	walk := string(synth.Build(p, synth.DeviceInput{Hostname: "huge"}, synth.BuildOptions{InterfaceCount: 100_000}))
	// Even if the request asks for 100k, the synthesiser caps it.
	expected := ".1.3.6.1.2.1.2.1.0 = INTEGER: " + itoa(synth.MaxInterfaces)
	mustContain(t, walk, expected)
}

func TestBuildRouterEmitsIPMIBAndForwarding(t *testing.T) {
	p := mustPick(t, synth.VendorCiscoIOS, synth.TypeRouter)
	walk := string(synth.Build(p, synth.DeviceInput{
		Hostname:      "r1",
		IP:            "10.0.0.1",
		AdditionalIPs: []string{"10.0.1.1"},
	}, synth.BuildOptions{}))

	// Router → ipForwarding = 1 (forwarding).
	mustContain(t, walk, ".1.3.6.1.2.1.4.1.0 = INTEGER: 1")
	// ipAddrTable rows for both IPs.
	mustContain(t, walk, ".1.3.6.1.2.1.4.20.1.1.10.0.0.1 = IpAddress: 10.0.0.1")
	mustContain(t, walk, ".1.3.6.1.2.1.4.20.1.1.10.0.1.1 = IpAddress: 10.0.1.1")
}

func TestBuildSwitchSkipsIPMIB(t *testing.T) {
	p := mustPick(t, synth.VendorCiscoIOS, synth.TypeSwitch)
	walk := string(synth.Build(p, synth.DeviceInput{
		Hostname: "sw-1",
		IP:       "10.0.0.10",
	}, synth.BuildOptions{}))

	// L2 switch → no IP-MIB at all.
	if strings.Contains(walk, ".1.3.6.1.2.1.4.1.0") {
		t.Error("switch walk should not emit ipForwarding")
	}
	// But should emit the bridge group.
	mustContain(t, walk, ".1.3.6.1.2.1.17.1.1.0")
}

func TestBuildAccessPointMistMinimal(t *testing.T) {
	// Junos + access_point routes to Mist, and Mist's IncludeLLDP=false,
	// IncludeIPMIB=false, IncludeBridge=false — confirm none of those
	// OID prefixes show up. Mist APs are API-first; SNMP is just
	// system + ifTable.
	p, _ := synth.Pick(synth.VendorJunos, synth.TypeAccessPoint)
	walk := string(synth.Build(p, synth.DeviceInput{Hostname: "ap-1"}, synth.BuildOptions{}))

	for _, omitted := range []string{
		".1.3.6.1.2.1.4.1.0",   // IP-MIB
		".1.3.6.1.2.1.17.1.",   // BRIDGE-MIB
		".1.0.8802.1.1.2.1.3.", // LLDP-MIB
	} {
		if strings.Contains(walk, omitted) {
			t.Errorf("Mist AP walk should not include %s prefix", omitted)
		}
	}
	// But the system group must be present.
	mustContain(t, walk, ".1.3.6.1.2.1.1.5.0 = STRING: \"ap-1\"")
}

func TestSyntheticMACIsDeterministicAndLocallyAdministered(t *testing.T) {
	p := mustPick(t, synth.VendorGeneric, synth.TypeHost)
	w1 := string(synth.Build(p, synth.DeviceInput{Hostname: "host-1"}, synth.BuildOptions{}))
	w2 := string(synth.Build(p, synth.DeviceInput{Hostname: "host-1"}, synth.BuildOptions{}))
	if w1 != w2 {
		t.Error("same input should produce identical walk (synthetic MACs are deterministic)")
	}

	// MACs we emit live on the ifPhysAddress lines (.1.3.6.1.2.1.2.2.1.6.<idx>).
	// Locally-administered bit means the first octet is 02.
	for _, line := range strings.Split(w1, "\n") {
		if !strings.Contains(line, ".1.3.6.1.2.1.2.2.1.6.") {
			continue
		}
		// Line looks like: .1.3.6.1.2.1.2.2.1.6.1 = STRING: "02:AB:CD:..."
		if !strings.Contains(line, "\"02:") {
			t.Errorf("synthetic MAC should start with 02 (locally administered), got line: %s", line)
		}
	}
}

func TestSynthesizableMatrix(t *testing.T) {
	// A few spot checks of the (vendor, type) compatibility matrix.
	cases := []struct {
		v    synth.Vendor
		dt   synth.DeviceType
		want bool
	}{
		{synth.VendorCiscoIOS, synth.TypeSwitch, true},
		{synth.VendorAristaEOS, synth.TypeAccessPoint, false}, // Arista doesn't ship APs
		{synth.VendorAristaEOS, synth.TypeFirewall, false},    // Arista doesn't ship firewalls
		{synth.VendorJunos, synth.TypePrinter, false},         // No Junos printer
		{synth.VendorGeneric, synth.TypePrinter, true},        // Generic Linux always works
	}
	for _, c := range cases {
		got := synth.Synthesizable(c.v, c.dt)
		if got != c.want {
			t.Errorf("Synthesizable(%s, %s) = %v, want %v", c.v, c.dt, got, c.want)
		}
	}
}

// --- tiny helpers ---

func mustContain(t *testing.T, haystack, needle string) {
	t.Helper()
	if !strings.Contains(haystack, needle) {
		t.Errorf("walk missing expected substring: %q", needle)
	}
}

func itoa(n int) string {
	// Local itoa to avoid pulling strconv just for one test case.
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}
