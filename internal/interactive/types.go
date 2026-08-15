package interactive

import (
	"time"

	"charm.land/lipgloss/v2"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/protocols"
	"github.com/MustardSeedNetworks/niac-go/internal/templates"
)

// Styles.
var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("170")).
			Background(lipgloss.Color("235")).
			Padding(0, 1)

	deviceStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("86"))

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true)

	successStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("82")).
			Bold(true)

	selectedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("170")).
			Bold(true)

	statsStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("246"))

	// Validation styles.
	validationErrorStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("196")) // Red for errors

	validationWarningStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("226")) // Yellow for warnings

	validationInfoStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("51")) // Cyan for info

	validationSuccessStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("82")).
				Bold(true) // Green for success

	// Config diff styles.
	diffAddedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("82")) // Green for added

	diffRemovedStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("196")) // Red for removed

	diffHeaderStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("39")).
			Bold(true) // Blue for headers

	// Topology styles.
	topologyNodeStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("86"))

	topologyLinkStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("246"))
)

// CapturedPacket stores packet data for hex dump viewer.
type CapturedPacket struct {
	Timestamp time.Time
	Protocol  string
	SrcAddr   string
	DstAddr   string
	Length    int
	Data      []byte
}

// HistoryEntry stores data about a past simulation run.
type HistoryEntry struct {
	StartTime       time.Time
	EndTime         time.Time
	ConfigFile      string
	DeviceCount     int
	PacketsSent     uint64
	PacketsReceived uint64
	ErrorsInjected  int
}

const (
	maxPacketBuffer = 20 // Keep last 20 packets

	// Key constants for keyboard input handling.
	keyEsc   = "esc"
	keyDown  = "down"
	keyEnter = "enter"

	// Format constants.
	formatJSON = "json"

	// Search category constants.
	searchCategoryAll     = "all"
	searchCategoryDevices = "devices"
	searchCategoryLogs    = "logs"

	// Search status and scope constants.
	searchCancelled   = "Search cancelled"
	searchScopeDevice = "device"

	// Default IP placeholder when no IP is configured.
	noIPPlaceholder = "no-ip"

	// Numeric constants for display and validation.
	maxInputDigits       = 3   // Max digits for 0-100 input
	minutesPerHour       = 60  // Minutes in an hour
	secondsPerMinute     = 60  // Seconds in a minute
	minEllipsisWidth     = 3   // Minimum width to show ellipsis
	maxHoursNoMinutes    = 100 // Hours threshold for simplified display
	maxDebugLogs         = 100 // Maximum debug log entries to keep
	displayedLogCount    = 10  // Number of logs to display
	panelContentWidth    = 64  // Width for panel content
	menuPaddingWidth     = 64  // Width for menu padding
	validationTruncate   = 62  // Max length for validation messages
	maxValidationDisplay = 8   // Max validation items to display
	minDiffLinesForNoOp  = 4   // Minimum diff lines to detect no changes
	maxTruncateName      = 12  // Max name length in topology diagram

	// Debug level constants.
	debugLevelVerbose = 2
	debugLevelDebug   = 3
	debugLevelCount   = 4 // Total number of debug levels

	// Column width constants for neighbor display.
	colWidthProto       = 5
	colWidthLocalDevice = 14
	colWidthRemote      = 18
	colWidthMgmt        = 15
	colWidthSeen        = 8

	// Hex dump constants.
	hexBytesPerLine = 16 // Bytes per line in hex dump

	// Search result display constants.
	maxSearchResultWidth = 64

	// Alert threshold constants.
	alertThresholdMax              = 100
	alertDefaultCPU                = 80
	alertDefaultMemory             = 85
	alertDefaultDisk               = 90
	alertDefaultPacketLoss         = 5
	alertDefaultLatency            = 100
	alertDefaultThresholdInDisplay = 80

	// Playback speed constants.
	maxPlaybackSpeed = 4.0
	minPlaybackSpeed = 0.25

	// History view constants.
	historyRowWidth    = 64
	historyVisibleRows = 8

	// SNMP walk constants.
	snmpVisibleRows       = 12
	snmpValueTruncateLen  = 30
	snmpValueDisplayWidth = 27

	// Device config constants.
	deviceConfigTableWidth   = 50
	deviceConfigTabCount     = 4
	deviceConfigTabGeneral   = 0
	deviceConfigTabInterface = 1
	deviceConfigTabProtocols = 2
	deviceConfigTabSNMP      = 3

	// Uptime timeticks multiplier.
	uptimeTicksMultiplier = 100
)

// SnmpOidEntry represents a single SNMP OID entry for the walk browser.
type SnmpOidEntry struct {
	OID         string // Full OID string (e.g., "1.3.6.1.2.1.1.1.0")
	Name        string // Human-readable name if available (e.g., "sysDescr")
	Value       string // Current value as string
	Type        string // SNMP type (STRING, INTEGER, etc.)
	Depth       int    // Depth in tree for indentation
	Expanded    bool   // Whether this branch is expanded
	HasChildren bool   // Whether this OID has children
}

type stackStatsSnapshot struct {
	PacketsReceived uint64
	PacketsSent     uint64
	ARPRequests     uint64
	ARPReplies      uint64
	ICMPRequests    uint64
	ICMPReplies     uint64
	DNSQueries      uint64
	DHCPRequests    uint64
}

type model struct {
	cfg           *config.Config
	interfaceName string
	debugLevel    int
	stack         *protocols.Stack
	reloadFunc    func() (*config.Config, error)

	// Menu state
	menuVisible      bool
	menuItems        []string
	selectedItem     int
	valueInputMode   bool
	valueInputPrompt string
	valueInputBuffer string

	// View state
	showHelp      bool
	showLogs      bool
	showStats     bool
	showHexDump   bool
	showNeighbors bool

	// Error injection state
	selectedDeviceIdx int
	selectedErrorType int

	// Stats
	stackStats      stackStatsSnapshot
	packetsInjected int
	errorsActive    int
	uptime          time.Duration
	startTime       time.Time

	// Logs
	debugLogs []string

	// Status
	statusMessage string
	statusIsError bool

	// Hex dump viewer state
	packetBuffer       []CapturedPacket
	hexDumpPacketIndex int
	hexDumpScrollY     int

	neighbors []protocols.NeighborRecord

	// Validation state
	showValidation    bool
	validationResults *config.ListError

	// Template browser state
	showTemplates          bool
	templateList           []templates.TemplateMetadata
	selectedTemplate       int
	showTemplatePreview    bool
	templatePreviewContent string

	// Config diff viewer state
	showConfigDiff    bool
	configDiffContent []string
	configDiffScrollY int
	previousConfig    *config.Config
	configFilePath    string

	// Search mode state
	showSearch     bool
	searchMode     bool
	searchQuery    string
	searchResults  []searchResult
	selectedResult int
	searchCategory string // "devices", "logs", "all"

	// Export state
	showExport     bool
	exportFormat   string // "csv", "json"
	lastExportPath string
	lastExportTime time.Time

	// Topology view state
	showTopology    bool
	topologyContent string
	topologyScrollY int

	// PCAP replay state
	showPcapReplay    bool
	pcapPackets       []CapturedPacket
	pcapPlaybackIndex int
	pcapPlaying       bool
	pcapPlaybackSpeed float64

	// Alert configuration state
	showAlertConfig   bool
	alertThresholds   map[string]int
	alertsEnabled     bool
	selectedAlertType int

	// Device config panel state
	showDeviceConfig    bool
	deviceConfigTab     int // 0=General, 1=Interfaces, 2=Protocols, 3=SNMP
	deviceConfigScrollY int

	// History view state
	showHistory        bool
	historyEntries     []HistoryEntry
	selectedHistoryIdx int
	historyScrollY     int

	// SNMP walk browser state
	showSnmpWalk    bool
	snmpOidTree     []SnmpOidEntry
	selectedSnmpOid int
	snmpScrollY     int
}

type (
	tickMsg   time.Time
	reloadMsg struct {
		cfg *config.Config
		err error
	}
)

// editorFinishedMsg is sent when the external editor finishes.
type editorFinishedMsg struct {
	err error
}

// searchResult represents a single search match.
type searchResult struct {
	Category string // "device", "log", "error", "neighbor"
	Title    string
	Detail   string
	Index    int
}
