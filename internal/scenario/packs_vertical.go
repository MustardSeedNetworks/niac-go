package scenario

func verticalScenarioPacks() []Pack {
	return []Pack{
		retailScenarioPack(),
		manufacturingScenarioPack(),
		serviceProviderScenarioPack(),
	}
}

func retailScenarioPack() Pack {
	pack := newScenarioPack(
		"retail", "Retail network",
		"Two retail sites: a back office, and a store whose lanes chain off one another "+
			"with wireless point-of-sale coverage and local business services.",
		MapPurposePresentation,
		"retail.example", packSites(retailSiteOctet,
			packSite{code: "HQ", location: "Retail Headquarters"},
			packSite{code: "STR", location: "Regional Flagship Store"},
		), packCounts(
			retailAccessSwitches,
			retailAccessPointsPerAccess,
			retailWorkstationsPerAccess,
		), Manifest{
			DeviceCount: retailDeviceCount, NetworkCount: twoSiteNetworkCount,
			LinkCount:         retailLinkCount,
			DeviceNamesSHA256: "e5944f8f0f8795d3f054f6ce082c3c8c2c1dde8eab5839992487cebd554215a8",
			NetworksSHA256:    "88cdfbd6e21a58552873afe83c93dac96b7fae0a28166ecb319475dbcc38d25b",
			LinksSHA256:       "e1c65c18481d92c20c743d920c2f7086d9e849350e13a89197d4856523179761",
		},
	)
	// A store runs its lanes off one another rather than home-running each till.
	pack.Request.AccessLayer = AccessLayerChain

	return pack
}

func manufacturingScenarioPack() Pack {
	pack := newScenarioPack(
		"manufacturing", "Manufacturing plant",
		"Single production plant with resilient switching, 30 Wi-Fi 7 APs, wired stations, and local services.",
		MapPurposePresentation, "industrial.example",
		packSites(industrialSiteOctet, packSite{code: "PLT", location: "Production Plant"}),
		packCounts(manufacturingAccessSwitches, manufacturingAccessPointsPerAccess,
			manufacturingWorkstationsPerAccess),
		Manifest{
			DeviceCount: manufacturingDeviceCount, NetworkCount: singleSiteNetworkCount,
			LinkCount:         manufacturingLinkCount,
			DeviceNamesSHA256: "4bae5bd1de01cf8a6918eadf3e1c88754c680258ed0a6b9cffc543ab789f25f1",
			NetworksSHA256:    "cc9ba550031f67fef5933891d5ff0dbd2aada452ba380493a2b52c830f25d0f8",
			LinksSHA256:       "f7f48d392ce924ab868a8b0116eee50a6e3eeee24bd0586a3ac31cca802243bb",
		},
	)
	// A plant runs its cells off a fiber ring, not a home run per closet.
	pack.Request.AccessLayer = AccessLayerRing

	return pack
}

func serviceProviderScenarioPack() Pack {
	pack := newScenarioPack(
		"service-provider", "Service provider network",
		"Three metro points of presence, each handing its access nodes off a ring, "+
			"with service, wireless, and wired-client layers.",
		MapPurposePresentation, "provider.example",
		packSites(serviceProviderSiteOctet,
			packSite{code: "NYC", location: "New York Metro POP"},
			packSite{code: "DEN", location: "Denver Metro POP"},
			packSite{code: "SFO", location: "San Francisco Metro POP"},
		),
		packCounts(providerAccessSwitches, providerAccessPointsPerAccess, providerWorkstationsPerAccess),
		Manifest{
			DeviceCount: serviceProviderDeviceCount, NetworkCount: threeSiteNetworkCount,
			LinkCount:         serviceProviderLinkCount,
			DeviceNamesSHA256: "00e2aa73f52847020ae7b8b2afe86329e93087efae74adecfd3835e8685e0ea3",
			NetworksSHA256:    "79fcb26f2a9f506a24540a583ae570b5c25e1dcf8a63ccfa21946b4912fc7720",
			LinksSHA256:       "e4e6e35ca4eb6c8ebf70f02f8617b1932b02d74bc6728d7646f3f00e0805daf1",
		},
	)
	// A metro POP hands its access nodes off a ring.
	pack.Request.AccessLayer = AccessLayerRing

	return pack
}
