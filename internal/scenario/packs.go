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
	compactVerticalDeviceCount  = 69
	campusDeviceCount           = 155
	enterpriseScaleDeviceCount  = 531
	retailDeviceCount           = 95
	serviceProviderDeviceCount  = 87
	singleSiteNetworkCount      = 12
	twoSiteNetworkCount         = 21
	threeSiteNetworkCount       = 30
	fourSiteNetworkCount        = 39
	hospitalLinkCount           = 88
	compactVerticalLinkCount    = 82
	campusLinkCount             = 202
	enterpriseScaleLinkCount    = 634
	retailLinkCount             = 118
	serviceProviderLinkCount    = 122
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
				DeviceCount: compactVerticalDeviceCount, NetworkCount: singleSiteNetworkCount,
				LinkCount:         compactVerticalLinkCount,
				DeviceNamesSHA256: "5e58dc80d18034ef8e07091a98bd2acda6c3507b1c2c12d3671b9d08aaac5044",
				NetworksSHA256:    "4b45bbf256fb1440d30d2149f3691664404012f192d5f98f788bcc0c413e90b5",
				LinksSHA256:       "b7e277f9b64855164d8b0dd7f2e02acc7d6db091da8b5f23fb95378aa801a729",
			},
		),
		newScenarioPack(
			"campus",
			"Enterprise campus",
			"Four readable sites with core, distribution, access, Wi-Fi 7, workstation, and service layers.",
			MapPurposePresentation,
			"campus.example",
			packSites(campusSiteOctet,
				packSite{code: "NTH", location: "North Campus"},
				packSite{code: "STH", location: "South Campus"},
				packSite{code: "ENG", location: "Engineering Campus"},
				packSite{code: "ADM", location: "Administration Campus"},
			),
			packCounts(
				campusAccessSwitches,
				campusAccessPointsPerAccess,
				campusWorkstationsPerAccess,
			),
			Manifest{
				DeviceCount: campusDeviceCount, NetworkCount: fourSiteNetworkCount,
				LinkCount:         campusLinkCount,
				DeviceNamesSHA256: "17ed0a70d1f95060126e399c19b0a9fc4fa2ad5779be8ac8763debd34f97b72f",
				NetworksSHA256:    "7262a118fbb0f2d4977d02895b839d0cbce5fd1161201b0f27a5b37fc3eb72ce",
				LinksSHA256:       "b171d72805e3ea4524118adcf58a99c8e52112c8d32f43f25816b90b3ea2e3e8",
			},
		),
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
