package scenario

func verticalScenarioPacks() []Pack {
	return []Pack{
		newScenarioPack(
			"retail", "Retail network",
			"Two retail sites with wireless point-of-sale coverage and local business services.",
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
				DeviceNamesSHA256: "fe35fa8827a70340c3990747010279b97f770e8c6658dfbfb5579b3d4f6ceb73",
				NetworksSHA256:    "88cdfbd6e21a58552873afe83c93dac96b7fae0a28166ecb319475dbcc38d25b",
				LinksSHA256:       "da53c82f48efbd25d23b21dc11e32fc8ea7c00043c7b39dc095d57ad6d340100",
			},
		),
		manufacturingScenarioPack(),
		serviceProviderScenarioPack(),
	}
}

func manufacturingScenarioPack() Pack {
	return newScenarioPack(
		"manufacturing", "Manufacturing plant",
		"Single production plant with resilient switching, 30 Wi-Fi 7 APs, wired stations, and local services.",
		MapPurposePresentation, "industrial.example",
		packSites(industrialSiteOctet, packSite{code: "PLT", location: "Production Plant"}),
		packCounts(manufacturingAccessSwitches, manufacturingAccessPointsPerAccess,
			manufacturingWorkstationsPerAccess),
		Manifest{
			DeviceCount: compactVerticalDeviceCount, NetworkCount: singleSiteNetworkCount,
			LinkCount:         compactVerticalLinkCount,
			DeviceNamesSHA256: "4bae5bd1de01cf8a6918eadf3e1c88754c680258ed0a6b9cffc543ab789f25f1",
			NetworksSHA256:    "cc9ba550031f67fef5933891d5ff0dbd2aada452ba380493a2b52c830f25d0f8",
			LinksSHA256:       "7a9980db15625aca3326a658c391d4a26edb066c45ea44bbb21f29e7feee7d90",
		},
	)
}

func serviceProviderScenarioPack() Pack {
	return newScenarioPack(
		"service-provider", "Service provider network",
		"Three metro points of presence with distribution, access, service, and wired-client layers.",
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
			DeviceNamesSHA256: "fc492e5fe3db7aa366a1c2662fdc4512849350c5046c170f108063fee437d1ab",
			NetworksSHA256:    "79fcb26f2a9f506a24540a583ae570b5c25e1dcf8a63ccfa21946b4912fc7720",
			LinksSHA256:       "b77376f0f247fa026ea07ac903999fd52027545d20296984c256a888f8b1bed7",
		},
	)
}
