package scenario_test

import (
	"net"
	"strings"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/scenario"
)

// The realism doc's server tier: six service servers are the infrastructure
// floor, and a real facility also runs the system its vertical depends on. The
// generic APP server is that slot, so each vertical names it for what it runs;
// campus stays deliberately ordinary.
func TestEachVerticalRunsItsWorkloadServer(t *testing.T) {
	want := map[string]struct{ name, sysDescr string }{
		"hospital":         {"PACS01", "PACS archive"},
		"manufacturing":    {"HIST01", "SCADA historian"},
		"warehouse":        {"WMS01", "warehouse management system"},
		"retail":           {"BOS01", "retail back-office server"},
		"service-provider": {"PROV01", "subscriber provisioning server"},
		"campus":           {"APP01", "APP service"},
	}

	// enterprise-scale shares campus's profile and is most of this package's
	// race budget (#2349), so campus stands for both.
	for _, pack := range scenario.Packs() {
		expected, ok := want[pack.ID]
		if !ok {
			continue
		}
		cfg := packConfig(t, pack)
		for _, site := range pack.Request.Sites {
			assertWorkloadServer(t, cfg, pack, site.Code, expected.name, expected.sysDescr)
		}
	}
}

func assertWorkloadServer(t *testing.T, cfg *config.Config, pack scenario.Pack, siteCode, suffix, sysDescr string) {
	t.Helper()

	name := siteCode + "-" + suffix
	device := findDevice(cfg, name)
	if device == nil {
		t.Errorf("%s: missing %s", pack.ID, name)
		return
	}
	if device.Type != "server" {
		t.Errorf("%s type = %q, want server", name, device.Type)
	}
	if !strings.HasSuffix(device.SNMPConfig.SysDescr, " "+sysDescr) {
		t.Errorf("%s sysDescr = %q, want it to end with %q", name, device.SNMPConfig.SysDescr, sysDescr)
	}
	if !resolvesTo(findDevice(cfg, siteCode+"-DNS01"), strings.ToLower(name)+"."+pack.Request.Domain,
		device.IPAddresses) {
		t.Errorf("%s: site DNS has no record for %s at its address", pack.ID, name)
	}
	if suffix != "APP01" && findDevice(cfg, siteCode+"-APP01") != nil {
		t.Errorf("%s: %s-APP01 still present beside its workload server", pack.ID, siteCode)
	}
}

func resolvesTo(dnsServer *config.Device, name string, addresses []net.IP) bool {
	if dnsServer == nil || dnsServer.DNSConfig == nil || len(addresses) == 0 {
		return false
	}
	for _, record := range dnsServer.DNSConfig.ForwardRecords {
		if record.Name == name && record.IP.Equal(addresses[0]) {
			return true
		}
	}

	return false
}
