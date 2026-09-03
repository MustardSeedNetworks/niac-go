package converter

import (
	"errors"

	"gopkg.in/yaml.v3"
)

// VLANTag is a segment VLAN tag. It accepts both YAML strings ("untagged",
// "200") and bare integers (200), so a config author can write `tag: 200`
// unquoted or `tag: untagged`. The raw scalar text is preserved for the loader
// to interpret.
type VLANTag string

// UnmarshalYAML reads the raw scalar so an int or a string tag both work.
func (t *VLANTag) UnmarshalYAML(node *yaml.Node) error {
	*t = VLANTag(node.Value)

	return nil
}

// Sentinel errors for converter.
var (
	ErrInvalidLoopTimeFormat   = errors.New("invalid LoopTime format")
	ErrInvalidScaleTimeFormat  = errors.New("invalid ScaleTime format")
	ErrInvalidVlanFormat       = errors.New("invalid Vlan format")
	ErrDeviceMissingMAC        = errors.New("device missing MAC address")
	ErrDeviceMACSourceConflict = errors.New(
		"device must use either mac or vendor identity, not both",
	)
	ErrAddMibMissingOID           = errors.New("AddMib missing OID")
	ErrAddMibMissingType          = errors.New("AddMib missing type")
	ErrCapturePlaybackMissingFile = errors.New("CapturePlayback missing file name")
)

// Parser constants for field counts and lengths.
const (
	addMibQuotedArgs       = 3  // number of quoted args in AddMib directive
	minRegexMatchParts     = 2  // minimum parts from regex match
	ttlFieldCount          = 3  // TTL has ttl, ip, mask
	routerFieldCount       = 2  // Router has address, preference
	addMibFieldCount       = 3  // AddMib has OID, type, value
	communityIncludeFields = 2  // CommunityInclude has community, walkfile
	dnsPartsWithTTL        = 3  // DNS record with TTL
	dnsPartsWithRCode      = 4  // DNS record with RCode
	macAddressRawLen       = 12 // MAC address hex chars (XXXXXXXXXXXX)
)

// Config represents the YAML configuration structure.
type Config struct {
	// IncludePath is a directory the loader searches for files referenced by
	// relative path elsewhere in this config (walk files, capture files).
	// Relative to the config file's own directory when itself relative.
	IncludePath string `yaml:"include_path,omitempty"`

	// CapturePlaybacks replays a recorded PCAP alongside the simulated
	// devices. At most one entry: the runtime plays exactly one capture
	// (config.Config.CapturePlayback, one engine in the replay controller).
	// The schema kept the shape of a list, so extra entries were dropped on
	// load and deleted from disk on the next save. Refuse them instead.
	CapturePlaybacks []CapturePlayback `yaml:"capture_playbacks,omitempty" validate:"omitempty,max=1,dive"`

	// BehaviorTimelines are saved, repeatable sequences of runtime phases
	// that drive traffic and faults on a schedule after the session starts.
	// Up to 64 timelines; they run concurrently.
	BehaviorTimelines []BehaviorTimeline `yaml:"behavior_timelines,omitempty" validate:"omitempty,max=64,dive"`

	// DiscoveryProtocols sets the fleet-wide default for LLDP, CDP, EDP and
	// FDP advertisement. A device's own lldp/cdp/edp/fdp block overrides it.
	DiscoveryProtocols *DiscoveryProtocols `yaml:"discovery_protocols,omitempty"`

	// Devices is the simulated device set served as a single untagged
	// segment. Mutually exclusive with `segments`: when segments is present,
	// this must be empty.
	Devices []Device `yaml:"devices" validate:"omitempty,dive"`

	// Segments binds independent device sets to VLAN tags for multi-VLAN
	// playback (ADR 0008). When present, each segment is served as its own
	// isolated network (own IP space) on its tag, and top-level `devices` must
	// be empty. When absent, `devices` is served as a single untagged segment —
	// today's flat behavior.
	Segments []Segment `yaml:"segments,omitempty" validate:"omitempty,dive"`

	// Networks declares the routed IPv4 networks devices attach to. An
	// interface naming one in its `network` field must carry an `address`
	// written as a prefix (10.10.0.5/24), not a bare address.
	Networks []Network `yaml:"networks,omitempty" validate:"omitempty,dive"`

	// Attachments bind this config's virtual networks to the deployment's
	// host-side interfaces. Preflight rejects an attachment naming a network
	// this config does not declare, or a host interface the daemon's
	// attachment policy does not permit.
	Attachments []LogicalAttachment `yaml:"attachments,omitempty" validate:"omitempty,dive"`
}

// BehaviorTimeline is one saved, repeatable sequence of runtime phases.
type BehaviorTimeline struct {
	// Name identifies the timeline in the runtime view and the API. Unique
	// within the config.
	Name string `yaml:"name" validate:"required,max=100"`

	// StartOffsetMS delays the whole timeline this many milliseconds after
	// the session starts. 0 starts it immediately.
	StartOffsetMS int `yaml:"start_offset_ms,omitempty" validate:"gte=0,lte=86400000"`

	// RepeatCount is how many times the phase sequence runs, 1..1000. There
	// is no infinite repeat; pick a count that covers the intended run.
	RepeatCount int `yaml:"repeat_count" validate:"gte=1,lte=1000"`

	// Phases are the ordered steps of the timeline. Each phase's
	// start_offset_ms is relative to the timeline, not to the previous phase.
	Phases []BehaviorPhase `yaml:"phases" validate:"required,max=256,dive"`
}

// BehaviorPhase applies traffic and faults at a deterministic offset.
type BehaviorPhase struct {
	// Name identifies the phase in the runtime view.
	Name string `yaml:"name" validate:"required,max=100"`

	// StartOffsetMS is when this phase begins, measured from the start of the
	// timeline (not from the previous phase).
	StartOffsetMS int `yaml:"start_offset_ms,omitempty" validate:"gte=0,lte=86400000"`

	// DurationMS is how long the phase's traffic and faults stay applied.
	DurationMS int `yaml:"duration_ms" validate:"gte=1,lte=86400000"`

	// Reset clears the traffic and faults set by earlier phases before this
	// one applies its own, instead of layering on top of them.
	Reset bool `yaml:"reset,omitempty"`

	// Traffic sets observable interface utilization for the phase's duration.
	Traffic []BehaviorTraffic `yaml:"traffic,omitempty" validate:"omitempty,max=1024,dive"`

	// Faults injects SNMP-visible interface faults for the phase's duration.
	Faults []BehaviorFault `yaml:"faults,omitempty" validate:"omitempty,max=1024,dive"`
}

// BehaviorTraffic sets observable utilization on one simulated interface.
type BehaviorTraffic struct {
	// Device is the `name` of the device carrying the interface.
	Device string `yaml:"device" validate:"required"`

	// Interface is the interface `name` on that device, as declared in its
	// `interfaces` list.
	Interface string `yaml:"interface" validate:"required"`

	// Utilization is the percentage, 1..100, reported through the interface's
	// IF-MIB counters while the phase is active.
	Utilization int `yaml:"utilization" validate:"gte=1,lte=100"`
}

// BehaviorFault sets one supported SNMP interface fault rate.
type BehaviorFault struct {
	// Device is the `name` of the device carrying the interface.
	Device string `yaml:"device" validate:"required"`

	// Interface is the interface `name` on that device, as declared in its
	// `interfaces` list.
	Interface string `yaml:"interface" validate:"required"`

	// Type is the fault to inject. Interface-scoped only: these raise SNMP
	// counters, they do not take a service (DHCP, DNS) out.
	Type string `yaml:"type" validate:"required,oneof=fcs_errors packet_discards interface_errors high_utilization"`

	// Value is the rate or percentage, 1..100, applied while the phase runs.
	Value int `yaml:"value" validate:"gte=1,lte=100"`
}

// Network declares one internal routed IPv4 network.
type Network struct {
	// Name is how interfaces and attachments refer to this network. Unique
	// within the config.
	Name string `yaml:"name" validate:"required"`

	// Subnet is the network in CIDR form, for example 10.10.0.0/24. Every
	// interface address on this network must fall inside it, and a DHCP pool
	// served onto it must sit within the same subnet.
	Subnet string `yaml:"subnet" validate:"required,cidr"`

	// VirtualVLAN tags this network's frames with a VLAN id, 1..4094, when
	// the network is served on a trunk. Omit for untagged.
	VirtualVLAN int `yaml:"virtual_vlan,omitempty" validate:"omitempty,gte=1,lte=4094"`
}

// LogicalAttachment names the virtual network exposed by a deployment binding.
type LogicalAttachment struct {
	// Name labels this attachment. It is what a session's binding selects the
	// attachment by at start time, not a network name — a binding naming an
	// attachment this config does not declare is reported as
	// `unknown_attachment`.
	Name string `yaml:"name" validate:"required"`

	// Connect is the `networks[].name` this attachment exposes to the host.
	// Whether the host interface may actually carry it is decided at start
	// against the daemon's attachment policy, which reports
	// `attachment_policy_denied` or `host_interface_unavailable`;
	// `niac validate` has no host binding and cannot check those.
	Connect string `yaml:"connect" validate:"required"`
}

// Segment binds a device set to a VLAN tag for multi-VLAN playback. Exactly one
// of Devices (inline) or Config (a path to a config file) is set.
type Segment struct {
	// Tag is "untagged" (the native VLAN) or a VLAN id in 1..4094.
	Tag VLANTag `yaml:"tag" validate:"required"`

	// Devices is an inline device set for this segment.
	Devices []Device `yaml:"devices,omitempty" validate:"omitempty,dive"`
	// Config is a path to a config file whose devices form this segment
	// (resolved by the loader). Mutually exclusive with Devices.
	Config string `yaml:"config,omitempty"`
}

// DiscoveryProtocols configures discovery protocol behavior.
type DiscoveryProtocols struct {
	// LLDP sets the fleet-wide default for IEEE 802.1AB advertisement.
	LLDP *ProtocolConfig `yaml:"lldp,omitempty"`

	// CDP sets the fleet-wide default for Cisco Discovery Protocol.
	CDP *ProtocolConfig `yaml:"cdp,omitempty"`

	// EDP sets the fleet-wide default for Extreme Discovery Protocol.
	EDP *ProtocolConfig `yaml:"edp,omitempty"`

	// FDP sets the fleet-wide default for Foundry Discovery Protocol.
	FDP *ProtocolConfig `yaml:"fdp,omitempty"`
}

// ProtocolConfig configures a discovery protocol.
type ProtocolConfig struct {
	// Enabled turns the protocol on for every device that does not override
	// it with its own block.
	Enabled bool `yaml:"enabled"`

	// Interval is the advertisement interval in seconds. Omit for the
	// protocol's own default (30 s for LLDP, 60 s for CDP).
	Interval int `yaml:"interval,omitempty"`
}

// CapturePlayback represents PCAP playback configuration.
type CapturePlayback struct {
	// FileName is the pcap or pcapng file to replay, resolved against
	// `include_path` when relative.
	FileName string `yaml:"file_name" validate:"required"`

	// LoopTime is the seconds to wait before replaying the file again.
	// 0 replays it once.
	LoopTime int `yaml:"loop_time,omitempty" validate:"omitempty,gte=0"`

	// ScaleTime multiplies the capture's inter-packet gaps: 2.0 replays at
	// half speed, 0.5 at double. Omit or 0 replays at the recorded rate.
	ScaleTime float64 `yaml:"scale_time,omitempty" validate:"omitempty,gte=0"`
}

// Device represents a network device.
type Device struct {
	// Name is the device's identity across the config: behaviour phases,
	// trunk_ports.remote_device and the topology graph all refer to it.
	// Unique within the config, and read-only once the device exists — the
	// daemon takes a device's name from the URL, so renaming through the
	// editor is not supported.
	Name string `yaml:"name,omitempty"`

	// Type selects the device persona: which MIBs it serves, which icon the
	// topology draws and how a scanner classifies it. Note the authored
	// spelling: the vocabulary hyphenates, so `voip-phone` and
	// `layer3-switch` are accepted and their underscored forms are not. An
	// access point is `ap` or `access-point`, never `access_point`.
	Type string `yaml:"type,omitempty" validate:"omitempty,oneof=router switch layer3-switch ap access-point firewall server host workstation iot printer voip-phone"`

	// MAC is the device's explicit base MAC address (aa:bb:cc:dd:ee:ff).
	// Mutually exclusive with `vendor`: set one or the other, never both.
	MAC string `yaml:"mac,omitempty" validate:"omitempty,mac"`

	// Vendor derives the MAC from a vendor OUI instead of stating one.
	// Mutually exclusive with `mac`. Set `mac_suffix` alongside it: without a
	// suffix every device of the same vendor collides on the same address,
	// ending :00:00:00.
	Vendor string `yaml:"vendor,omitempty"`

	// MACSuffix is the per-device low 24 bits appended to the vendor OUI,
	// 0..16777215. Required in practice whenever `vendor` is used more than
	// once.
	MACSuffix uint32 `yaml:"mac_suffix,omitempty" validate:"lte=16777215"`

	// IPs are the device's addresses as bare IPs (10.10.0.5), used when the
	// device is not modelled through `interfaces` and `networks`. Interface
	// addresses use a prefix instead.
	IPs []string `yaml:"ips,omitempty" validate:"omitempty,dive,ip"`

	// VLAN is the access VLAN, 1..4094, for a device served on a trunk
	// without its own interface list.
	VLAN int `yaml:"vlan,omitempty" validate:"omitempty,gte=1,lte=4094"`

	// MapToIP answers for this additional address as well, so one simulated
	// device can respond on a second IP.
	MapToIP string `yaml:"map_to_ip,omitempty" validate:"omitempty,ip"`

	// Babble makes the device emit unsolicited background chatter, so a
	// passive scanner sees it without probing.
	Babble bool `yaml:"babble,omitempty"`

	// TTL configures ICMP TTL expiry so the device appears as a traceroute
	// hop. It is an object (ttl / ip / mask), not a bare integer — a plain
	// `ttl: 64` is rejected. The per-packet IP TTL lives in
	// `os_fingerprint.ttl`.
	TTL *TTLConfig `yaml:"ttl,omitempty"`

	// SnmpAgent serves SNMP for this device, from a walk file and/or
	// individual OID overrides.
	SnmpAgent *SnmpAgent `yaml:"snmp_agent,omitempty"`

	// Dhcp makes the device a DHCP server. Its pool must sit inside a routed
	// network this config declares, or preflight rejects the config.
	Dhcp *DhcpServer `yaml:"dhcp,omitempty"`

	// DNS makes the device a DNS server. Its records key the address as `ip`,
	// not `address`.
	DNS *DNSServer `yaml:"dns,omitempty"`

	// Lldp overrides the fleet-wide `discovery_protocols.lldp` for this
	// device and carries its advertised strings.
	Lldp *LldpConfig `yaml:"lldp,omitempty"`

	// Cdp overrides the fleet-wide `discovery_protocols.cdp` for this device.
	Cdp *CdpConfig `yaml:"cdp,omitempty"`

	// Edp overrides the fleet-wide `discovery_protocols.edp` for this device.
	Edp *EdpConfig `yaml:"edp,omitempty"`

	// Fdp overrides the fleet-wide `discovery_protocols.fdp` for this device.
	Fdp *FdpConfig `yaml:"fdp,omitempty"`

	// Stp makes the device participate in spanning tree, so a scanner sees
	// bridge priorities and a root election.
	Stp *StpConfig `yaml:"stp,omitempty"`

	// HTTP serves a web listener with author-defined endpoints, which is what
	// makes a device identifiable as a server or an appliance.
	HTTP *HTTPConfig `yaml:"http,omitempty"`

	// Ftp serves an FTP listener with a banner and optional accounts.
	Ftp *FtpConfig `yaml:"ftp,omitempty"`

	// Netbios answers NetBIOS name queries, which is how Windows-facing
	// scanners name a host. Names are capped at 15 characters.
	Netbios *NetbiosConfig `yaml:"netbios,omitempty"`

	// Mdns publishes the device and its services over multicast DNS, the way
	// Bonjour and Avahi announce a printer or a camera.
	Mdns *MdnsConfig `yaml:"mdns,omitempty"`

	// Snmpv3 adds USM users for authenticated, optionally encrypted SNMP.
	// Independent of `snmp_agent`, which serves v1/v2c.
	Snmpv3 *Snmpv3Config `yaml:"snmpv3,omitempty"`

	// Icmp configures ping and IPv4 router advertisement behaviour.
	Icmp *IcmpConfig `yaml:"icmp,omitempty"`

	// Icmpv6 configures IPv6 ping, neighbour discovery and router
	// advertisement behaviour.
	Icmpv6 *Icmpv6Config `yaml:"icmpv6,omitempty"`

	// Dhcpv6 makes the device a DHCPv6 server. Omit the block entirely when
	// the device should not serve DHCPv6: an empty `dhcpv6: {}` is a
	// configured server that picks up the default lifetimes.
	Dhcpv6 *Dhcpv6Config `yaml:"dhcpv6,omitempty"`

	// OSFingerprint shapes the device's TCP/IP stack and service banners so a
	// fingerprinting scanner identifies the intended operating system.
	OSFingerprint *OSFingerprintConfig `yaml:"os_fingerprint,omitempty"`

	// SSH serves an authenticated vendor-like CLI. The password is never
	// written in the config: `password_env` names an environment variable
	// that must be set in the daemon's environment, or the device cannot
	// start.
	SSH *SSHConfig `yaml:"ssh,omitempty"`

	// Syslog sends this device's state-change messages to RFC 5424
	// collectors.
	Syslog *SyslogConfig `yaml:"syslog,omitempty"`

	// IPerf3 answers iperf3 throughput tests with the authored rates.
	IPerf3 *IPerf3Config `yaml:"iperf3,omitempty"`

	// Reflector makes the device a NetAlly-style UDP reflector endpoint for
	// TrueSpeed and performance tests. Presence enables it; there is no
	// separate enable flag.
	Reflector *ReflectorConfig `yaml:"reflector,omitempty"`

	// Interfaces declares the device's ports: addresses, speeds, status and
	// the network each one attaches to. This is what IF-MIB reports and what
	// behaviour phases target by name.
	Interfaces []Interface `yaml:"interfaces,omitempty" validate:"omitempty,dive"`

	// Routes are static IPv4 routes this device advertises and forwards on.
	Routes []Route `yaml:"routes,omitempty" validate:"omitempty,dive"`

	// TrunkPorts declare the VLAN-tagged links to neighbouring devices. This
	// is how topology edges are authored — there is no separate `links`
	// section; an edge exists because a trunk port names a `remote_device`.
	TrunkPorts []TrunkPort `yaml:"trunk_ports,omitempty"`

	// PortChannels bundle member interfaces into a LAG. They draw no edge on
	// their own: a trunk_port whose `interface` is "port-channel<id>" is what
	// surfaces the link.
	PortChannels []PortChannel `yaml:"port_channels,omitempty"`

	// Properties is a free-form vendor-metadata block used by the
	// vendor template pack (cmd/niac/templates/vendor-templates) to
	// carry vendor / model / OS / port-config strings into the
	// device list. Keys are author-defined; consumers treat unknown
	// keys as informational. The loader's own derived keys (vlan,
	// custom_mibs_count) are not authored here.
	Properties map[string]string `yaml:"properties,omitempty"`
}

// Interface represents configured metadata for a device interface.
type Interface struct {
	// Name is the port name, for example GigabitEthernet0/1 or eth0.
	// Behaviour phases and trunk ports target the interface by this name.
	Name string `yaml:"name"`

	// Type is the IF-MIB interface type reported for this port.
	Type string `yaml:"type,omitempty" validate:"omitempty,oneof=ethernet ieee80211 l2vlan l3ipvlan loopback tunnel other"`

	// Network is the `networks[].name` this port attaches to. When set, the
	// port's `address` must be written as a prefix and must fall inside that
	// network's subnet.
	Network string `yaml:"network,omitempty"`

	// Address is the port's IPv4 address, written as a prefix
	// (10.10.0.5/24) — a bare address is rejected. When `network` is set the
	// address must fall inside that network's subnet and carry the same
	// prefix length; a /32 host address on a /24 network is refused.
	Address string `yaml:"address,omitempty" validate:"omitempty,cidr"`

	// MTU is the port's MTU in bytes, 576..1000000.
	MTU int `yaml:"mtu,omitempty" validate:"omitempty,gte=576,lte=1000000"`

	// Speed is the port's nominal speed in Mbps, reported through IF-MIB.
	Speed int `yaml:"speed,omitempty"`

	// Duplex is the port's duplex mode.
	Duplex string `yaml:"duplex,omitempty" validate:"omitempty,oneof=full half"`

	// AdminStatus is the configured state of the port (ifAdminStatus).
	AdminStatus string `yaml:"admin_status,omitempty" validate:"omitempty,oneof=up down testing"`

	// OperStatus is the observed state of the port (ifOperStatus). A link
	// fault drives this down at runtime.
	OperStatus string `yaml:"oper_status,omitempty" validate:"omitempty,oneof=up down testing"`

	// Description is the port's ifAlias text, as a switch's port description.
	Description string `yaml:"description,omitempty"`

	// InUtilization is the steady-state inbound utilization percentage,
	// 0..100, before any behaviour phase changes it.
	InUtilization float64 `yaml:"in_utilization,omitempty" validate:"omitempty,gte=0,lte=100"`

	// OutUtilization is the steady-state outbound utilization percentage,
	// 0..100, before any behaviour phase changes it.
	OutUtilization float64 `yaml:"out_utilization,omitempty" validate:"omitempty,gte=0,lte=100"`

	// VLANs are the VLAN ids carried on this port when it is a switch port.
	VLANs []int `yaml:"vlans,omitempty"`
}

// Route declares an IPv4 static route through a named device interface.
type Route struct {
	// Destination is the target network in CIDR form, for example
	// 10.20.0.0/24. Use 0.0.0.0/0 for a default route.
	Destination string `yaml:"destination" validate:"required,cidr"`

	// Via is the `name` of the local interface the route leaves by.
	Via string `yaml:"via" validate:"required"`

	// NextHop is the gateway's bare IPv4 address — an address, not a prefix.
	NextHop string `yaml:"next_hop" validate:"required,ip4_addr"`
}

// TrunkPort declares a VLAN-tagged trunk link to a neighbouring device.
// The remote_device field is what BuildTopology uses to draw an edge
// between two devices on the topology graph.
type TrunkPort struct {
	// Interface is the local port carrying the trunk, matching an
	// `interfaces[].name`, or "port-channel<id>" for a LAG.
	Interface string `yaml:"interface"`

	// VLANs are the tagged VLAN ids the trunk carries. A port with no VLANs
	// and no native VLAN is an access port, not a trunk.
	VLANs []int `yaml:"vlans,omitempty"`

	// NativeVLAN is the untagged VLAN on this trunk.
	NativeVLAN int `yaml:"native_vlan,omitempty"`

	// RemoteDevice is the `name` of the device at the far end. This is what
	// draws the topology edge; without it the port is a stub.
	RemoteDevice string `yaml:"remote_device,omitempty"`

	// RemoteInterface is the far-end port name, used to label the edge.
	RemoteInterface string `yaml:"remote_interface,omitempty"`

	// FDBOnly records the neighbour in the forwarding database without
	// drawing a topology edge, for a device seen but not linked.
	FDBOnly bool `yaml:"fdb_only,omitempty"`
}

// PortChannel declares a Link Aggregation (LACP) bundle. Doesn't
// produce topology edges on its own — trunk_ports referencing
// "port-channel<id>" as their interface are what surface edges.
type PortChannel struct {
	// ID is the channel number; a trunk port refers to the bundle as
	// "port-channel<id>".
	ID int `yaml:"id"`

	// Members are the `interfaces[].name` values bundled into the channel.
	Members []string `yaml:"members,omitempty"`

	// Mode is the LACP mode, for example active, passive or on.
	Mode string `yaml:"mode,omitempty"`
}

// Parser handles parsing Java DSL format.
type Parser struct {
	lines   []string
	pos     int
	verbose bool
}

// maxInputFileSize is the maximum allowed input file size (10 MB).
const maxInputFileSize = 10 * 1024 * 1024
