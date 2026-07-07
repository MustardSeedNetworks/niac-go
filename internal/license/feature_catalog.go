// SPDX-License-Identifier: BUSL-1.1

package license

// FeatureMeta describes a single licensable feature flag with copy the UI
// can render directly instead of a raw feature ID. Label and Description
// are English-only product copy — the backend registry is the source of
// truth and is NOT translated; the frontend only localizes the UI chrome
// around this data (column headers, etc.), never the feature names
// themselves.
type FeatureMeta struct {
	// ID is the wire feature name (matches keygen's productCatalog and
	// the string proFeatures() returns).
	ID string `json:"id"`
	// Label is a short, human-readable name (e.g. "BGP Routing").
	Label string `json:"label"`
	// Description is a one-sentence explanation of what the feature
	// unlocks.
	Description string `json:"description"`
}

// featureMeta maps every feature ID proFeatures() can return to its
// human-readable label + description. Every ID in proFeatures() MUST have
// an entry here — TestFeatureCatalogCoversProFeatures enforces this so a
// newly added Pro feature can't ship without display copy.
var featureMeta = map[string]FeatureMeta{
	"unlimited_devices": {
		Label:       "Unlimited Devices",
		Description: "Simulate more than the Free tier's device cap in a single config.",
	},
	"bgp": {
		Label:       "BGP Routing",
		Description: "Simulate BGP peering and route advertisement between devices.",
	},
	"ospf": {
		Label:       "OSPF Routing",
		Description: "Simulate OSPF adjacency formation and link-state routing.",
	},
	"netbios": {
		Label:       "NetBIOS",
		Description: "Respond to NetBIOS name service queries for legacy Windows discovery.",
	},
	"ftp": {
		Label:       "FTP Server",
		Description: "Simulate an FTP server on a device for file-transfer test scenarios.",
	},
	"stp": {
		Label:       "Spanning Tree (STP)",
		Description: "Simulate Spanning Tree Protocol topology and port states.",
	},
	"ipv6_advanced": {
		Label:       "Advanced IPv6",
		Description: "Simulate advanced IPv6 behaviors beyond basic SLAAC/address assignment.",
	},
	"error_injection": {
		Label:       "Error Injection",
		Description: "Inject configurable protocol errors and malformed responses for negative testing.",
	},
	"traffic_shaping": {
		Label:       "Traffic Shaping",
		Description: "Apply bandwidth limits, latency, and jitter to simulated device traffic.",
	},
	"config_templates": {
		Label:       "Config Templates",
		Description: "Save and reuse device configurations as reusable templates.",
	},
	"multi_ip": {
		Label:       "Multi-IP Devices",
		Description: "Assign multiple IP addresses to a single simulated device.",
	},
	"pcap_ingest": {
		Label:       "PCAP Ingest",
		Description: "Import packet captures to seed or replay simulated device behavior.",
	},
	"rest_api": {
		Label:       "REST API",
		Description: "Drive NIAC programmatically via the full REST API surface.",
	},
}

// FeatureCatalog returns metadata for every feature Pro can grant, in the
// stable order proFeatures() returns them. Callers (e.g. the /api/v1/license
// handler) pair this with the caller's granted feature set to build a
// display-ready list.
func FeatureCatalog() []FeatureMeta {
	ids := proFeatures()
	catalog := make([]FeatureMeta, 0, len(ids))
	for _, id := range ids {
		meta := featureMeta[id]
		meta.ID = id
		catalog = append(catalog, meta)
	}
	return catalog
}
