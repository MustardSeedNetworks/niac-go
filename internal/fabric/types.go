// Package fabric compiles routed scenario configuration into immutable topology.
package fabric

import (
	"net/netip"
	"slices"
)

// AttachmentMode controls how NIAC's untagged interface is externally isolated.
type AttachmentMode string

// Attachment modes for NIAC's untagged interface: unisolated, single-VLAN
// access, or 802.1Q trunk carrying the compiled network's VLAN set.
const (
	ModeDirect AttachmentMode = "direct"
	ModeAccess AttachmentMode = "access"
	ModeTrunk  AttachmentMode = "trunk"
)

// DiagnosticCode identifies a stable compiler finding.
type DiagnosticCode string

// Stable diagnostic codes the fabric compiler attaches to configuration
// errors, so API/CLI callers can classify a finding without string-matching
// its message.
const (
	CodeAttachmentPolicyDenied   DiagnosticCode = "attachment_policy_denied"
	CodeHostInterfaceUnavailable DiagnosticCode = "host_interface_unavailable"
	CodeInvalidAccessVLAN        DiagnosticCode = "invalid_access_vlan"
	CodeInvalidAttachmentMode    DiagnosticCode = "invalid_attachment_mode"
	CodeUnknownAttachment        DiagnosticCode = "unknown_attachment"
	CodeUnknownNetwork           DiagnosticCode = "unknown_network"
	CodeInvalidNetwork           DiagnosticCode = "invalid_network"
	CodeInvalidVirtualVLAN       DiagnosticCode = "invalid_virtual_vlan"
	CodeDuplicateNetwork         DiagnosticCode = "duplicate_network"
	CodeDuplicateDevice          DiagnosticCode = "duplicate_device"
	CodeDuplicateInterface       DiagnosticCode = "duplicate_interface"
	CodeDuplicateInterfaceAddr   DiagnosticCode = "duplicate_interface_address"
	CodeOverlappingNetworks      DiagnosticCode = "overlapping_networks"
	CodeInvalidInterfaceAddress  DiagnosticCode = "invalid_interface_address"
	CodeAddressOutsideNetwork    DiagnosticCode = "address_outside_network"
	CodeInterfacePrefixMismatch  DiagnosticCode = "interface_prefix_mismatch"
	CodeReservedInterfaceAddr    DiagnosticCode = "reserved_interface_address"
	CodeUnknownRouteInterface    DiagnosticCode = "unknown_route_interface"
	CodeInvalidRoute             DiagnosticCode = "invalid_route"
	CodeInvalidRouteNextHop      DiagnosticCode = "invalid_route_next_hop"
	CodeRouteNextHopOffLink      DiagnosticCode = "route_next_hop_off_link"
	CodeUnknownRouteNextHop      DiagnosticCode = "unknown_route_next_hop"
	CodeRouteNextHopSelf         DiagnosticCode = "route_next_hop_self"
	CodeDHCPNetworkAmbiguous     DiagnosticCode = "dhcp_network_ambiguous"
	CodeDHCPPoolOutsideNetwork   DiagnosticCode = "dhcp_pool_outside_network"
	CodeInvalidDHCPRange         DiagnosticCode = "invalid_dhcp_range"
	CodeInvalidDHCPRouter        DiagnosticCode = "invalid_dhcp_router"
	CodeInvalidDHCPLease         DiagnosticCode = "invalid_dhcp_lease"
	CodeInvalidDHCPOption        DiagnosticCode = "invalid_dhcp_option"
	CodeReservedDHCPAddress      DiagnosticCode = "reserved_dhcp_address"
	CodeDHCPAddressCollision     DiagnosticCode = "dhcp_address_collision"
)

// Binding maps a scenario attachment to one physical deployment interface.
type Binding struct {
	Attachment     string         `json:"attachment"`
	Interface      string         `json:"interface"`
	Mode           AttachmentMode `json:"mode"`
	AccessVLAN     uint16         `json:"physicalVlan,omitempty"`
	PolicyApproved bool           `json:"-"`
}

// PhysicalAttachmentPolicy is an operator-owned permission for one exact host attachment.
type PhysicalAttachmentPolicy struct {
	Interface    string
	Mode         AttachmentMode
	AccessVLAN   uint16
	AllowedVLANs []uint16
}

// Approves reports whether the policy exactly matches a requested physical binding.
func (p PhysicalAttachmentPolicy) Approves(binding Binding) bool {
	if p.Interface != binding.Interface || p.Mode != binding.Mode {
		return false
	}
	if p.Mode != ModeTrunk {
		return p.AccessVLAN == binding.AccessVLAN
	}
	return slices.Contains(p.AllowedVLANs, binding.AccessVLAN)
}

// CompiledBinding is the physical exposure contract shown by preflight.
type CompiledBinding struct {
	Binding

	Network    string `json:"network"`
	WireTagged bool   `json:"wireTagged"`
}

// Network is one canonical internal IPv4 network.
type Network struct {
	Name        string       `json:"name"`
	Prefix      netip.Prefix `json:"prefix"`
	VirtualVLAN uint16       `json:"virtualVlan,omitempty"`
}

// Interface is one device attachment to a virtual network.
type Interface struct {
	Device  string       `json:"device"`
	Name    string       `json:"name"`
	Network string       `json:"network"`
	Address netip.Prefix `json:"address"`
}

// Route is one connected or authored static IPv4 route.
type Route struct {
	Device      string       `json:"device"`
	Destination netip.Prefix `json:"destination"`
	Via         string       `json:"via"`
	NextHop     netip.Addr   `json:"nextHop,omitzero"`
	Connected   bool         `json:"connected"`
}

// DHCPScope is the network ownership established for one existing DHCP server.
type DHCPScope struct {
	Device  string     `json:"device"`
	Network string     `json:"network"`
	Start   netip.Addr `json:"start"`
	End     netip.Addr `json:"end"`
	Router  netip.Addr `json:"router,omitzero"`
}

// Topology is the immutable result consumed by preflight and later forwarding.
type Topology struct {
	Binding    CompiledBinding `json:"binding"`
	Networks   []Network       `json:"networks"`
	Interfaces []Interface     `json:"interfaces"`
	Routes     []Route         `json:"routes"`
	DHCPScopes []DHCPScope     `json:"dhcpScopes"`
}

// NewTopology returns a Topology whose collections are empty slices rather
// than nil. encoding/json renders a nil slice as `null`, and the wire contract
// (mirrored by ui/src/api/fabric-types.ts) declares these as arrays, so a
// consumer doing `topology.networks.length` must never be handed null (D6).
func NewTopology() Topology {
	return Topology{
		Networks:   []Network{},
		Interfaces: []Interface{},
		Routes:     []Route{},
		DHCPScopes: []DHCPScope{},
	}
}

// Diagnostic explains why a topology is unsafe.
type Diagnostic struct {
	Code    DiagnosticCode `json:"code"`
	Field   string         `json:"field"`
	Message string         `json:"message"`
}

// Report is returned for both safe and unsafe compiler inputs.
type Report struct {
	Safe        bool         `json:"safe"`
	Topology    Topology     `json:"topology"`
	Diagnostics []Diagnostic `json:"diagnostics"`
}
