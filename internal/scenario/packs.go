package scenario

const (
	scenarioPackManifestVersion = 3
	hospitalSiteOctet           = 51
	warehouseSiteOctet          = 61
	campusSiteOctet             = 71
	retailSiteOctet             = 81
	industrialSiteOctet         = 91
	serviceProviderSiteOctet    = 101
	hospitalDeviceCount         = 75
	warehouseDeviceCount        = 57
	manufacturingDeviceCount    = 69
	campusDeviceCount           = 147
	enterpriseScaleDeviceCount  = 531
	retailDeviceCount           = 95
	serviceProviderDeviceCount  = 117
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

type MapPurpose string

const (
	MapPurposePresentation MapPurpose = "presentation"
	MapPurposeStress       MapPurpose = "stress"
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
	Manifest        Manifest   `json:"manifest"`
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
		newScenarioPack(
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
			Manifest{
				DeviceCount: hospitalDeviceCount, NetworkCount: singleSiteNetworkCount,
				LinkCount:         hospitalLinkCount,
				DeviceNamesSHA256: "a026920162b3dcc95656a4d3d69a0aeed84482e7724b018d5a49488383609030",
				NetworksSHA256:    "af29ba1bf3ae3a58f46809ba0e126fa436ea4e78193842f8ce12b9d276686b30",
				LinksSHA256:       "99be6cdbe704f4e4d661a27be11b6294e62e8b52ca83e2c6198b4ba9fe8836b2",
			},
		),
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
			Manifest{
				DeviceCount: warehouseDeviceCount, NetworkCount: singleSiteNetworkCount,
				LinkCount:         warehouseLinkCount,
				DeviceNamesSHA256: "f622b9683718ece6b665f88c858df863b6b7d54d8a4371efdc9b20cd79255b01",
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
			Manifest{
				DeviceCount: enterpriseScaleDeviceCount, NetworkCount: fourSiteNetworkCount,
				LinkCount:         enterpriseScaleLinkCount,
				DeviceNamesSHA256: "6da20f0044fb2f696efc3d886c209eb47c07b5fa1e64222f93d58f6dc00f1979",
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
		Manifest{
			DeviceCount: campusDeviceCount, NetworkCount: fourSiteNetworkCount,
			LinkCount:         campusLinkCount,
			DeviceNamesSHA256: "c15f6f9ddff32dc30e5751858c3ea2650dd2aa1b2900abf89b5f9b23692236f7",
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
	manifest Manifest,
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
