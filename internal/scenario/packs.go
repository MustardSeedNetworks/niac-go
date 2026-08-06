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
				DeviceNamesSHA256: "b808be4a818b9a3e3dffbaa57448f12d9939e50f2c0b297e49a96dc8f6fb33b3",
				NetworksSHA256:    "af29ba1bf3ae3a58f46809ba0e126fa436ea4e78193842f8ce12b9d276686b30",
				LinksSHA256:       "30459dc94c17785cbc4c9394af16ed2820d2b089d84bf244fb95dd8fd01e8a9c",
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
				DeviceNamesSHA256: "a6c7bf51ab462c09992b8f40d3c9cf7d6580d487bd3c67d4b65157b22c35781c",
				NetworksSHA256:    "7262a118fbb0f2d4977d02895b839d0cbce5fd1161201b0f27a5b37fc3eb72ce",
				LinksSHA256:       "1393d2beae5a030ae96b8b1a295a8b5001417bc5bea114408cc7945902fe7a6f",
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
				DeviceNamesSHA256: "1286a7b93d0c7185189db240c88d6d400d4deb7dbcc361f1384c4a176bcfce0d",
				NetworksSHA256:    "e879b7ba38e40f925809edc3bf98d2044959df5d2f76d492e6f2019cbcba5555",
				LinksSHA256:       "ac59d25316d4c1e422199e0dc8d4a1f14b8ca220f09992583ea860bf074dc85f",
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
