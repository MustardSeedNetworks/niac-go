package scenario

const (
	scenarioPackManifestVersion = ManifestSchemaVersion
	hospitalSiteOctet           = 51
	warehouseSiteOctet          = 61
	campusSiteOctet             = 71
	retailSiteOctet             = 81
	industrialSiteOctet         = 91
	serviceProviderSiteOctet    = 101
	hospitalDeviceCount         = 78
	warehouseDeviceCount        = 60
	manufacturingDeviceCount    = 72
	campusDeviceCount           = 159
	enterpriseScaleDeviceCount  = 543
	retailDeviceCount           = 101
	serviceProviderDeviceCount  = 126
	singleSiteNetworkCount      = 12
	twoSiteNetworkCount         = 21
	threeSiteNetworkCount       = 30
	fourSiteNetworkCount        = 39
	hospitalLinkCount           = 88
	warehouseLinkCount          = 67
	manufacturingLinkCount      = 78
	campusLinkCount             = 186
	enterpriseScaleLinkCount    = 634
	retailLinkCount             = 112
	serviceProviderLinkCount    = 146
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
			"Single-site fulfillment center with 30 Wi-Fi 7 APs, wired stations, local services, and redundant uplinks.",
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
			Parity{
				DeviceCount: warehouseDeviceCount, NetworkCount: singleSiteNetworkCount,
				LinkCount:         warehouseLinkCount,
				DeviceNamesSHA256: "3c76bcb87bcc9df0701d2b2c34bb5e671832e2f28089ab4d7dedc0c44b17717b",
				NetworksSHA256:    "4b45bbf256fb1440d30d2149f3691664404012f192d5f98f788bcc0c413e90b5",
				LinksSHA256:       "a495dcb76177b01348573b330b54640318d21c34f7c71e596de2aba0bb8c9939",
			},
		),
		campusScenarioPack(),
		newScenarioPack(
			"enterprise-scale",
			"Enterprise scale reference",
			"Four-site, 531-device stress workload for discovery and scale testing; not intended as a presentation map.",
			MapPurposeStress,
			defaultDomain,
			EnterpriseReferenceRequest().Sites,
			EnterpriseReferenceRequest().Counts,
			Parity{
				DeviceCount: enterpriseScaleDeviceCount, NetworkCount: fourSiteNetworkCount,
				LinkCount:         enterpriseScaleLinkCount,
				DeviceNamesSHA256: "8514a6d423b598a11d6ebc6edfc399c978883b831106e8c187e681619229346f",
				NetworksSHA256:    "e879b7ba38e40f925809edc3bf98d2044959df5d2f76d492e6f2019cbcba5555",
				LinksSHA256:       "4c1acbf07eccc6464a4a86d8a53f867fdaa1cc7374d18b881bf30487a98713e6",
			},
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
		"Single-site medical center with resilient wired access, 30 Wi-Fi 7 APs, clinical clients, and local services.",
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
		Parity{
			DeviceCount: hospitalDeviceCount, NetworkCount: singleSiteNetworkCount,
			LinkCount:         hospitalLinkCount,
			DeviceNamesSHA256: "93d15a9fe811e623d831d3987909cdd35642e25e3e0afb48fa4be219aeabb426",
			NetworksSHA256:    "af29ba1bf3ae3a58f46809ba0e126fa436ea4e78193842f8ce12b9d276686b30",
			LinksSHA256:       "99be6cdbe704f4e4d661a27be11b6294e62e8b52ca83e2c6198b4ba9fe8836b2",
		},
	)
	pack.Request.Congestion = imagingCongestion()

	return pack
}

func imagingCongestion() []CongestedLink {
	const (
		saturated = 88.0
		busy      = 84.0
	)

	return []CongestedLink{
		{
			Device: "MED-ACC-SW02", Interface: "HundredGigabitEthernet1/0/49",
			InUtilization: saturated, OutUtilization: busy,
		},
		{
			Device: "MED-ACC-SW02", Interface: "HundredGigabitEthernet1/0/50",
			InUtilization: busy, OutUtilization: saturated,
		},
		{
			Device: "MED-DIST-SW01", Interface: "HundredGigabitEthernet1/0/4",
			InUtilization: busy, OutUtilization: saturated,
		},
		{
			Device: "MED-DIST-SW02", Interface: "HundredGigabitEthernet1/0/4",
			InUtilization: saturated, OutUtilization: busy,
		},
	}
}

func campusScenarioPack() Pack {
	pack := newScenarioPack(
		"campus",
		"Enterprise campus",
		"Four readable sites, each wide and shallow: closets land straight on a collapsed core, with Wi-Fi 7, workstation, and service layers.",
		MapPurposePresentation,
		"campus.example",
		packSites(campusSiteOctet,
			packSite{code: "NTH", location: "North Campus"},
			packSite{code: "STH", location: "South Campus"},
			packSite{code: "ENG", location: "Engineering Campus"},
			packSite{code: "ADM", location: "Administration Campus"},
		),
		campusCounts(),
		Parity{
			DeviceCount: campusDeviceCount, NetworkCount: fourSiteNetworkCount,
			LinkCount:         campusLinkCount,
			DeviceNamesSHA256: "e67474b172037c2b38c1b74c4a48c0c2a2fa1d9cb2d3201226a8187916edf243",
			NetworksSHA256:    "7262a118fbb0f2d4977d02895b839d0cbce5fd1161201b0f27a5b37fc3eb72ce",
			LinksSHA256:       "faa6df268e4654e542f89b707ee9bc05744de70b6edc9419f990e77da478410d",
		},
	)
	// A campus is wide and shallow; its closets land on the core directly.
	pack.Request.AccessLayer = AccessLayerCollapsedCore

	return pack
}

func newScenarioPack(
	id, name, description string,
	purpose MapPurpose,
	domain string,
	sites []Site,
	counts Counts,
	manifest Parity,
) Pack {
	return Pack{
		ID: id, Version: "1.3.0", ManifestVersion: scenarioPackManifestVersion,
		Name: name, Description: description, MapPurpose: purpose,
		Request: Request{
			Sites: sites, Counts: counts, Domain: domain,
			SNMPCommunity: defaultCommunity, AttachmentName: defaultAttachmentName,
			EndpointProfile: packEndpointProfile(id),
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
