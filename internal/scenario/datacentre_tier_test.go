package scenario_test

import (
	"strings"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/scenario"
)

type dataCentreDevice struct {
	role, sysObjectID, sysDescr, httpServer string
}

// The realism doc's shared server tier: a vertical's workload systems run on a
// virtualisation host, keep their data on a storage array and are protected by
// a backup server. An organization keeps that tier in its main building, so it
// appears once per pack, at the first site, and each device answers with the
// sysObjectID and sysDescr a discovery tool files its platform by.
func TestEachVerticalRunsADataCentreTierAtItsFirstSite(t *testing.T) {
	tier := map[string]dataCentreDevice{
		"ESX01": {"hypervisor", "1.3.6.1.4.1.6876.4.1", "VMware ESXi ", ""},
		"SAN01": {"storage-array", "1.3.6.1.4.1.789.2.5", "NetApp Release ", "libzapid-httpd"},
		"BKP01": {"backup-server", "1.3.6.1.4.1.311.1.1.3.1.2", "Hardware: ", "Microsoft-IIS/10.0"},
	}

	// enterprise-scale shares campus's profile and is most of this package's
	// race budget (#2349), so campus stands for both.
	for _, pack := range scenario.Packs() {
		if pack.ID == "enterprise-scale" {
			continue
		}
		cfg := packConfig(t, pack)
		for index, site := range pack.Request.Sites {
			for suffix, want := range tier {
				name := site.Code + "-" + suffix
				device := findDevice(cfg, name)
				if index > 0 || pack.ID == "campus" {
					if device != nil {
						t.Errorf("%s: unexpected %s", pack.ID, name)
					}
					continue
				}
				assertDataCentreDevice(t, cfg, pack, name, device, want)
			}
		}
	}
}

func assertDataCentreDevice(
	t *testing.T, cfg *config.Config, pack scenario.Pack, name string, device *config.Device, want dataCentreDevice,
) {
	t.Helper()

	if device == nil {
		t.Errorf("%s: missing %s", pack.ID, name)
		return
	}
	if device.Type != "server" || device.Properties["role"] != want.role {
		t.Errorf("%s type/role = %q/%q, want server/%s", name, device.Type, device.Properties["role"], want.role)
	}
	checkAnswersAs(t, pack.ID, device, sysObjectIDOID, want.sysObjectID)
	checkAnswersAs(t, pack.ID, device, sysDescrOID, want.sysDescr)
	switch {
	case want.httpServer == "" && device.HTTPConfig != nil:
		t.Errorf("%s serves HTTP, want none", name)
	case want.httpServer != "" && (device.HTTPConfig == nil || device.HTTPConfig.ServerName != want.httpServer):
		t.Errorf("%s HTTP config = %+v, want Server %q", name, device.HTTPConfig, want.httpServer)
	}
	if !resolvesTo(findDevice(cfg, pack.Request.Sites[0].Code+"-DNS01"),
		strings.ToLower(name)+"."+pack.Request.Domain, device.IPAddresses) {
		t.Errorf("%s: site DNS has no record for %s at its address", pack.ID, name)
	}
}
