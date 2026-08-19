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

// CongestedLink is one authored trouble spot: a link the guided demo is meant
// to find. It replaces the generated utilization band on one interface, which is
// how a pack tells a story rather than rendering uniformly healthy.
type CongestedLink struct {
	Device         string  `json:"device"`
	Interface      string  `json:"interface"`
	InUtilization  float64 `json:"inUtilization"`
	OutUtilization float64 `json:"outUtilization"`
}

// Request is the complete deterministic fleet-generation contract.
type Request struct {
	Sites           []Site          `json:"sites"`
	Counts          Counts          `json:"counts"`
	Domain          string          `json:"domain"`
	SNMPCommunity   string          `json:"snmpCommunity"`
	AttachmentName  string          `json:"attachmentName"`
	EndpointProfile string          `json:"endpointProfile,omitempty"`
	AccessLayer     AccessLayer     `json:"accessLayer,omitempty"`
	Congestion      []CongestedLink `json:"congestion,omitempty"`
}

// ManifestSchemaVersion is the version of the manifest document. Version 3
// carried counts and three hashes; version 4 adds the reproducible identity,
// interface truth, expected observations, and timing tolerances a consumer
// needs to assert against a running scenario without operating it by hand.
const ManifestSchemaVersion = 4

// SEED's collector names. These are the strings SEED's snmp.Collector.Name()
// returns and the keys its polling_targets.collector_chain uses — four of them
// differ from the package directory they live in. Neither repository imports
// the other's internal packages, so this is a written-down contract: changing a
// name here without changing it in SEED silently breaks the pairing.
const (
	CollectorSysInfo = "sys_info"
	CollectorIfTable = "if_table"
	CollectorLLDP    = "lldp"
	CollectorCDP     = "cdp"
	CollectorFDP     = "fdp"
	CollectorRouting = "routing"
	CollectorFDB     = "fdb"
)

// Parity is the frozen authored-truth contract a pack pins: the counts and
// digests that must not drift. It is deliberately comparable, so a pack can
// assert equality against a freshly generated scenario in one expression.
type Parity struct {
	DeviceCount       int    `json:"deviceCount"`
	NetworkCount      int    `json:"networkCount"`
	LinkCount         int    `json:"linkCount"`
	DeviceNamesSHA256 string `json:"deviceNamesSha256"`
	NetworksSHA256    string `json:"networksSha256"`
	LinksSHA256       string `json:"linksSha256"`
}

// Identity is the reproducible input that produced a scenario. Generation is
// deterministic and takes no random seed, so the digest of the request *is* the
// seed: a consumer holding it can regenerate byte-identical YAML.
type Identity struct {
	RequestSHA256   string `json:"requestSha256"`
	Domain          string `json:"domain"`
	AccessLayer     string `json:"accessLayer,omitempty"`
	EndpointProfile string `json:"endpointProfile,omitempty"`
}

// InterfaceTruth is what a consumer polling ifTable should find. The digest
// covers the operational facts a collector actually reads, so an edit to speed,
// duplex, or either status changes it while a cosmetic edit does not.
type InterfaceTruth struct {
	Count     int             `json:"count"`
	SHA256    string          `json:"sha256"`
	Congested []CongestedLink `json:"congested,omitempty"`
}

// Observation is what one SEED collector should find against this scenario.
//
// An absent collector key means the scenario authors nothing that collector
// reads. That is a different claim from a count of zero: zero says "poll this
// and expect an empty table", absent says "this scenario makes no promise".
// A consumer that conflates them will assert an emptiness never promised.
type Observation struct {
	Devices int `json:"devices"`
	Rows    int `json:"rows,omitempty"`
}

// Timing bounds how long a consumer must wait before an observation is stable.
// Neighbour tables cannot be complete until every advertiser has transmitted at
// least once, so the tolerance follows the slowest advertisement interval the
// scenario actually authors rather than a chosen number.
type Timing struct {
	LLDPIntervalSeconds         int `json:"lldpIntervalSeconds,omitempty"`
	CDPIntervalSeconds          int `json:"cdpIntervalSeconds,omitempty"`
	FDPIntervalSeconds          int `json:"fdpIntervalSeconds,omitempty"`
	NeighborsStableAfterSeconds int `json:"neighborsStableAfterSeconds"`
}

// Manifest is the authored truth a consumer checks a running scenario against.
// The parity fields stay inline at the top level so existing readers of
// deviceCount and the digests are unaffected by the version 4 additions.
type Manifest struct {
	SchemaVersion int `json:"schemaVersion"`

	DeviceCount       int    `json:"deviceCount"`
	NetworkCount      int    `json:"networkCount"`
	LinkCount         int    `json:"linkCount"`
	DeviceNamesSHA256 string `json:"deviceNamesSha256"`
	NetworksSHA256    string `json:"networksSha256"`
	LinksSHA256       string `json:"linksSha256"`

	Identity     Identity               `json:"identity"`
	Interfaces   InterfaceTruth         `json:"interfaces"`
	Observations map[string]Observation `json:"expectedObservations"`
	Timing       Timing                 `json:"timing"`
}

// Parity returns the frozen subset a pack pins.
func (m Manifest) Parity() Parity {
	return Parity{
		DeviceCount: m.DeviceCount, NetworkCount: m.NetworkCount, LinkCount: m.LinkCount,
		DeviceNamesSHA256: m.DeviceNamesSHA256, NetworksSHA256: m.NetworksSHA256,
		LinksSHA256: m.LinksSHA256,
	}
}

// Result contains portable YAML plus its deterministic parity manifest.
type Result struct {
	YAML     []byte   `json:"-"`
	Manifest Manifest `json:"manifest"`
}
