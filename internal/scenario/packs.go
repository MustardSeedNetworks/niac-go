package scenario

const (
	scenarioPackManifestVersion = ManifestSchemaVersion
	hospitalSiteOctet           = 51
	warehouseSiteOctet          = 61
	campusSiteOctet             = 71
	retailSiteOctet             = 81
	industrialSiteOctet         = 91
	serviceProviderSiteOctet    = 101
)

type packSite struct {
	code     string
	location string
}

// MapPurpose classifies what a Pack's map is optimized to demonstrate:
// a small, presentable topology versus a scale/load stress case.
type MapPurpose string

const (
	// MapPurposePresentation marks a pack sized and laid out for demos and screenshots.
	MapPurposePresentation MapPurpose = "presentation"
	// MapPurposeStress marks a pack sized to exercise the simulator at scale.
	MapPurposeStress MapPurpose = "stress"
)

// Pack is one versioned composer preset with a frozen authored-truth manifest.
type Pack struct {
	ID              string     `json:"id"`
	Version         string     `json:"version"`
	ManifestVersion int        `json:"manifestVersion"`
	Name            string     `json:"name"`
	Description     string     `json:"description"`
	MapPurpose      MapPurpose `json:"mapPurpose"`
	Request         Request    `json:"request"`
	// Manifest pins the parity contract only. The richer manifest fields are
	// derived at generation time and would be meaningless frozen by hand.
	Manifest Parity `json:"manifest"`
}

// Packs returns copies of the built-in scenario composer presets.
func Packs() []Pack {
	definitions := scenarioPackDefinitions()
	result := make([]Pack, len(definitions))
	for index, pack := range definitions {
		result[index] = pack
		result[index].Request.Sites = append([]Site(nil), pack.Request.Sites...)
	}
	return result
}

func scenarioPackDefinitions() []Pack {
	return append(customerScenarioPacks(), verticalScenarioPacks()...)
}

func customerScenarioPacks() []Pack {
	return []Pack{
		hospitalScenarioPack(),
		newScenarioPack(
			"warehouse",
			"Warehouse network",
			"Fulfillment center covering a large open floor from a few closets, with "+
				"long-range Wi-Fi 7 radios, wired stations, local services, and redundant uplinks.",
			MapPurposePresentation,
			"fulfillment.example",
			packSites(warehouseSiteOctet,
				packSite{code: "FUL", location: "Regional Fulfillment Center"},
			),
			packCounts(
				warehouseAccessSwitches,
				warehouseAccessPointsPerAccess,
				warehouseWorkstationsPerAccess,
			),
			dockAccessPointPoELoss()...,
		),
		campusScenarioPack(),
		newScenarioPack(
			"enterprise-scale",
			"Enterprise scale reference",
			"Multi-site stress workload for discovery and scale testing; not intended as a presentation map.",
			MapPurposeStress,
			defaultDomain,
			EnterpriseReferenceRequest().Sites,
			EnterpriseReferenceRequest().Counts,
		),
	}
}

func packSites(firstOctet int, definitions ...packSite) []Site {
	sites := make([]Site, len(definitions))
	for index, definition := range definitions {
		sites[index] = Site{
			Code:     definition.code,
			Octet:    firstOctet + index,
			Location: definition.location,
		}
	}
	return sites
}

// hospitalScenarioPack is the guided demo, so it is the one pack that carries a
// story: the imaging closet saturates both of its uplinks, and both ends of
// both links report it. Everything else stays healthy, because a finding only
// reads as a finding when it is the exception on the map.
func hospitalScenarioPack() Pack {
	pack := newScenarioPack(
		"hospital",
		"Hospital network",
		"Medical center with resilient wired access, Wi-Fi 7 coverage, clinical clients, and local services.",
		MapPurposePresentation,
		"care.example",
		packSites(hospitalSiteOctet,
			packSite{code: "MED", location: "Regional Medical Center"},
		),
		packCounts(
			hospitalAccessSwitches,
			hospitalAccessPointsPerAccess,
			hospitalWorkstationsPerAccess,
		),
	)
	pack.Request.Faults = imagingSaturation()

	return pack
}

func imagingSaturation() []PackFault {
	const (
		saturated = 88
		busy      = 84
	)

	return []PackFault{
		{
			Device: "MED-ACC-SW02", Interface: "HundredGigabitEthernet1/0/49",
			Type: faultHighUtilization, Value: faultValue(saturated),
		},
		{
			Device: "MED-ACC-SW02", Interface: "HundredGigabitEthernet1/0/50",
			Type: faultHighUtilization, Value: faultValue(busy),
		},
		{
			Device: "MED-DIST-SW01", Interface: "HundredGigabitEthernet1/0/4",
			Type: faultHighUtilization, Value: faultValue(busy),
		},
		{
			Device: "MED-DIST-SW02", Interface: "HundredGigabitEthernet1/0/4",
			Type: faultHighUtilization, Value: faultValue(saturated),
		},
	}
}

// faultValue names the pointer so a pack reads as a sentence. A nil value is
// meaningful — it marks a condition with no magnitude — so the pointer cannot
// be dropped.
func faultValue(percent int) *int {
	return new(percent)
}

func campusScenarioPack() Pack {
	pack := newScenarioPack(
		"campus",
		"Enterprise campus",
		"Readable campus sites, each wide and shallow: closets land straight on a collapsed core, with Wi-Fi 7, workstation, and service layers.",
		MapPurposePresentation,
		"campus.example",
		packSites(campusSiteOctet,
			packSite{code: "NTH", location: "North Campus"},
			packSite{code: "STH", location: "South Campus"},
			packSite{code: "ENG", location: "Engineering Campus"},
			packSite{code: "ADM", location: "Administration Campus"},
		),
		campusCounts(),
		closetUplinkDiscards()...,
	)
	// A campus is wide and shallow; its closets land on the core directly.
	pack.Request.AccessLayer = AccessLayerCollapsedCore

	return pack
}

// newScenarioPack builds one preset and pins it to its frozen row in the
// generated parity table. A pack that names no row is a programming error, the
// same class as an unknown profile role, so it fails at construction rather
// than generating a scenario that silently promises nothing.
func newScenarioPack(
	id, name, description string,
	purpose MapPurpose,
	domain string,
	sites []Site,
	counts Counts,
	faults ...PackFault,
) Pack {
	manifest, pinned := packParity()[id]
	if !pinned {
		panic("unpinned scenario pack: " + id)
	}

	return Pack{
		ID: id, Version: "1.3.0", ManifestVersion: scenarioPackManifestVersion,
		Name: name, Description: description, MapPurpose: purpose,
		Request: Request{
			Sites: sites, Counts: counts, Domain: domain,
			SNMPCommunity: defaultCommunity, AttachmentName: defaultAttachmentName,
			EndpointProfile: packEndpointProfile(id),
			Faults:          faults,
		},
		Manifest: manifest,
	}
}

func packEndpointProfile(id string) string {
	if id == "campus" || id == "enterprise-scale" {
		return "enterprise"
	}
	return id
}
