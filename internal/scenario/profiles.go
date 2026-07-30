package scenario

import "github.com/MustardSeedNetworks/niac-go/internal/protocols/snmp/synth"

// DeviceProfile is the reusable vendor, model, role, and discovery identity.
type DeviceProfile struct {
	Role              string             `json:"role"`
	DeviceType        string             `json:"deviceType"`
	Vendor            string             `json:"vendor"`
	Model             string             `json:"model"`
	Platform          string             `json:"platform"`
	Software          string             `json:"software"`
	SysObjectID       string             `json:"sysObjectId"`
	WalkName          string             `json:"walkName,omitempty"`
	InterfaceCount    int                `json:"interfaceCount,omitempty"`
	SupportedSNMPData []string           `json:"supportedSnmpData,omitempty"`
	Interfaces        []ProfileInterface `json:"interfaces,omitempty"`
	Source            string             `json:"source,omitempty"`
}

// ProfileInterface is the reusable interface inventory inferred from a walk.
type ProfileInterface struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	MTU         int    `json:"mtu,omitempty"`
	Speed       int64  `json:"speed,omitempty"`
	AdminStatus string `json:"adminStatus,omitempty"`
	OperStatus  string `json:"operStatus,omitempty"`
}

// EnterpriseReferenceRequest returns the accepted four-site Link-Live lab shape.
func EnterpriseReferenceRequest() Request {
	return Request{
		Sites: []Site{
			{Code: "COS", Octet: enterpriseCOSOctet, Location: "Colorado Springs, CO"},
			{Code: "EVT", Octet: enterpriseEVTOctet, Location: "Everett, WA"},
			{Code: "EHV", Octet: enterpriseEHVOctet, Location: "Eindhoven, Netherlands"},
			{Code: "LON", Octet: enterpriseLONOctet, Location: "London, UK"},
		},
		Counts: Counts{
			SiteWANRouters: maxRedundantPeers, Firewalls: maxRedundantPeers, CoreSwitches: maxRedundantPeers,
			DistributionSwitches: enterpriseDistributionSwitches, AccessSwitches: enterpriseAccessSwitches,
			ServerSwitches: maxRedundantPeers, AccessPointsPerAccess: maxRedundantPeers,
			WorkstationsPerAccess: enterpriseWorkstationsPerAccess, WirelessControllers: maxRedundantPeers,
		},
		Domain: defaultDomain, SNMPCommunity: defaultCommunity, AttachmentName: defaultAttachmentName,
	}
}

// Profiles returns the generator's role catalog using synthesized-walk identities.
func Profiles() []DeviceProfile {
	profiles := networkProfiles()
	profiles = append(profiles, campusProfiles()...)
	return append(profiles, endpointProfiles()...)
}

func newProfile(role, deviceType, vendor, model, platform, software string,
	synthVendor synth.Vendor, synthType synth.DeviceType,
) DeviceProfile {
	walkProfile, err := synth.Pick(synthVendor, synthType)
	if err != nil {
		panic(err)
	}
	return DeviceProfile{
		Role: role, DeviceType: deviceType, Vendor: vendor, Model: model,
		Platform: platform, Software: software, SysObjectID: walkProfile.SysObjectID,
	}
}

func profileByRole(role string) DeviceProfile {
	for _, profile := range Profiles() {
		if profile.Role == role {
			return profile
		}
	}
	panic("unknown scenario profile: " + role)
}
