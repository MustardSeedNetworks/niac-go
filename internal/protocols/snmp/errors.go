package snmp

import "errors"

// Sentinel errors for agent operations.
var (
	ErrAgentNotInitialized = errors.New("agent not initialized")
	ErrNoWalkFileSpecified = errors.New("no walk file specified")
	ErrNoSuchObject        = errors.New("no such object")
	ErrEndOfMIBView        = errors.New("end of MIB view")
	ErrEmptyOID            = errors.New("empty OID")

	// ErrV3Disabled reports a v3 operation on a device with no v3 engine,
	// which is what "snmpv3 is not configured on this device" looks like to a
	// caller that asked for a v3 notification.
	ErrV3Disabled = errors.New("snmpv3 is not enabled on this device")

	// ErrV3UnknownUser reports a notification addressed to a USM user the
	// device does not have.
	ErrV3UnknownUser       = errors.New("snmpv3 user not configured")
	ErrInvalidOIDComponent = errors.New("invalid OID component")
	ErrInvalidValue        = errors.New("invalid value")
	// ErrNotWritable reports a manager trying to write an object this agent
	// serves read-only, which is what a real device answers for almost all of
	// its MIB.
	ErrNotWritable = errors.New("object is not writable")
)

// Sentinel errors for addmib operations.
var (
	ErrInvalidVarimibValue     = errors.New("invalid varimib value")
	ErrInvalidVarimibIntervals = errors.New("invalid varimib intervals")
	ErrNoVarimibPairs          = errors.New("no varimib pairs")
	ErrUnsupportedSNMPType     = errors.New("unsupported SNMP type")
	ErrInvalidIPAddress        = errors.New("invalid IP address")
)

// Sentinel errors for mibzip operations.
var (
	ErrDataTooShortForMagic = errors.New("data too short for mibzip magic")
	ErrInvalidMibzipMagic   = errors.New("invalid mibzip magic")
	ErrExpectedDownCommand  = errors.New("expected initial DOWN command")
	ErrExpectedISONode      = errors.New("expected iso(1)")
	ErrUnknownMibzipCommand = errors.New("unknown mibzip command")
	ErrLengthTooLong        = errors.New("length too long")
	ErrDirectoryTraversal   = errors.New("invalid path: directory traversal not allowed")
)

// Sentinel errors for trap operations.
var (
	ErrTrapConfigDisabled = errors.New("trap configuration disabled or not provided")
	ErrNoTrapReceivers    = errors.New("no trap receivers configured")
	ErrTrapSenderRunning  = errors.New("trap sender already running")
)

// Sentinel errors for varimib operations.
var (
	ErrInvalidCentiseconds  = errors.New("invalid centiseconds")
	ErrInvalidVarimibFormat = errors.New("invalid varimib format")
	ErrInvalidInteger       = errors.New("invalid integer")
	ErrInvalidUnsignedInt   = errors.New("invalid unsigned integer")
	ErrInvalidCounter64     = errors.New("invalid counter64")
	ErrInvalidTimeticks     = errors.New("invalid timeticks")
	ErrNoPatternMatch       = errors.New("no pattern match")
)

// Sentinel errors for walk file operations.
var (
	ErrWalkFileSymlink        = errors.New("walk file cannot be a symbolic link")
	ErrWalkFileIsDirectory    = errors.New("walk file path is a directory, not a file")
	ErrInvalidWalkFormat      = errors.New("invalid format: missing '='")
	ErrMissingColon           = errors.New("invalid format: missing ':'")
	ErrFailedToParseValue     = errors.New("failed to parse value")
	ErrInvalidTimeticksFormat = errors.New("invalid Timeticks format")
	ErrFailedToCreateWalkFile = errors.New("failed to create walk file")
	ErrFailedToWriteEntry     = errors.New("failed to write entry")
)

// Sentinel errors for walk validator operations.
var (
	ErrFailedToAccessWalkFile = errors.New("failed to access walk file")
	ErrFailedToOpenWalkFile   = errors.New("failed to open walk file")
	ErrReadingWalkFile        = errors.New("error reading walk file")
	ErrFailedToReadWalkFile   = errors.New("failed to read walk file")
	ErrFailedToCreateBackup   = errors.New("failed to create backup")
	ErrFailedToWriteFixedFile = errors.New("failed to write fixed file")
	ErrFailedToParseWalkFile  = errors.New("failed to parse walk file")
)
