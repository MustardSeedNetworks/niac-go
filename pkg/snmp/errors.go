package snmp

import "errors"

// Sentinel errors for agent operations.
var (
	ErrAgentNotInitialized = errors.New("agent not initialized")
	ErrNoWalkFileSpecified = errors.New("no walk file specified")
	ErrNoSuchObject        = errors.New("no such object")
	ErrEndOfMIBView        = errors.New("end of MIB view")
	ErrEmptyOID            = errors.New("empty OID")
	ErrInvalidOIDComponent = errors.New("invalid OID component")
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
