// Package scenario builds deterministic, validated simulation configurations
// from typed customer authoring requests.
package scenario

const (
	defaultDomain            = "demo.lab"
	defaultCommunity         = "NetAllyDemo"
	defaultAttachmentName    = "cyberscope"
	maxSites                 = 4
	maxSiteAccessPoints      = 154
	maxSiteWorkstations      = 79
	maxRedundantPeers        = 2
	maxDistributionSwitches  = 8
	maxAccessSwitches        = 20
	maxServerSwitches        = 8
	maxAccessPointsPerAccess = 9
	maxWorkstationsPerAccess = 39
	maxWirelessControllers   = 8
	maxScenarioDomainLength  = 237
	maxSNMPCommunityLength   = 255
	maxAttachmentNameLength  = 64
	maxSiteLocationLength    = 128

	vlanManagement = 200
	vlanData       = 210
	vlanWiFiCorp   = 220
	vlanWiFiGuest  = 230
	vlanServers    = 240
	vlanVoiceIoT   = 250

	speedOneGigabit     = 1000
	speedTenGigabit     = 10000
	speedHundredGigabit = 100000
	standardMTU         = 1500

	transitPeerOffset            = 2
	secondProviderWANRouterIndex = 2
	firstSiteInterfaceIndex      = 2
	wanManagementHostOffset      = 3
	firewallManagementHostOffset = 5
	distributionHostOffset       = 10
	accessHostOffset             = 20
	serverSwitchHostOffset       = 40
	accessPointHostOffset        = 99
	workstationHostOffset        = 20
	serviceHostOffset            = 9
	controllerHostOffset         = 19
	accessUplinkPortStart        = 49
	// Fewer than three nodes is a pair of parallel links, not a ring.
	minimumRingNodes = 3
	// The ring ports sit above the access-point range and below the uplinks, so
	// a ring switch keeps the same port plan as a dual-homed one.
	ringEastPort                = "TenGigabitEthernet1/0/47"
	ringWestPort                = "TenGigabitEthernet1/0/48"
	coreServerPortOffset        = 8
	workstationPortOffset       = 9
	serverPortOffset            = 11
	primaryCoreGatewayHost      = 2
	dnsServerHost               = 10
	dhcpServerHost              = 11
	dhcpPoolStartHost           = 100
	dhcpPoolEndHost             = 199
	managedDeviceTTL            = 255
	windowsTTL                  = 128
	windowsTCPWindowSize        = 64240
	windowsMSS                  = 1460
	dnsRecordTTL                = 300
	performanceTestPort         = 5201
	performanceBandwidthMbps    = 10000
	performanceLatencyMillis    = 2
	performanceJitterMillis     = 1
	performancePacketLoss       = 0.01
	wanIdentityOffset           = 10
	transitBlockSize            = 8
	firstDistributionAccessPort = 3
	macHighByteShift            = 16
	macMiddleByteShift          = 8

	enterpriseCOSOctet              = 240
	enterpriseEVTOctet              = 241
	enterpriseEHVOctet              = 242
	enterpriseLONOctet              = 243
	enterpriseDistributionSwitches  = 4
	enterpriseAccessSwitches        = 16
	enterpriseWorkstationsPerAccess = 4
)

const (
	roleLabCode = 1 + iota
	roleWANCode
	roleFirewallCode
	roleCoreCode
	roleDistributionCode
	roleAccessCode
	roleServerSwitchCode
	roleAccessPointCode
	roleWorkstationCode
	roleServerCode
	roleControllerCode
)

// Site identifies one generated location and its second IPv4 octet.
type Site struct {
	Code     string `json:"code"`
	Octet    int    `json:"octet"`
	Location string `json:"location"`
}

// Counts controls how many devices the generator repeats at each site.
type Counts struct {
	SiteWANRouters        int `json:"siteWanRouters"`
	Firewalls             int `json:"firewalls"`
	CoreSwitches          int `json:"coreSwitches"`
	DistributionSwitches  int `json:"distributionSwitches"`
	AccessSwitches        int `json:"accessSwitches"`
	ServerSwitches        int `json:"serverSwitches"`
	AccessPointsPerAccess int `json:"accessPointsPerAccess"`
	WorkstationsPerAccess int `json:"workstationsPerAccess"`
	WirelessControllers   int `json:"wirelessControllers"`
}

// AccessLayer is how a site organizes its access tier. It is shape rather than
// a count, and it is what makes one vertical's map read differently from
// another's at a glance.
type AccessLayer string

const (
	// AccessLayerDualHomed hangs every access switch off both distribution
	// switches. It is the default, and the hospital's Link-Live-validated shape.
	AccessLayerDualHomed AccessLayer = ""
	// AccessLayerRing chains the access switches into one closed ring that meets
	// the distribution tier at two opposite points, the way a plant runs its
	// cells off a fiber ring rather than home-running every closet.
	AccessLayerRing AccessLayer = "ring"
	// AccessLayerCollapsedCore lands every access switch straight on the core
	// pair. A campus that is wide and shallow does not earn a distribution tier,
	// and generating one it does not use would be a tier nobody deployed.
	AccessLayerCollapsedCore AccessLayer = "collapsed-core"
	// AccessLayerChain daisy-chains the access switches, with only the first
	// uplinked. A store runs its lanes off one another rather than home-running
	// every till to the back office.
	AccessLayerChain AccessLayer = "chain"
)

// Request is the complete deterministic fleet-generation contract.
type Request struct {
	Sites           []Site      `json:"sites"`
	Counts          Counts      `json:"counts"`
	Domain          string      `json:"domain"`
	SNMPCommunity   string      `json:"snmpCommunity"`
	AttachmentName  string      `json:"attachmentName"`
	EndpointProfile string      `json:"endpointProfile,omitempty"`
	AccessLayer     AccessLayer `json:"accessLayer,omitempty"`
}

// Manifest summarizes authored identity and topology for parity checks.
type Manifest struct {
	DeviceCount       int    `json:"deviceCount"`
	NetworkCount      int    `json:"networkCount"`
	LinkCount         int    `json:"linkCount"`
	DeviceNamesSHA256 string `json:"deviceNamesSha256"`
	NetworksSHA256    string `json:"networksSha256"`
	LinksSHA256       string `json:"linksSha256"`
}

// Result contains portable YAML plus its deterministic parity manifest.
type Result struct {
	YAML     []byte   `json:"-"`
	Manifest Manifest `json:"manifest"`
}
