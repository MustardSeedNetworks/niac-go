package scenario

const (
	scenarioPackManifestVersion = 2
	hospitalSiteOctet           = 51
	warehouseSiteOctet          = 61
	campusSiteOctet             = 71
	retailSiteOctet             = 81
	industrialSiteOctet         = 91
	serviceProviderSiteOctet    = 101

	hospitalDistributionSwitches    = 4
	hospitalAccessSwitches          = 12
	hospitalAccessPointsPerAccess   = 3
	hospitalWorkstationsPerAccess   = 4
	warehouseDistributionSwitches   = 2
	warehouseAccessSwitches         = 10
	warehouseAccessPointsPerAccess  = 4
	warehouseWorkstationsPerAccess  = 3
	campusDistributionSwitches      = 4
	campusAccessSwitches            = 16
	campusAccessPointsPerAccess     = 2
	campusWorkstationsPerAccess     = 4
	retailDistributionSwitches      = 2
	retailAccessSwitches            = 8
	retailAccessPointsPerAccess     = 4
	retailWorkstationsPerAccess     = 5
	industrialDistributionSwitches  = 4
	industrialAccessSwitches        = 12
	industrialAccessPointsPerAccess = 3
	industrialWorkstationsPerAccess = 3
	providerDistributionSwitches    = 8
	providerAccessSwitches          = 10
	providerServerSwitches          = 4
	providerAccessPointsPerAccess   = 3
	providerWorkstationsPerAccess   = 3
	hospitalDeviceCount             = 351
	hospitalNetworkCount            = 30
	hospitalLinkCount               = 416
	warehouseDeviceCount            = 297
	warehouseNetworkCount           = 30
	warehouseLinkCount              = 350
	campusDeviceCount               = 531
	campusNetworkCount              = 39
	campusLinkCount                 = 634
	retailDeviceCount               = 395
	retailNetworkCount              = 39
	retailLinkCount                 = 458
	industrialDeviceCount           = 315
	industrialNetworkCount          = 30
	industrialLinkCount             = 380
	serviceProviderDeviceCount      = 387
	serviceProviderNetworkCount     = 39
	serviceProviderLinkCount        = 490
)

type packSite struct {
	code     string
	location string
}

// Pack is one versioned composer preset with a frozen authored-truth manifest.
type Pack struct {
	ID              string   `json:"id"`
	Version         string   `json:"version"`
	ManifestVersion int      `json:"manifestVersion"`
	Name            string   `json:"name"`
	Description     string   `json:"description"`
	Request         Request  `json:"request"`
	Manifest        Manifest `json:"manifest"`
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
			"hospital", "Hospital network",
			"Acute-care, ambulatory, and clinical data-center sites with resilient wired and Wi-Fi access.",
			"care.example", packSites(hospitalSiteOctet,
				packSite{code: "MED", location: "Regional Medical Center"},
				packSite{code: "AMB", location: "Ambulatory Care Campus"},
				packSite{code: "CLD", location: "Clinical Data Center"},
			), packCounts(
				hospitalDistributionSwitches,
				hospitalAccessSwitches,
				maxRedundantPeers,
				hospitalAccessPointsPerAccess,
				hospitalWorkstationsPerAccess,
			), Manifest{
				DeviceCount: hospitalDeviceCount, NetworkCount: hospitalNetworkCount, LinkCount: hospitalLinkCount,
				DeviceNamesSHA256: "46da2595721bae0ef4a4d20ec3d95a12bb1b3e09be71e618cebecbf7ff59860b",
				NetworksSHA256:    "3903357b0345fa756d65a537dffd67dfbf361ad414305a99137762e26f4816a4",
				LinksSHA256:       "83b19407c072afdb45cd9665d7c9c76975dde503626f423f0f40b70a587e1e28",
			},
		),
		newScenarioPack(
			"warehouse",
			"Warehouse network",
			"Three fulfillment centers with dense scanner coverage, wired stations, local services, and redundant uplinks.",
			"fulfillment.example",
			packSites(warehouseSiteOctet,
				packSite{code: "EAS", location: "Eastern Fulfillment Center"},
				packSite{code: "CEN", location: "Central Fulfillment Center"},
				packSite{code: "WES", location: "Western Fulfillment Center"},
			),
			packCounts(
				warehouseDistributionSwitches,
				warehouseAccessSwitches,
				maxRedundantPeers,
				warehouseAccessPointsPerAccess,
				warehouseWorkstationsPerAccess,
			),
			Manifest{
				DeviceCount: warehouseDeviceCount, NetworkCount: warehouseNetworkCount, LinkCount: warehouseLinkCount,
				DeviceNamesSHA256: "19e88c9625d3d38c2a864786ab6f76d43a23f7d6723ae9af014b64eb4c106bc9",
				NetworksSHA256:    "2647fd6b5e33c0d7177be982854a7571d8d173d64677b6177c33afb5a7607aac",
				LinksSHA256:       "22836ae0988c218259b60c8e94e55b4ad61d2beff5e179f47da1a6d93e75a676",
			},
		),
		newScenarioPack(
			"campus", "Enterprise campus",
			"Four large buildings with core, distribution, access, Wi-Fi 7, workstation, and service layers.",
			"campus.example", packSites(campusSiteOctet,
				packSite{code: "NTH", location: "North Campus"},
				packSite{code: "STH", location: "South Campus"},
				packSite{code: "ENG", location: "Engineering Campus"},
				packSite{code: "ADM", location: "Administration Campus"},
			), packCounts(
				campusDistributionSwitches,
				campusAccessSwitches,
				maxRedundantPeers,
				campusAccessPointsPerAccess,
				campusWorkstationsPerAccess,
			), Manifest{
				DeviceCount: campusDeviceCount, NetworkCount: campusNetworkCount, LinkCount: campusLinkCount,
				DeviceNamesSHA256: "ff1862a05edded994e0d6ca5962a0f602c052f24002260ddd1becc2f89964947",
				NetworksSHA256:    "7262a118fbb0f2d4977d02895b839d0cbce5fd1161201b0f27a5b37fc3eb72ce",
				LinksSHA256:       "ee50d491a344adf6f7069de069c8d7b0be91c222505244aff7c9ce34afbb37c2",
			},
		),
	}
}

func verticalScenarioPacks() []Pack {
	return []Pack{
		newScenarioPack(
			"retail", "Retail network",
			"Headquarters and three retail regions with wireless point-of-sale coverage and local business services.",
			"retail.example", packSites(retailSiteOctet,
				packSite{code: "HQ", location: "Retail Headquarters"},
				packSite{code: "NER", location: "Northeast Retail Region"},
				packSite{code: "SER", location: "Southeast Retail Region"},
				packSite{code: "WER", location: "Western Retail Region"},
			), packCounts(
				retailDistributionSwitches,
				retailAccessSwitches,
				maxRedundantPeers,
				retailAccessPointsPerAccess,
				retailWorkstationsPerAccess,
			), Manifest{
				DeviceCount: retailDeviceCount, NetworkCount: retailNetworkCount, LinkCount: retailLinkCount,
				DeviceNamesSHA256: "972c99c87edd3fc07c128b19e2347d8692011605ef2fad7495b07a779495ca24",
				NetworksSHA256:    "ff97ea73b175dd19168b1500d97165a5f44ee9d4f7337db67edc700a0ba0a829",
				LinksSHA256:       "108edbda094f681ea78f25912f26375ffbd6742cc7ec7a4fb5e35ff06917ccf6",
			},
		),
		newScenarioPack(
			"industrial",
			"Industrial network",
			"Two production plants and an engineering lab with resilient switching, wired stations, and Wi-Fi 7 coverage.",
			"industrial.example",
			packSites(industrialSiteOctet,
				packSite{code: "PLT1", location: "Production Plant One"},
				packSite{code: "PLT2", location: "Production Plant Two"},
				packSite{code: "LAB", location: "Industrial Engineering Lab"},
			),
			packCounts(
				industrialDistributionSwitches,
				industrialAccessSwitches,
				maxRedundantPeers,
				industrialAccessPointsPerAccess,
				industrialWorkstationsPerAccess,
			),
			Manifest{
				DeviceCount:       industrialDeviceCount,
				NetworkCount:      industrialNetworkCount,
				LinkCount:         industrialLinkCount,
				DeviceNamesSHA256: "6cc2ddcaa3671c6791b13c569ba04ef49c26a9b92a3b86c0c5a01ecef3d426f2",
				NetworksSHA256:    "0259604d06e92b16ef8d4d1d6a2b0d9893963b23f936f7b65c7abd11562bc4e0",
				LinksSHA256:       "6f0ae7af17e5a726169f835936c11f621af7e61371e45f28bc3f841b78b87d80",
			},
		),
		newScenarioPack(
			"service-provider", "Service provider network",
			"Four metro points of presence with deep distribution, access, service, wired-client, and wireless layers.",
			"provider.example", packSites(serviceProviderSiteOctet,
				packSite{code: "NYC", location: "New York Metro POP"},
				packSite{code: "CHI", location: "Chicago Metro POP"},
				packSite{code: "DEN", location: "Denver Metro POP"},
				packSite{code: "SFO", location: "San Francisco Metro POP"},
			), packCounts(
				providerDistributionSwitches,
				providerAccessSwitches,
				providerServerSwitches,
				providerAccessPointsPerAccess,
				providerWorkstationsPerAccess,
			), Manifest{
				DeviceCount:       serviceProviderDeviceCount,
				NetworkCount:      serviceProviderNetworkCount,
				LinkCount:         serviceProviderLinkCount,
				DeviceNamesSHA256: "c799d4b5fb95310c477721262a80043ed67e55785ea02e28f64d550d566bdb7c",
				NetworksSHA256:    "95a2b3c773f6492777efba084054781170ec2b9b01745308bd7f061e658d0bd4",
				LinksSHA256:       "3afe4b8ed0af88ade74971972a853a1a3bb6b6af54600bc6836c5279bbfc750c",
			},
		),
	}
}

func packSites(firstOctet int, definitions ...packSite) []Site {
	sites := make([]Site, len(definitions))
	for index, definition := range definitions {
		sites[index] = Site{Code: definition.code, Octet: firstOctet + index, Location: definition.location}
	}
	return sites
}

func packCounts(distribution, access, server, accessPoints, workstations int) Counts {
	return Counts{
		SiteWANRouters: maxRedundantPeers, Firewalls: maxRedundantPeers, CoreSwitches: maxRedundantPeers,
		DistributionSwitches: distribution, AccessSwitches: access, ServerSwitches: server,
		AccessPointsPerAccess: accessPoints, WorkstationsPerAccess: workstations,
		WirelessControllers: maxRedundantPeers,
	}
}

func newScenarioPack(
	id, name, description, domain string,
	sites []Site,
	counts Counts,
	manifest Manifest,
) Pack {
	return Pack{
		ID: id, Version: "1.1.0", ManifestVersion: scenarioPackManifestVersion,
		Name: name, Description: description,
		Request: Request{
			Sites: sites, Counts: counts, Domain: domain,
			SNMPCommunity: defaultCommunity, AttachmentName: defaultAttachmentName,
		},
		Manifest: manifest,
	}
}
