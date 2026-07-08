package sanitize_test

import (
	"strings"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/sanitize"
)

func TestContentDeterministic(t *testing.T) {
	content := []byte(`SNMPv2-MIB::sysName.0 = STRING: test-switch
SNMPv2-MIB::sysContact.0 = STRING: admin@test.com
.1.3.6.1.2.1.4.20.1.1.192.168.1.1 = IpAddress: 192.168.1.1
`)
	opts := sanitize.DefaultOptions()

	out1, stats1, err := sanitize.Content(content, nil, opts)
	if err != nil {
		t.Fatalf("Content() error = %v", err)
	}
	out2, stats2, err := sanitize.Content(content, nil, opts)
	if err != nil {
		t.Fatalf("Content() error = %v", err)
	}

	if string(out1) != string(out2) {
		t.Errorf("Content() not deterministic across separate calls:\n%q\nvs\n%q", out1, out2)
	}
	if stats1 != stats2 {
		t.Errorf("Stats not deterministic: %+v vs %+v", stats1, stats2)
	}
	if stats1.IPsTransformed == 0 {
		t.Error("expected at least one IP transformed")
	}
	if stats1.HostnamesTransformed == 0 {
		t.Error("expected at least one hostname transformed")
	}
}

func TestContentSameMappingAcrossCalls(t *testing.T) {
	line1 := []byte(".1.3.6.1.2.1.4.20.1.1.192.168.1.1 = IpAddress: 192.168.1.1\n")
	line2 := []byte(".1.3.6.1.2.1.4.20.1.1.192.168.1.1 = IpAddress: 192.168.1.1\n")

	mapping := sanitize.NewMapping()
	opts := sanitize.DefaultOptions()

	out1, _, err := sanitize.Content(line1, mapping, opts)
	if err != nil {
		t.Fatalf("Content() error = %v", err)
	}
	out2, _, err := sanitize.Content(line2, mapping, opts)
	if err != nil {
		t.Fatalf("Content() error = %v", err)
	}

	if string(out1) != string(out2) {
		t.Errorf("same IP across calls with shared mapping produced different output:\n%q\nvs\n%q", out1, out2)
	}
	// The second call sees the IP already mapped, so it contributes no new
	// transformation — mapping.Statistics accumulates only the first hit.
	if mapping.Statistics.IPsTransformed != 1 {
		t.Errorf("cumulative IPsTransformed = %d, want 1", mapping.Statistics.IPsTransformed)
	}
}

func TestContentKeepsNonSensitiveFieldsUntouched(t *testing.T) {
	content := []byte(`.1.3.6.1.2.1.2.2.1.6.1 = STRING: "aa:bb:cc:dd:ee:ff"
.1.3.6.1.2.1.47.1.1.1.1.13.1 = STRING: "WS-C3850-48P"
.1.3.6.1.2.1.2.2.1.1.10 = INTEGER: 10
`)
	out, _, err := sanitize.Content(content, nil, sanitize.DefaultOptions())
	if err != nil {
		t.Fatalf("Content() error = %v", err)
	}
	got := string(out)
	for _, want := range []string{"aa:bb:cc:dd:ee:ff", "WS-C3850-48P", ".1.3.6.1.2.1.2.2.1.1.10 = INTEGER: 10"} {
		if !strings.Contains(got, want) {
			t.Errorf("expected kept field %q in output, got:\n%s", want, got)
		}
	}
}

func TestContentNilMapping(t *testing.T) {
	out, stats, err := sanitize.Content([]byte("plain text\n"), nil, sanitize.DefaultOptions())
	if err != nil {
		t.Fatalf("Content() error = %v", err)
	}
	if string(out) != "plain text\n" {
		t.Errorf("Content() = %q, want unchanged", out)
	}
	if stats.IPsTransformed != 0 || stats.HostnamesTransformed != 0 {
		t.Errorf("stats = %+v, want zero", stats)
	}
}

func TestContentLocationAndCommunity(t *testing.T) {
	content := []byte(`SNMPv2-MIB::sysLocation.0 = STRING: Building A, Floor 2
SNMPv2-MIB::snmpCommunity.1 = STRING: secretCommunity
`)
	out, _, err := sanitize.Content(content, nil, sanitize.Options{
		Domain: "niac-go.com", Location: "DC-EAST", Contact: "netadmin@niac-go.com", Community: "public",
	})
	if err != nil {
		t.Fatalf("Content() error = %v", err)
	}
	got := string(out)
	if !strings.Contains(got, "NiAC-Go - DC-EAST - Network Operations") {
		t.Errorf("location not branded: %s", got)
	}
	if strings.Contains(got, "secretCommunity") {
		t.Errorf("community string not scrubbed: %s", got)
	}
	if !strings.Contains(got, "= STRING: public") {
		t.Errorf("community not replaced with default: %s", got)
	}
}
