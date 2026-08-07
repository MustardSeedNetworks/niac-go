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
				DeviceNamesSHA256: "e5944f8f0f8795d3f054f6ce082c3c8c2c1dde8eab5839992487cebd554215a8",
				NetworksSHA256:    "88cdfbd6e21a58552873afe83c93dac96b7fae0a28166ecb319475dbcc38d25b",
				LinksSHA256:       "c57237bbe4a9c00de9506bd06c2da4d0a32acf93e930ebf287f4f83da83d18b3",
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
			DeviceNamesSHA256: "279b172d5ce92a4652ff5015da8325297a8d7217211b19ac020cec462d1a711f",
			NetworksSHA256:    "79fcb26f2a9f506a24540a583ae570b5c25e1dcf8a63ccfa21946b4912fc7720",
			LinksSHA256:       "e64dc64a8796d5327205367f70b8934fd02c2cefd3a9d79b9a667b74f7031c9d",
		},
	)
}
