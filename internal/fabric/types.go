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

	// CodeAttachmentFormAmbiguous and the codes below it are the
	// attachment-pool findings. All of them describe the scenario file, so none
	// belongs on bindingDiagnosticCodes().
	CodeAttachmentFormAmbiguous         DiagnosticCode = "attachment_form_ambiguous"
	CodeUnknownAttachmentDevice         DiagnosticCode = "unknown_attachment_device"
	CodeUnknownAttachmentPort           DiagnosticCode = "unknown_attachment_port"
	CodeDuplicateAttachmentPort         DiagnosticCode = "duplicate_attachment_port"
	CodeAttachmentPoolEmpty             DiagnosticCode = "attachment_pool_empty"
	CodeAttachmentPortOccupied          DiagnosticCode = "attachment_port_occupied"
	CodeAttachmentPortVLANUnresolved    DiagnosticCode = "attachment_port_vlan_unresolved"
	CodeAttachmentPortNetworkUnresolved DiagnosticCode = "attachment_port_network_unresolved"
	CodeAttachmentPortNetworkAmbiguous  DiagnosticCode = "attachment_port_network_ambiguous"
	CodeAttachmentPoolNetworksDiffer    DiagnosticCode = "attachment_pool_networks_differ"
	CodeAttachmentPinOutsidePool        DiagnosticCode = "attachment_pin_outside_pool"
	CodeAttachmentPinDuplicate          DiagnosticCode = "attachment_pin_duplicate"
	CodeInvalidAttachmentPinMAC         DiagnosticCode = "invalid_attachment_pin_mac"
)

// bindingDiagnosticCodes are the findings that describe a physical deployment
// rather than the scenario file. Every other code is config-scoped.
//
// The split is the contract behind CompileConfig: an authoring surface has no
// binding, so it reports every config-scoped code a later preflight of the
// same file will report, and none of these. A new code belongs on this list or
// it is config-scoped by default -- decide when you add it, not when a surface
// disagrees.
func bindingDiagnosticCodes() []DiagnosticCode {
	return []DiagnosticCode{
		CodeAttachmentPolicyDenied,
		CodeHostInterfaceUnavailable,
		CodeInvalidAccessVLAN,
		CodeInvalidAttachmentMode,
		CodeUnknownAttachment,
	}
}

// IsBinding reports whether the code describes the physical deployment rather
// than the scenario file.
func (c DiagnosticCode) IsBinding() bool {
	return slices.Contains(bindingDiagnosticCodes(), c)
}

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

// AttachmentPort is one free port a tester can appear on, with the network
// that port lands a client on already resolved.
//
// VLAN here is the *port* VLAN -- which network, DHCP scope and gateway a
// client gets inside the scenario. It is a different namespace from the wire
// VLAN on Binding, which describes how the NIAC host is cabled upstream. The
// two are independent and must never be compared.
type AttachmentPort struct {
	Device    string `json:"device"`
	Interface string `json:"interface"`
	VLAN      uint16 `json:"vlan,omitempty"`
	Network   string `json:"network"`
}

// AttachmentPin is one client MAC fixed to one port of the pool.
type AttachmentPin struct {
	MAC       string `json:"mac"`
	Device    string `json:"device"`
	Interface string `json:"interface"`
}

// CompiledAttachment is a port-scoped attachment with every port resolved.
type CompiledAttachment struct {
	Name    string           `json:"name"`
	Device  string           `json:"device"`
	Network string           `json:"network"`
	Ports   []AttachmentPort `json:"ports"`
	Pins    []AttachmentPin  `json:"pins,omitempty"`
}

// Topology is the immutable result consumed by preflight and later forwarding.
type Topology struct {
	Binding     CompiledBinding      `json:"binding"`
	Networks    []Network            `json:"networks"`
	Interfaces  []Interface          `json:"interfaces"`
	Routes      []Route              `json:"routes"`
	DHCPScopes  []DHCPScope          `json:"dhcpScopes"`
	Attachments []CompiledAttachment `json:"attachments"`
}

// NewTopology returns a Topology whose collections are empty slices rather
// than nil. encoding/json renders a nil slice as `null`, and the wire contract
// (mirrored by ui/src/api/fabric-types.ts) declares these as arrays, so a
// consumer doing `topology.networks.length` must never be handed null (D6).
func NewTopology() Topology {
	return Topology{
		Networks:    []Network{},
		Interfaces:  []Interface{},
		Routes:      []Route{},
		DHCPScopes:  []DHCPScope{},
		Attachments: []CompiledAttachment{},
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

// NewReport returns a Report whose collections are empty rather than nil.
//
// The UI declares all five as non-nullable arrays, and a nil Go slice marshals
// as null — omitempty does not help, the fields have to be initialised. Build
// every Report through this so the wire contract holds by construction instead
// of by each caller remembering (#1467).
func NewReport() Report {
	return Report{Topology: NewTopology(), Diagnostics: []Diagnostic{}}
}
