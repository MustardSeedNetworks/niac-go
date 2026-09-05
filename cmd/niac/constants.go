package main

// Shared numeric literals, named so the linter's mnd rule and the reader both
// get what they need.
const (
	// Argument counts for cobra commands.
	argsCountTwo   = 2
	argsCountThree = 3

	// Exit codes.
	exitCodeError = 2

	// Time constants.
	secondsPerMinute    = 60
	shortTimeout        = 5   // seconds
	tickerInterval      = 2   // seconds
	statsTickerInterval = 10  // seconds for stats reporting
	httpReadTimeout     = 15  // seconds for HTTP read timeout
	logPollMilliseconds = 500 // milliseconds for log polling
	maxLogEntries       = 100
	maxLogWidth         = 500

	// Display widths.
	lineWidthStandard = 80
	lineWidthWide     = 90
	tabPadding        = 2
	colWidthMAC       = 18
	colWidthIP        = 15
	colWidthType      = 17
	colWidthVendor    = 8
	colWidthHelp      = 51

	// Other constants.
	protocolCapacity      = 9
	templatePadOffset     = 2
	minPageLen            = 20
	maxDeviceCount        = 20 // maximum devices in generated config
	millisecondsThreshold = 1000
	randomBound           = 10 // for rand.Intn
	baseIPOffset          = 10 // offset for generated device IPs
	cidrParts             = 2  // IP/mask CIDR notation parts
	minArgsForConfig      = 2  // minimum arguments for config operations
	minSaltLen            = 5  // minimum salt length for hashing
	hexCharsPerByte       = 2  // 2 hex characters per byte
	ipRegexParts          = 5  // full match + 4 IP octets in regex
)
