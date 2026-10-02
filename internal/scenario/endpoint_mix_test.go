package scenario_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/protocols/snmp"
	"github.com/MustardSeedNetworks/niac-go/internal/scenario"
)

// apcNetworkManagementCard is the sysObjectID the SNMP agent keys UPS-MIB on.
const (
	apcNetworkManagementCard = "1.3.6.1.4.1.318.1.3.27"
	upsMIB                   = "1.3.6.1.2.1.33"
)

func endpointRoleCounts(t *testing.T, request scenario.Request) (map[string]map[string]int, *config.Config) {
	t.Helper()

	cfg := generatedRequest(t, "endpoint mix", request).Config
	counts := map[string]map[string]int{}
	for index := range cfg.Devices {
		device := &cfg.Devices[index]
		site := device.Properties["site"]
		if counts[site] == nil {
			counts[site] = map[string]int{}
		}
		counts[site][device.Properties["role"]]++
	}

	return counts, cfg
}

// The realism doc: "A hospital runs dozens of infusion pumps and a single MRI."
// A per-site rotation repeats its signature devices with the endpoint count,
// so the ~160-endpoint hospital P-PACK-1 builds would have had eighteen MRIs.
// 64 is as large as a site goes before that resize lifts the per-site cap.
func TestHospitalMixKeepsSignatureDevicesRareAtScale(t *testing.T) {
	request := scenario.EnterpriseReferenceRequest()
	request.Sites = request.Sites[:1]
	request.EndpointProfile = "hospital"
	request.Counts.AccessSwitches = 16
	request.Counts.WorkstationsPerAccess = 4

	counts, _ := endpointRoleCounts(t, request)
	site := counts[request.Sites[0].Code]
	if got := site["mr-system"]; got != 1 {
		t.Errorf("MRI systems = %d, want 1", got)
	}
	if got := site["ups"]; got != 1 {
		t.Errorf("UPS = %d, want 1", got)
	}
	pumps := site["infusion-pump"]
	for role, count := range site {
		if role != "infusion-pump" && isWiredEndpoint(role) && count >= pumps {
			t.Errorf("%s = %d, want fewer than the %d infusion pumps", role, count, pumps)
		}
	}
}

func isWiredEndpoint(role string) bool {
	switch role {
	case "nurse-station", "philips-patient-monitor", "ge-patient-monitor", "label-printer", "mr-system", "ups",
		"nas", "pdu", "badge-controller", "conference-room":
		return true
	default:
		return false
	}
}

// Every site carries one UPS, and it reports the APC card's sysObjectID so the
// agent answers UPS-MIB for it. Without that property the pack's UPS would be
// a generic host that happens to be named like one, answering nothing under
// UPS-MIB.
func TestEveryPackSiteCarriesAUPSThatAnswersUPSMIB(t *testing.T) {
	for _, pack := range scenario.Packs() {
		counts, cfg := endpointRoleCounts(t, pack.Request)
		for _, site := range pack.Request.Sites {
			if got := counts[site.Code]["ups"]; got != 1 {
				t.Errorf("%s %s: UPS = %d, want 1", pack.ID, site.Code, got)
			}
		}
		for index := range cfg.Devices {
			if cfg.Devices[index].Properties["role"] == "ups" {
				checkPackUPS(t, pack.ID, &cfg.Devices[index])
			}
		}
	}
}

func checkPackUPS(t *testing.T, pack string, device *config.Device) {
	t.Helper()

	if got := device.Properties["sysObjectID"]; got != apcNetworkManagementCard {
		t.Errorf("%s %s: sysObjectID = %q, want %q", pack, device.Name, got, apcNetworkManagementCard)
	}
	if device.SNMPConfig.SysName == "" {
		t.Errorf("%s %s: UPS has no SNMP agent", pack, device.Name)
		return
	}
	next, _, err := snmp.NewAgent(device, 0).HandleGetNext(upsMIB)
	if err != nil || !strings.HasPrefix(next, upsMIB+".") {
		t.Errorf("%s %s: GETNEXT %s = %q, %v; want a UPS-MIB object", pack, device.Name, upsMIB, next, err)
	}
}

// The common tier's storage and meeting-room devices answer SNMP as
// themselves: each site's one NAS reports a Synology DiskStation, and a room
// system reports the Cisco codec arc a discovery tool files as a
// collaboration endpoint. Without their profiles they would be hosts named
// like a NAS or a room.
func TestEveryPackSiteCarriesItsNASAndRoomSystems(t *testing.T) {
	for _, pack := range scenario.Packs() {
		counts, cfg := endpointRoleCounts(t, pack.Request)
		for _, site := range pack.Request.Sites {
			if got := counts[site.Code]["nas"]; got != 1 {
				t.Errorf("%s %s: NAS = %d, want 1", pack.ID, site.Code, got)
			}
			if counts[site.Code]["conference-room"] == 0 {
				t.Errorf("%s %s: no conference room", pack.ID, site.Code)
			}
		}
		for index := range cfg.Devices {
			device := &cfg.Devices[index]
			switch device.Properties["role"] {
			case "nas":
				checkAnswersAs(t, pack.ID, device, sysDescrOID, "Synology DiskStation")
			case "conference-room":
				checkAnswersAs(t, pack.ID, device, sysObjectIDOID, ciscoCodec)
			}
		}
	}
}

const (
	sysDescrOID    = "1.3.6.1.2.1.1.1.0"
	sysObjectIDOID = "1.3.6.1.2.1.1.2.0"
	ciscoCodec     = "1.3.6.1.4.1.5596.150.6.4.1"
)

func checkAnswersAs(t *testing.T, pack string, device *config.Device, oid, want string) {
	t.Helper()

	value, err := snmp.NewAgent(device, 0).HandleGet(oid)
	if err != nil || value == nil {
		t.Errorf("%s %s: GET %s = %v, %v", pack, device.Name, oid, value, err)
		return
	}
	if got := strings.TrimPrefix(fmt.Sprint(value.Value), "."); !strings.Contains(got, want) {
		t.Errorf("%s %s: GET %s = %q, want it to carry %q", pack, device.Name, oid, got, want)
	}
}
