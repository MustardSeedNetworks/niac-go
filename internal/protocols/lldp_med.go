package protocols

import (
	"encoding/binary"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
)

// LLDP-MED (TIA-1057) organizationally specific TLVs.
//
// These are what a discovery tool reads to tell a phone from a printer, and to
// learn the voice VLAN an endpoint belongs on. NIAC advertised plain 802.1AB
// only, so every simulated phone, camera and access point appeared as an
// anonymous endpoint and a pack meant to look like a hospital showed a rack of
// unidentified MAC addresses.

// tiaOUI is the TIA organizationally unique identifier that marks a TLV as
// LLDP-MED.
//
//nolint:gochecknoglobals // a protocol constant that must be a slice.
var tiaOUI = []byte{0x00, 0x12, 0xBB}

// LLDP-MED subtypes, per TIA-1057.
const (
	medSubtypeCapabilities     = 1
	medSubtypeNetworkPolicy    = 2
	medSubtypeExtendedPowerMDI = 4
	medSubtypeHardwareRevision = 5
	medSubtypeFirmwareRevision = 6
	medSubtypeSoftwareRevision = 7
	medSubtypeSerialNumber     = 8
	medSubtypeManufacturer     = 9
	medSubtypeModelName        = 10
	medSubtypeAssetID          = 11
)

// MED capability bits, advertised in the Capabilities TLV.
const (
	medCapCapabilities  = 1 << 0
	medCapNetworkPolicy = 1 << 1
	medCapPowerPSE      = 1 << 3
	medCapPowerPD       = 1 << 4
	medCapInventory     = 1 << 5
)

// MED device classes.
const (
	medDeviceTypeNotDefined          = 0
	medDeviceTypeEndpointClass1      = 1
	medDeviceTypeEndpointClass2      = 2
	medDeviceTypeEndpointClass3      = 3
	medDeviceTypeNetworkConnectivity = 4
)

// buildMEDTLVs returns every LLDP-MED TLV the device advertises, in the order
// TIA-1057 requires: Capabilities first, then policy, power and inventory.
//
// Capabilities is not a summary written by hand — it is derived from what the
// rest of this function will actually emit. Advertising a capability the frame
// does not then carry is the kind of disagreement a receiver reports as a
// malformed endpoint.
func (h *LLDPHandler) buildMEDTLVs(device *config.Device) []byte {
	med := medConfig(device)
	if med == nil {
		return nil
	}

	var tlvs []byte
	tlvs = append(tlvs, h.buildMEDCapabilitiesTLV(med)...)
	for i := range med.NetworkPolicies {
		tlvs = append(tlvs, h.buildMEDNetworkPolicyTLV(&med.NetworkPolicies[i])...)
	}
	if med.Power != nil {
		tlvs = append(tlvs, h.buildMEDPowerTLV(med.Power)...)
	}
	tlvs = append(tlvs, h.buildMEDInventoryTLVs(med.Inventory)...)

	return tlvs
}

// medConfig returns the device's MED block, or nil when it advertises none.
func medConfig(device *config.Device) *config.LLDPMEDConfig {
	if device == nil || device.LLDPConfig == nil {
		return nil
	}

	return device.LLDPConfig.MED
}

// buildMEDCapabilitiesTLV advertises which MED extensions follow.
func (h *LLDPHandler) buildMEDCapabilitiesTLV(med *config.LLDPMEDConfig) []byte {
	capabilities := uint16(medCapCapabilities)
	if len(med.NetworkPolicies) > 0 {
		capabilities |= medCapNetworkPolicy
	}
	if med.Power != nil {
		if med.Power.DeviceType == "pse" {
			capabilities |= medCapPowerPSE
		} else {
			capabilities |= medCapPowerPD
		}
	}
	if inventoryValues(med.Inventory) != nil {
		capabilities |= medCapInventory
	}

	value := binary.BigEndian.AppendUint16(nil, capabilities)
	value = append(value, medDeviceType(med.DeviceType))

	return medTLV(medSubtypeCapabilities, value)
}

// medDeviceType maps the configured class onto its TIA-1057 code.
func medDeviceType(deviceType string) byte {
	switch deviceType {
	case "endpoint_class1":
		return medDeviceTypeEndpointClass1
	case "endpoint_class2":
		return medDeviceTypeEndpointClass2
	case "endpoint_class3":
		return medDeviceTypeEndpointClass3
	case "network_connectivity":
		return medDeviceTypeNetworkConnectivity
	default:
		return medDeviceTypeNotDefined
	}
}

// medApplicationTypes maps the configured application onto its TIA-1057 code.
//
//nolint:gochecknoglobals // a lookup table, read-only after init.
var medApplicationTypes = map[string]byte{
	"voice":                 medAppVoice,
	"voice_signaling":       medAppVoiceSignaling,
	"guest_voice":           medAppGuestVoice,
	"guest_voice_signaling": medAppGuestVoiceSignaling,
	"softphone_voice":       medAppSoftphoneVoice,
	"video_conferencing":    medAppVideoConferencing,
	"streaming_video":       medAppStreamingVideo,
	"video_signaling":       medAppVideoSignaling,
}

// TIA-1057 application types.
const (
	medAppVoice               = 1
	medAppVoiceSignaling      = 2
	medAppGuestVoice          = 3
	medAppGuestVoiceSignaling = 4
	medAppSoftphoneVoice      = 5
	medAppVideoConferencing   = 6
	medAppStreamingVideo      = 7
	medAppVideoSignaling      = 8
)

// Network Policy field widths, packed into three bytes after the application
// type: U | T | X | VLAN(12) | priority(3) | DSCP(6).
const (
	medPolicyUnknownBit  = 1 << 23
	medPolicyTaggedBit   = 1 << 22
	medPolicyVLANShift   = 9
	medPolicyVLANMask    = 0xFFF
	medPolicyPriShift    = 6
	medPolicyPriMask     = 0x7
	medPolicyDSCPMask    = 0x3F
	medPolicyValueBytes  = 3
	medPolicyTopByteMask = 0xFF
)

// buildMEDNetworkPolicyTLV advertises the VLAN, priority and DSCP one
// application uses.
func (h *LLDPHandler) buildMEDNetworkPolicyTLV(policy *config.LLDPMEDNetworkPolicy) []byte {
	application, known := medApplicationTypes[policy.Application]
	if !known {
		// An application NIAC does not know is dropped rather than sent as
		// type 0: a receiver reads 0 as a malformed policy and may discard
		// every TLV after it.
		return nil
	}

	var packed uint32
	if policy.Unknown {
		packed |= medPolicyUnknownBit
	}
	if policy.Tagged {
		packed |= medPolicyTaggedBit
	}
	packed |= uint32(policy.VLANID&medPolicyVLANMask) << medPolicyVLANShift
	packed |= uint32(policy.Priority&medPolicyPriMask) << medPolicyPriShift
	packed |= uint32(policy.DSCP & medPolicyDSCPMask)

	value := []byte{
		application,
		byte(packed >> 16 & medPolicyTopByteMask),
		byte(packed >> 8 & medPolicyTopByteMask),
		byte(packed & medPolicyTopByteMask),
	}

	return medTLV(medSubtypeNetworkPolicy, value)
}

// Extended Power-via-MDI field widths: type(2) | source(2) | priority(4), then
// the value in tenths of a watt.
const (
	medPowerTypeShift   = 6
	medPowerSourceShift = 4
	medPowerPriMask     = 0xF

	medPowerTypePSE = 0
	medPowerTypePD  = 1

	// medMaxPowerTenthWatts is the widest value the two-byte field holds. The
	// schema bounds the config to this too; clamping here as well means a value
	// that reached the stack another way cannot wrap into a small one.
	medMaxPowerTenthWatts = 1023
)

// buildMEDPowerTLV advertises the device's power role and draw.
func (h *LLDPHandler) buildMEDPowerTLV(power *config.LLDPMEDPower) []byte {
	powerType := byte(medPowerTypePSE)
	if power.DeviceType == "pd" {
		powerType = medPowerTypePD
	}

	flags := powerType<<medPowerTypeShift |
		medPowerSource(power.Source)<<medPowerSourceShift |
		medPowerPriority(power.Priority)&medPowerPriMask

	tenthWatts := min(max(power.ValueTenthWatts, 0), medMaxPowerTenthWatts)

	value := []byte{flags}
	value = binary.BigEndian.AppendUint16(value, uint16(tenthWatts))

	return medTLV(medSubtypeExtendedPowerMDI, value)
}

// medPowerSource maps the configured source onto its two-bit code.
func medPowerSource(source string) byte {
	switch source {
	case "primary", "pse":
		return medPowerSourcePrimary
	case "backup", "local":
		return medPowerSourceBackup
	case "pse_local":
		return medPowerSourceBoth
	default:
		return medPowerSourceUnknown
	}
}

// TIA-1057 power source codes. Their meaning depends on the device type: for a
// PD they read PSE / local / both, for a PSE primary / backup.
const (
	medPowerSourceUnknown = 0
	medPowerSourcePrimary = 1
	medPowerSourceBackup  = 2
	medPowerSourceBoth    = 3
)

// medPowerPriority maps the configured priority onto its code.
func medPowerPriority(priority string) byte {
	switch priority {
	case "critical":
		return medPowerPriorityCritical
	case "high":
		return medPowerPriorityHigh
	case "low":
		return medPowerPriorityLow
	default:
		return medPowerPriorityUnknown
	}
}

// TIA-1057 power priority codes.
const (
	medPowerPriorityUnknown  = 0
	medPowerPriorityCritical = 1
	medPowerPriorityHigh     = 2
	medPowerPriorityLow      = 3
)

// buildMEDInventoryTLVs emits one TLV per non-empty inventory field.
//
// An empty field is omitted rather than sent blank: a receiver shows a
// zero-length serial number as a serial number it read, which is worse than
// showing none.
func (h *LLDPHandler) buildMEDInventoryTLVs(inventory *config.LLDPMEDInventory) []byte {
	var tlvs []byte
	for _, entry := range inventoryValues(inventory) {
		tlvs = append(tlvs, medTLV(entry.subtype, []byte(entry.value))...)
	}

	return tlvs
}

// inventoryEntry pairs an inventory field with the subtype that carries it.
type inventoryEntry struct {
	subtype byte
	value   string
}

// inventoryValues returns the inventory fields that are set, in subtype order.
// Nil when the device advertises none, which is what the capability bit reads.
func inventoryValues(inventory *config.LLDPMEDInventory) []inventoryEntry {
	if inventory == nil {
		return nil
	}

	candidates := []inventoryEntry{
		{medSubtypeHardwareRevision, inventory.HardwareRevision},
		{medSubtypeFirmwareRevision, inventory.FirmwareRevision},
		{medSubtypeSoftwareRevision, inventory.SoftwareRevision},
		{medSubtypeSerialNumber, inventory.SerialNumber},
		{medSubtypeManufacturer, inventory.Manufacturer},
		{medSubtypeModelName, inventory.ModelName},
		{medSubtypeAssetID, inventory.AssetID},
	}

	var set []inventoryEntry
	for _, candidate := range candidates {
		if candidate.value != "" {
			set = append(set, candidate)
		}
	}

	return set
}

// medTLV wraps a MED payload in the organizationally specific TLV header:
// type 127, then the TIA OUI and the subtype.
func medTLV(subtype byte, value []byte) []byte {
	body := make([]byte, 0, len(tiaOUI)+1+len(value))
	body = append(body, tiaOUI...)
	body = append(body, subtype)
	body = append(body, value...)

	return buildTLV(LLDPTLVTypeOrganizationSpecific, body)
}

// buildTLV writes the 802.1AB TLV header around body.
//
// The length is nine bits and straddles both header bytes. The older builders
// in lldp.go write `type << 1` and drop the high bit, which is correct only
// while every TLV stays under 256 bytes; a MED inventory string plus the OUI
// header can be handed a longer value, and silently truncating the length is
// how a decoder is sent off the end of a frame.
func buildTLV(tlvType byte, body []byte) []byte {
	if len(body) > lldpMaxTLVLength {
		body = body[:lldpMaxTLVLength]
	}
	length := len(body)

	// length is capped at lldpMaxTLVLength above, so both halves fit a byte.
	tlv := make([]byte, 0, lldpTLVHeaderSize+length)
	tlv = append(tlv,
		tlvType<<1|byte(length>>tlvLengthHighShift&tlvLengthLowMask),
		byte(length&tlvLengthLowMask),
	)

	return append(tlv, body...)
}

// The TLV length field's split across the two header bytes.
const (
	tlvLengthHighShift = 8
	tlvLengthLowMask   = 0xFF
)
