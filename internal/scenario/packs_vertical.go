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
		"A back office, and a store whose lanes chain off each other, "+
			"with wireless point-of-sale coverage and local business services.",
		MapPurposePresentation,
		"retail.example", packSites(retailSiteOctet,
			packSite{code: "HQ", location: "Retail Headquarters"},
			packSite{code: "STR", location: "Regional Flagship Store"},
		), packCounts(
			retailAccessSwitches,
			retailAccessPointsPerAccess,
			retailWorkstationsPerAccess,
		),
		storeDHCPNoOffer()...,
	)
	// A store runs its lanes off one another rather than home-running each till.
	pack.Request.AccessLayer = AccessLayerChain

	return pack
}

func manufacturingScenarioPack() Pack {
	pack := newScenarioPack(
		"manufacturing", "Manufacturing plant",
		"Production plant with resilient switching, Wi-Fi 7 coverage, wired stations, and local services.",
		MapPurposePresentation, "industrial.example",
		packSites(industrialSiteOctet, packSite{code: "PLT", location: "Production Plant"}),
		packCounts(manufacturingAccessSwitches, manufacturingAccessPointsPerAccess,
			manufacturingWorkstationsPerAccess),
		ringSegmentFCSErrors()...,
	)
	// A plant runs its cells off a fiber ring, not a home run per closet.
	pack.Request.AccessLayer = AccessLayerRing

	return pack
}

func serviceProviderScenarioPack() Pack {
	pack := newScenarioPack(
		"service-provider", "Service provider network",
		"Metro points of presence, each handing its access nodes off a ring, "+
			"with service, wireless, and wired-client layers.",
		MapPurposePresentation, "provider.example",
		packSites(serviceProviderSiteOctet,
			packSite{code: "NYC", location: "New York Metro POP"},
			packSite{code: "DEN", location: "Denver Metro POP"},
			packSite{code: "SFO", location: "San Francisco Metro POP"},
		),
		packCounts(providerAccessSwitches, providerAccessPointsPerAccess, providerWorkstationsPerAccess),
		popUplinkLatency()...,
	)
	// A metro POP hands its access nodes off a ring.
	pack.Request.AccessLayer = AccessLayerRing

	return pack
}
