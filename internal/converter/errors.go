package converter

import "errors"

// Sentinel errors for converter.
var (
	ErrInvalidLoopTimeFormat      = errors.New("invalid LoopTime format")
	ErrInvalidScaleTimeFormat     = errors.New("invalid ScaleTime format")
	ErrInvalidVlanFormat          = errors.New("invalid Vlan format")
	ErrDeviceMissingMAC           = errors.New("device missing MAC address")
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
