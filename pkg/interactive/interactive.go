// Package interactive provides a terminal user interface for network simulation control and monitoring
package interactive

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/krisarmstrong/niac-go/pkg/api"
	"github.com/krisarmstrong/niac-go/pkg/config"
	"github.com/krisarmstrong/niac-go/pkg/errors"
	"github.com/krisarmstrong/niac-go/pkg/logging"
	"github.com/krisarmstrong/niac-go/pkg/protocols"
	"github.com/krisarmstrong/niac-go/pkg/templates"
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

		// Search styles.
	searchMatchStyle = lipgloss.NewStyle().
				Background(lipgloss.Color("226")).
				Foreground(lipgloss.Color("0")) // Yellow highlight

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

	// Default IP placeholder when no IP is configured.
	noIPPlaceholder = "no-ip"
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
	stateManager  *errors.StateManager
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
	validationResults *config.ConfigErrorList

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
	pcapFilePath      string
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
	showSnmpWalk     bool
	snmpOidTree      []SnmpOidEntry
	selectedSnmpOid  int
	snmpScrollY      int
	snmpExpandedOids map[string]bool
	snmpSearchMode   bool
	snmpSearchQuery  string
}

type (
	tickMsg     time.Time
	pcapTickMsg time.Time
	reloadMsg   struct {
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

// Init initializes the interactive TUI model.
func (m model) Init() tea.Cmd {
	return tea.Batch(
		tickCmd(),
		tea.EnterAltScreen,
	)
}

func tickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func pcapTickCmd(speed float64) tea.Cmd {
	// Calculate interval based on playback speed
	// Normal speed (1.0) = 100ms per packet
	interval := time.Duration(float64(100*time.Millisecond) / speed)

	return tea.Tick(interval, func(t time.Time) tea.Msg {
		return pcapTickMsg(t)
	})
}

func reloadCmd(fn func() (*config.Config, error)) tea.Cmd {
	if fn == nil {
		return nil
	}

	return func() tea.Msg {
		cfg, err := fn()

		return reloadMsg{cfg: cfg, err: err}
	}
}

// Update handles messages and updates the TUI model state.
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case reloadMsg:
		if msg.err != nil {
			m.statusMessage = errorStyle.Render(fmt.Sprintf("✗ Reload failed: %v", msg.err))
			m.statusIsError = true
		} else if msg.cfg != nil {
			// Store previous config for diff viewer
			if m.cfg != nil {
				m.previousConfig = m.cfg
			}

			m.cfg = msg.cfg
			m.selectedDeviceIdx = 0
			m.statusMessage = successStyle.Render(
				fmt.Sprintf("✓ Reloaded configuration (%d devices)", len(msg.cfg.Devices)),
			)
			m.statusIsError = false
			m.addDebugLog(fmt.Sprintf("Config reloaded: %d devices", len(msg.cfg.Devices)))
		}

		return m, nil

	case editorFinishedMsg:
		if msg.err != nil {
			m.statusMessage = errorStyle.Render(fmt.Sprintf("✗ Editor error: %v", msg.err))
			m.statusIsError = true
		} else {
			m.statusMessage = successStyle.Render("Editor closed - press [r] to reload config")
			m.statusIsError = false
		}

		return m, nil
	case tea.KeyMsg:
		// Handle search mode input
		if m.searchMode {
			return m.handleSearchInput(msg)
		}

		// Handle export panel input
		if m.showExport {
			return m.handleExportInput(msg)
		}

		// Handle alert config input
		if m.showAlertConfig {
			return m.handleAlertConfigInput(msg)
		}

		// Handle PCAP replay panel input
		if m.showPcapReplay {
			return m.handlePcapReplayInput(msg)
		}

		// Handle history viewer input
		if m.showHistory {
			return m.handleHistoryInput(msg)
		}

		// Handle SNMP walk browser input
		if m.showSnmpWalk {
			return m.handleSnmpWalkInput(msg)
		}

		// Handle device config panel input
		if m.showDeviceConfig {
			return m.handleDeviceConfigInput(msg)
		}

		// Handle value input mode
		if m.valueInputMode {
			return m.handleValueInput(msg)
		}

		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit

		case keyEsc:
			// Close validation modal if open
			if m.showValidation {
				m.showValidation = false

				return m, nil
			}
			// Close template browser or preview
			if m.showTemplatePreview {
				m.showTemplatePreview = false
				m.templatePreviewContent = ""

				return m, nil
			}

			if m.showTemplates {
				m.showTemplates = false

				return m, nil
			}
			// Close config diff viewer
			if m.showConfigDiff {
				m.showConfigDiff = false

				return m, nil
			}
			// Close search mode
			if m.searchMode || m.showSearch {
				m.searchMode = false
				m.showSearch = false
				m.searchQuery = ""
				m.searchResults = nil
				m.statusMessage = "Search cancelled"
				m.statusIsError = false

				return m, nil
			}
			// Close export panel
			if m.showExport {
				m.showExport = false

				return m, nil
			}
			// Close topology view
			if m.showTopology {
				m.showTopology = false

				return m, nil
			}
			// Close alert config
			if m.showAlertConfig {
				m.showAlertConfig = false
				m.statusMessage = successStyle.Render("Alert configuration saved")
				m.statusIsError = false

				return m, nil
			}

			return m, nil

		case "v":
			// Toggle validation view
			if m.showValidation {
				m.showValidation = false
			} else {
				// Run validation
				validator := config.NewValidator("current config")
				m.validationResults = validator.Validate(m.cfg)
				m.showValidation = true
				m.showHelp = false
				m.showLogs = false
				m.showStats = false
				m.showNeighbors = false
				m.showHexDump = false

				m.menuVisible = false
				switch {
				case m.validationResults.HasErrors():
					m.statusMessage = errorStyle.Render(fmt.Sprintf("Validation found %d error(s), %d warning(s)",
						len(m.validationResults.Errors), len(m.validationResults.Warnings)))
					m.statusIsError = true
				case m.validationResults.HasWarnings():
					m.statusMessage = fmt.Sprintf(
						"Validation passed with %d warning(s)",
						len(m.validationResults.Warnings),
					)
					m.statusIsError = false
				default:
					m.statusMessage = successStyle.Render("Validation passed - no errors or warnings")
					m.statusIsError = false
				}

				m.addDebugLog(fmt.Sprintf("Config validation: %d errors, %d warnings",
					len(m.validationResults.Errors), len(m.validationResults.Warnings)))
			}

			return m, nil

		case "t":
			// Toggle template browser
			if m.showTemplates {
				m.showTemplates = false
				m.showTemplatePreview = false
				m.templatePreviewContent = ""
			} else {
				m.templateList = templates.List()
				m.selectedTemplate = 0
				m.showTemplates = true
				m.showTemplatePreview = false
				m.showHelp = false
				m.showLogs = false
				m.showStats = false
				m.showNeighbors = false
				m.showHexDump = false
				m.showValidation = false
				m.showConfigDiff = false
				m.showTopology = false
				m.showSearch = false
				m.showExport = false
				m.menuVisible = false
				m.statusMessage = "Template Browser - use arrow keys to navigate, Enter to preview"
				m.statusIsError = false
			}

			return m, nil

		case "C":
			// Toggle config diff viewer
			if m.showConfigDiff {
				m.showConfigDiff = false
			} else {
				m.generateConfigDiff()
				m.showConfigDiff = true
				m.configDiffScrollY = 0
				m.closeAllOverlays()
				m.statusMessage = "Config Diff Viewer - showing changes since last reload"
				m.statusIsError = false
			}

			return m, nil

		case "e":
			// Quick config edit - open config in $EDITOR
			if m.configFilePath == "" {
				m.statusMessage = errorStyle.Render("No config file path available")
				m.statusIsError = true

				return m, nil
			}

			return m, m.openEditor()

		case "E":
			// Toggle export panel
			if m.showExport {
				m.showExport = false
			} else {
				m.showExport = true
				m.exportFormat = formatJSON // Default format
				m.closeAllOverlays()
				m.statusMessage = "Export Stats - Press [j] for JSON, [c] for CSV, [Enter] to save"
				m.statusIsError = false
			}

			return m, nil

		case "a":
			// Toggle alert configuration panel
			if m.showAlertConfig {
				m.showAlertConfig = false
				m.statusMessage = successStyle.Render("Alert configuration saved")
				m.statusIsError = false
			} else {
				// Initialize alert thresholds if not set
				if m.alertThresholds == nil {
					m.alertThresholds = map[string]int{
						"CPU":        80,
						"Memory":     85,
						"Disk":       90,
						"PacketLoss": 5,
						"Latency":    100,
					}
				}

				m.showAlertConfig = true
				m.selectedAlertType = 0
				m.closeAllOverlays()
				m.statusMessage = "Alert Config - [Up/Down] navigate, [Left/Right] adjust, [Enter] toggle, [Space] all"
				m.statusIsError = false
			}

			return m, nil

		case "T":
			// Toggle topology view
			if m.showTopology {
				m.showTopology = false
			} else {
				m.generateTopologyView()
				m.showTopology = true
				m.topologyScrollY = 0
				m.closeAllOverlays()
				m.statusMessage = "Topology View - ASCII network diagram"
				m.statusIsError = false
			}

			return m, nil

		case "P":
			// Toggle PCAP replay control panel
			if m.showPcapReplay {
				m.showPcapReplay = false
				m.pcapPlaying = false
			} else {
				// Copy captured packets to PCAP buffer for replay
				m.pcapPackets = make([]CapturedPacket, len(m.packetBuffer))
				copy(m.pcapPackets, m.packetBuffer)
				m.pcapPlaybackIndex = 0

				m.pcapPlaying = false
				if m.pcapPlaybackSpeed == 0 {
					m.pcapPlaybackSpeed = 1.0
				}

				m.showPcapReplay = true
				m.closeAllOverlays()
				m.statusMessage = "PCAP Replay - [Space] play/pause, [←→] step, [+/-] speed"
				m.statusIsError = false
			}

			return m, nil

		case "H":
			// Toggle history viewer
			if m.showHistory {
				m.showHistory = false
			} else {
				m.showHistory = true
				m.selectedHistoryIdx = 0
				m.historyScrollY = 0
				m.closeAllOverlays()
				m.statusMessage = "Run History - [↑↓] navigate, [Enter] details"
				m.statusIsError = false
			}

			return m, nil

		case "W":
			// Toggle SNMP walk browser
			if m.showSnmpWalk {
				m.showSnmpWalk = false
			} else {
				// Populate SNMP OID tree with sample data
				m.snmpOidTree = []SnmpOidEntry{
					{OID: ".1.3.6.1.2.1.1.1.0", Name: "sysDescr", Value: "Network Simulator", Type: "STRING"},
					{OID: ".1.3.6.1.2.1.1.2.0", Name: "sysObjectID", Value: ".1.3.6.1.4.1.99999", Type: "OID"},
					{
						OID:   ".1.3.6.1.2.1.1.3.0",
						Name:  "sysUpTime",
						Value: strconv.Itoa(int(m.uptime.Seconds() * 100)),
						Type:  "TIMETICKS",
					},
					{OID: ".1.3.6.1.2.1.1.4.0", Name: "sysContact", Value: "admin@niac.local", Type: "STRING"},
					{OID: ".1.3.6.1.2.1.1.5.0", Name: "sysName", Value: m.interfaceName, Type: "STRING"},
					{OID: ".1.3.6.1.2.1.1.6.0", Name: "sysLocation", Value: "Network Lab", Type: "STRING"},
				}
				m.selectedSnmpOid = 0
				m.snmpScrollY = 0
				m.showSnmpWalk = true
				m.closeAllOverlays()
				m.statusMessage = "SNMP Walk Browser - [↑↓] navigate, [Enter] expand"
				m.statusIsError = false
			}

			return m, nil

		case "F":
			// Toggle device configuration panel
			if m.showDeviceConfig {
				m.showDeviceConfig = false
			} else {
				m.showDeviceConfig = true
				m.deviceConfigTab = 0
				m.deviceConfigScrollY = 0
				m.closeAllOverlays()
				m.statusMessage = "Device Config - [Tab] switch tab, [↑↓] scroll"
				m.statusIsError = false
			}

			return m, nil

		case "/":
			// Enter search mode
			if m.searchMode {
				m.searchMode = false
				m.showSearch = false
				m.searchQuery = ""
				m.searchResults = nil
			} else {
				m.searchMode = true
				m.showSearch = true
				m.searchQuery = ""
				m.searchResults = nil
				m.selectedResult = 0
				m.searchCategory = searchCategoryAll
				m.closeAllOverlays()
				m.statusMessage = "Search Mode - type to filter devices/logs, [Esc] to exit"
				m.statusIsError = false
			}

			return m, nil

		case "i":
			m.menuVisible = !m.menuVisible
			if m.menuVisible {
				m.statusMessage = "Interactive menu opened - use arrow keys to navigate"
				m.statusIsError = false
			}

			return m, nil

		case "D":
			// Cycle through devices: 0 -> 1 -> 2 -> ... -> N-1 -> 0
			if len(m.cfg.Devices) > 0 {
				m.selectedDeviceIdx = (m.selectedDeviceIdx + 1) % len(m.cfg.Devices)
				device := m.cfg.Devices[m.selectedDeviceIdx]

				deviceIP := noIPPlaceholder
				if len(device.IPAddresses) > 0 {
					deviceIP = device.IPAddresses[0].String()
				}

				m.statusMessage = successStyle.Render(fmt.Sprintf("✓ Selected device: %s (%s)", device.Name, deviceIP))
				m.statusIsError = false
				m.addDebugLog(fmt.Sprintf("Selected device: %s (%s)", device.Name, deviceIP))
			} else {
				m.statusMessage = errorStyle.Render("✗ No devices configured")
				m.statusIsError = true
			}

			return m, nil

		case "d":
			// Cycle through debug levels: 0 -> 1 -> 2 -> 3 -> 0
			m.debugLevel = (m.debugLevel + 1) % 4
			debugLevelName := getDebugLevelName(m.debugLevel)
			m.statusMessage = successStyle.Render(fmt.Sprintf("✓ Debug level: %d (%s)", m.debugLevel, debugLevelName))
			m.statusIsError = false
			m.addDebugLog(fmt.Sprintf("Debug level changed to %d (%s)", m.debugLevel, debugLevelName))

			return m, nil
		case "r":
			if m.reloadFunc == nil {
				m.statusMessage = errorStyle.Render("✗ Reload not available in this mode")
				m.statusIsError = true

				return m, nil
			}

			m.statusMessage = "Reloading configuration..."
			m.statusIsError = false

			return m, reloadCmd(m.reloadFunc)

		case "h", "?":
			m.showHelp = !m.showHelp
			m.showLogs = false
			m.showStats = false
			m.showNeighbors = false
			m.showHexDump = false
			m.menuVisible = false

			return m, nil

		case "l":
			m.showLogs = !m.showLogs
			m.showHelp = false
			m.showStats = false
			m.showNeighbors = false
			m.menuVisible = false

			return m, nil

		case "s":
			m.showStats = !m.showStats
			m.showHelp = false
			m.showLogs = false
			m.showHexDump = false
			m.showNeighbors = false
			m.menuVisible = false

			return m, nil

		case "x":
			m.showHexDump = !m.showHexDump
			m.showHelp = false
			m.showLogs = false
			m.showStats = false
			m.showNeighbors = false

			m.menuVisible = false
			if m.showHexDump {
				m.hexDumpScrollY = 0
				m.statusMessage = "Hex dump viewer opened - use arrow keys to navigate, [n]/[p] for next/prev packet"
			}

			return m, nil

		case "N":
			m.toggleNeighborView()

			return m, nil

		case "n":
			if m.showHexDump && len(m.packetBuffer) > 0 {
				m.hexDumpPacketIndex = (m.hexDumpPacketIndex + 1) % len(m.packetBuffer)
				m.hexDumpScrollY = 0
				m.statusMessage = successStyle.Render(
					fmt.Sprintf("✓ Packet %d/%d", m.hexDumpPacketIndex+1, len(m.packetBuffer)),
				)

				return m, nil
			}

			m.toggleNeighborView()

			return m, nil

		case "p":
			if m.showHexDump && len(m.packetBuffer) > 0 {
				m.hexDumpPacketIndex--
				if m.hexDumpPacketIndex < 0 {
					m.hexDumpPacketIndex = len(m.packetBuffer) - 1
				}

				m.hexDumpScrollY = 0
				m.statusMessage = successStyle.Render(
					fmt.Sprintf("✓ Packet %d/%d", m.hexDumpPacketIndex+1, len(m.packetBuffer)),
				)
			}

			return m, nil

		case "c":
			if m.showTemplates && !m.showTemplatePreview {
				// Copy template path to clipboard (show path info)
				if m.selectedTemplate >= 0 && m.selectedTemplate < len(m.templateList) {
					templateName := m.templateList[m.selectedTemplate].Name
					// Show the template path (users can use `niac template use <name> <output>`)
					m.statusMessage = successStyle.Render(
						fmt.Sprintf(
							"Template: %s - Use: niac template use %s <output.yaml>",
							templateName,
							templateName,
						),
					)
					m.statusIsError = false
					m.addDebugLog("Template path shown: " + templateName)
				}

				return m, nil
			}

			m.stateManager.ClearAll()
			m.statusMessage = successStyle.Render("All error injections cleared")
			m.statusIsError = false
			m.errorsActive = 0
			m.addDebugLog("All error injections cleared")

			return m, nil

		case "up":
			switch {
			case m.showSearch && len(m.searchResults) > 0 && m.selectedResult > 0:
				m.selectedResult--
			case m.showConfigDiff && m.configDiffScrollY > 0:
				m.configDiffScrollY--
			case m.showTopology && m.topologyScrollY > 0:
				m.topologyScrollY--
			case m.showTemplates && !m.showTemplatePreview && m.selectedTemplate > 0:
				m.selectedTemplate--
			case m.menuVisible && m.selectedItem > 0:
				m.selectedItem--
			case m.showHexDump && m.hexDumpScrollY > 0:
				m.hexDumpScrollY--
			}

			return m, nil

		case keyDown:
			switch {
			case m.showSearch && len(m.searchResults) > 0 && m.selectedResult < len(m.searchResults)-1:
				m.selectedResult++
			case m.showConfigDiff:
				m.configDiffScrollY++
			case m.showTopology:
				m.topologyScrollY++
			case m.showTemplates && !m.showTemplatePreview && m.selectedTemplate < len(m.templateList)-1:
				m.selectedTemplate++
			case m.menuVisible && m.selectedItem < len(m.menuItems)-1:
				m.selectedItem++
			case m.showHexDump:
				m.hexDumpScrollY++
			}

			return m, nil

		case "pgup":
			switch {
			case m.showConfigDiff:
				m.configDiffScrollY -= 10
				if m.configDiffScrollY < 0 {
					m.configDiffScrollY = 0
				}
			case m.showTopology:
				m.topologyScrollY -= 10
				if m.topologyScrollY < 0 {
					m.topologyScrollY = 0
				}
			case m.showHexDump:
				m.hexDumpScrollY -= 10
				if m.hexDumpScrollY < 0 {
					m.hexDumpScrollY = 0
				}
			}

			return m, nil

		case "pgdown":
			switch {
			case m.showConfigDiff:
				m.configDiffScrollY += 10
			case m.showTopology:
				m.topologyScrollY += 10
			case m.showHexDump:
				m.hexDumpScrollY += 10
			}

			return m, nil

		case keyEnter:
			if m.showTemplates && !m.showTemplatePreview {
				// Show template preview
				if m.selectedTemplate >= 0 && m.selectedTemplate < len(m.templateList) {
					tmpl, err := templates.Get(m.templateList[m.selectedTemplate].Name)
					if err == nil {
						m.templatePreviewContent = tmpl.Content
						m.showTemplatePreview = true
						m.statusMessage = successStyle.Render(
							fmt.Sprintf("Previewing: %s - Press ESC to go back", tmpl.Name),
						)
						m.statusIsError = false
					} else {
						m.statusMessage = errorStyle.Render(fmt.Sprintf("Error loading template: %v", err))
						m.statusIsError = true
					}
				}
			} else if m.menuVisible {
				m.handleMenuSelection()
			}

			return m, nil

		// Quick access number keys (1-7) for error injection with default values
		case "1":
			if !m.menuVisible && !m.showHelp && !m.showLogs && !m.showStats {
				m.promptForValue(errors.ErrorTypeFCS, "Enter FCS error count (0-100): ")
			}

			return m, nil
		case "2":
			if !m.menuVisible && !m.showHelp && !m.showLogs && !m.showStats {
				m.promptForValue(errors.ErrorTypeDiscards, "Enter packet discard rate (0-100): ")
			}

			return m, nil
		case "3":
			if !m.menuVisible && !m.showHelp && !m.showLogs && !m.showStats {
				m.promptForValue(errors.ErrorTypeInterface, "Enter interface error count (0-100): ")
			}

			return m, nil
		case "4":
			if !m.menuVisible && !m.showHelp && !m.showLogs && !m.showStats {
				m.promptForValue(errors.ErrorTypeUtilization, "Enter utilization percentage (0-100): ")
			}

			return m, nil
		case "5":
			if !m.menuVisible && !m.showHelp && !m.showLogs && !m.showStats {
				m.promptForValue(errors.ErrorTypeCPU, "Enter CPU percentage (0-100): ")
			}

			return m, nil
		case "6":
			if !m.menuVisible && !m.showHelp && !m.showLogs && !m.showStats {
				m.promptForValue(errors.ErrorTypeMemory, "Enter memory percentage (0-100): ")
			}

			return m, nil
		case "7":
			if !m.menuVisible && !m.showHelp && !m.showLogs && !m.showStats {
				m.promptForValue(errors.ErrorTypeDisk, "Enter disk percentage (0-100): ")
			}

			return m, nil
		}

	case tickMsg:
		m.uptime = time.Since(m.startTime)
		m.errorsActive = len(m.stateManager.GetAllStates())
		m.refreshStats()

		return m, tickCmd()
	}

	return m, nil
}

func (m *model) refreshStats() {
	if m.stack == nil {
		return
	}

	stats := m.stack.GetStats()
	m.stackStats = stackStatsSnapshot{
		PacketsReceived: stats.PacketsReceived,
		PacketsSent:     stats.PacketsSent,
		ARPRequests:     stats.ARPRequests,
		ARPReplies:      stats.ARPReplies,
		ICMPRequests:    stats.ICMPRequests,
		ICMPReplies:     stats.ICMPReplies,
		DNSQueries:      stats.DNSQueries,
		DHCPRequests:    stats.DHCPRequests,
	}
	m.neighbors = m.stack.GetNeighbors()
}

func (m *model) toggleNeighborView() {
	m.showNeighbors = !m.showNeighbors
	m.showHelp = false
	m.showLogs = false
	m.showStats = false
	m.showHexDump = false

	m.menuVisible = false
	if m.showNeighbors {
		if len(m.neighbors) == 0 {
			m.statusMessage = "Neighbor table opened - waiting for discovery packets"
			m.statusIsError = false
		} else {
			m.statusMessage = successStyle.Render(fmt.Sprintf("✓ Showing %d learned neighbors", len(m.neighbors)))
			m.statusIsError = false
		}
	}
}

func (m *model) handleMenuSelection() {
	if m.selectedItem < 0 || m.selectedItem >= len(m.menuItems) {
		return
	}

	selection := m.menuItems[m.selectedItem]

	// Handle menu selections - now with custom value input
	switch {
	case strings.Contains(selection, "FCS Errors"):
		m.promptForValue(errors.ErrorTypeFCS, "Enter FCS error count (0-100): ")
	case strings.Contains(selection, "Packet Discards"):
		m.promptForValue(errors.ErrorTypeDiscards, "Enter packet discard rate (0-100): ")
	case strings.Contains(selection, "Interface Errors"):
		m.promptForValue(errors.ErrorTypeInterface, "Enter interface error count (0-100): ")
	case strings.Contains(selection, "High Utilization"):
		m.promptForValue(errors.ErrorTypeUtilization, "Enter utilization percentage (0-100): ")
	case strings.Contains(selection, "High CPU"):
		m.promptForValue(errors.ErrorTypeCPU, "Enter CPU percentage (0-100): ")
	case strings.Contains(selection, "High Memory"):
		m.promptForValue(errors.ErrorTypeMemory, "Enter memory percentage (0-100): ")
	case strings.Contains(selection, "High Disk"):
		m.promptForValue(errors.ErrorTypeDisk, "Enter disk percentage (0-100): ")
	case strings.Contains(selection, "Clear All"):
		m.stateManager.ClearAll()
		m.statusMessage = successStyle.Render("✓ All errors cleared")
		m.statusIsError = false
		m.errorsActive = 0
		m.addDebugLog("All error injections cleared")
	case strings.Contains(selection, "Exit"):
		m.menuVisible = false
	}
}

func (m *model) promptForValue(errorType errors.ErrorType, prompt string) {
	m.selectedErrorType = int(getErrorTypeIndex(errorType))
	m.valueInputPrompt = prompt
	m.valueInputBuffer = ""
	m.valueInputMode = true
	m.menuVisible = false
}

func (m model) handleValueInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case keyEnter:
		// Process the input
		var value int

		_, err := fmt.Sscanf(m.valueInputBuffer, "%d", &value)
		if err != nil || value < 0 || value > 100 {
			m.statusMessage = errorStyle.Render("✗ Invalid value. Must be between 0 and 100")
			m.statusIsError = true
		} else {
			errorType := getErrorTypeByIndex(m.selectedErrorType)
			m.injectError(errorType, value)
		}

		m.valueInputMode = false
		m.valueInputBuffer = ""

		return m, nil

	case keyEsc:
		// Cancel input
		m.valueInputMode = false
		m.valueInputBuffer = ""
		m.statusMessage = "Input cancelled"
		m.statusIsError = false

		return m, nil

	case "backspace":
		if len(m.valueInputBuffer) > 0 {
			m.valueInputBuffer = m.valueInputBuffer[:len(m.valueInputBuffer)-1]
		}

		return m, nil

	default:
		// Only accept digits
		if len(msg.String()) == 1 && msg.String()[0] >= '0' && msg.String()[0] <= '9' {
			if len(m.valueInputBuffer) < 3 { // Max 3 digits for 0-100
				m.valueInputBuffer += msg.String()
			}
		}

		return m, nil
	}
}

func getErrorTypeIndex(errorType errors.ErrorType) int {
	types := errors.AllErrorTypes()
	for i, t := range types {
		if t == errorType {
			return i
		}
	}

	return 0
}

func getErrorTypeByIndex(index int) errors.ErrorType {
	types := errors.AllErrorTypes()
	if index >= 0 && index < len(types) {
		return types[index]
	}

	return errors.ErrorTypeFCS
}

func (m *model) injectError(errorType errors.ErrorType, value int) {
	// Inject error on currently selected device
	if len(m.cfg.Devices) == 0 {
		m.statusMessage = errorStyle.Render("✗ No devices configured")
		m.statusIsError = true
		m.addDebugLog("ERROR: No devices configured for error injection")

		return
	}

	// Ensure selectedDeviceIdx is within bounds
	if m.selectedDeviceIdx < 0 || m.selectedDeviceIdx >= len(m.cfg.Devices) {
		m.selectedDeviceIdx = 0
	}

	device := m.cfg.Devices[m.selectedDeviceIdx]

	deviceIP := "unknown"
	if len(device.IPAddresses) > 0 {
		deviceIP = device.IPAddresses[0].String()
	}

	m.stateManager.SetError(deviceIP, "eth0", errorType, value)
	m.statusMessage = successStyle.Render(fmt.Sprintf("✓ Injected %s (%d%%) on %s", errorType, value, device.Name))
	m.statusIsError = false
	m.packetsInjected++
	m.errorsActive++
	m.addDebugLog(fmt.Sprintf("Injected %s (%d%%) on %s (%s)", errorType, value, device.Name, deviceIP))
}

// View renders the TUI display to the terminal.
func (m model) View() string {
	var s strings.Builder

	// Title
	s.WriteString(titleStyle.Render(fmt.Sprintf(" NIAC-Go Interactive Mode - %s ", m.interfaceName)))
	s.WriteString("\n\n")

	// Status bar with selected device
	selectedDeviceName := "None"
	if len(m.cfg.Devices) > 0 && m.selectedDeviceIdx >= 0 && m.selectedDeviceIdx < len(m.cfg.Devices) {
		selectedDeviceName = m.cfg.Devices[m.selectedDeviceIdx].Name
	}

	stats := fmt.Sprintf("Uptime: %s  |  Debug: %d (%s)  |  Selected Device: %s  |  Errors Active: %d  |  Injected: %d",
		formatDuration(m.uptime),
		m.debugLevel,
		getDebugLevelName(m.debugLevel),
		selectedDeviceName,
		m.errorsActive,
		m.packetsInjected,
	)
	s.WriteString(statsStyle.Render(stats))
	s.WriteString("\n\n")

	// Devices
	s.WriteString(deviceStyle.Render("📡 Simulated Devices:"))
	s.WriteString("\n")

	for i, device := range m.cfg.Devices {
		ip := noIPPlaceholder
		if len(device.IPAddresses) > 0 {
			ip = device.IPAddresses[0].String()
		}

		// Highlight selected device
		prefix := "  "
		suffix := ""

		if i == m.selectedDeviceIdx {
			prefix = selectedStyle.Render("→ ")
			suffix = selectedStyle.Render(" [SELECTED]")
		}

		s.WriteString(fmt.Sprintf("%s%d. %s (%s) - %s - %s%s\n",
			prefix,
			i+1,
			device.Name,
			device.Type,
			ip,
			device.MACAddress.String(),
			suffix,
		))
	}

	s.WriteString("\n")

	// Active errors
	activeStates := m.stateManager.GetAllStates()
	if len(activeStates) > 0 {
		s.WriteString(errorStyle.Render("⚠️  Active Error Injections:"))
		s.WriteString("\n")

		for _, state := range activeStates {
			s.WriteString(fmt.Sprintf("  • %s on %s:%s (%d%%)\n",
				state.ErrorType,
				state.DeviceIP,
				state.Interface,
				state.Value,
			))
		}

		s.WriteString("\n")
	}

	// Status message
	if m.statusMessage != "" {
		if m.statusIsError {
			s.WriteString(errorStyle.Render(m.statusMessage))
		} else {
			s.WriteString(m.statusMessage)
		}

		s.WriteString("\n\n")
	}

	// Value input prompt
	if m.valueInputMode {
		s.WriteString(m.renderValueInput())
		s.WriteString("\n")
	}

	// Menu
	if m.menuVisible && !m.valueInputMode {
		s.WriteString(m.renderMenu())
		s.WriteString("\n")
	}

	// Help overlay
	if m.showHelp {
		s.WriteString(m.renderHelp())
		s.WriteString("\n")
	}

	// Debug log viewer
	if m.showLogs {
		s.WriteString(m.renderLogs())
		s.WriteString("\n")
	}

	// Statistics viewer
	if m.showStats {
		s.WriteString(m.renderStatistics())
		s.WriteString("\n")
	}

	// Neighbor viewer
	if m.showNeighbors {
		s.WriteString(m.renderNeighbors())
		s.WriteString("\n")
	}

	// Hex dump viewer
	if m.showHexDump {
		s.WriteString(m.renderHexDump())
		s.WriteString("\n")
	}

	// Template browser
	if m.showTemplates {
		s.WriteString(m.renderTemplateBrowser())
		s.WriteString("\n")
	}

	// Validation overlay
	if m.showValidation {
		s.WriteString(m.renderValidation())
		s.WriteString("\n")
	}

	// Config diff overlay
	if m.showConfigDiff {
		s.WriteString(m.renderConfigDiff())
		s.WriteString("\n")
	}

	// Search overlay
	if m.showSearch {
		s.WriteString(m.renderSearch())
		s.WriteString("\n")
	}

	// Export overlay
	if m.showExport {
		s.WriteString(m.renderExport())
		s.WriteString("\n")
	}

	// Topology overlay
	if m.showTopology {
		s.WriteString(m.renderTopology())
		s.WriteString("\n")
	}

	// Alert config overlay
	if m.showAlertConfig {
		s.WriteString(m.renderAlertConfig())
		s.WriteString("\n")
	}

	// PCAP replay overlay
	if m.showPcapReplay {
		s.WriteString(m.renderPcapReplay())
		s.WriteString("\n")
	}

	// History viewer overlay
	if m.showHistory {
		s.WriteString(m.renderHistory())
		s.WriteString("\n")
	}

	// SNMP walk browser overlay
	if m.showSnmpWalk {
		s.WriteString(m.renderSnmpWalk())
		s.WriteString("\n")
	}

	// Device config overlay
	if m.showDeviceConfig {
		s.WriteString(m.renderDeviceConfig())
		s.WriteString("\n")
	}

	// Controls - first row
	s.WriteString("Controls: ")
	s.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("86")).Render("[i]"))
	s.WriteString(" Menu  ")
	s.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("86")).Render("[D]"))
	s.WriteString(" Device  ")
	s.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("86")).Render("[d]"))
	s.WriteString(" Debug  ")
	s.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("86")).Render("[h]"))
	s.WriteString(" Help  ")
	s.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("86")).Render("[l]"))
	s.WriteString(" Logs  ")
	s.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("86")).Render("[s]"))
	s.WriteString(" Stats  ")
	s.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("86")).Render("[N]"))
	s.WriteString(" Neighbors  ")
	s.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("86")).Render("[x]"))
	s.WriteString(" Hex\n")
	// Controls - second row
	s.WriteString("          ")
	s.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("86")).Render("[t]"))
	s.WriteString(" Templates  ")
	s.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("86")).Render("[v]"))
	s.WriteString(" Validate  ")
	s.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("86")).Render("[C]"))
	s.WriteString(" Diff  ")
	s.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("86")).Render("[e]"))
	s.WriteString(" Edit  ")
	s.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("86")).Render("[E]"))
	s.WriteString(" Export  ")
	s.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("86")).Render("[a]"))
	s.WriteString(" Alerts\n")
	// Controls - third row
	s.WriteString("          ")
	s.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("86")).Render("[T]"))
	s.WriteString(" Topology  ")
	s.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("86")).Render("[/]"))
	s.WriteString(" Search  ")
	s.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("86")).Render("[c]"))
	s.WriteString(" Clear  ")
	s.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Render("[q]"))
	s.WriteString(" Quit")

	return s.String()
}

func (m model) renderValueInput() string {
	var input strings.Builder

	input.WriteString("╔══════════════════════════════════════════════════════════════════╗\n")
	input.WriteString("║                    Error Value Input                            ║\n")
	input.WriteString("╠══════════════════════════════════════════════════════════════════╣\n")
	input.WriteString(fmt.Sprintf("║ %s%-60s ║\n", "", m.valueInputPrompt))
	input.WriteString("║                                                                  ║\n")

	// Show current input
	inputDisplay := m.valueInputBuffer
	if inputDisplay == "" {
		inputDisplay = "_"
	}

	input.WriteString(fmt.Sprintf("║ Value: %-56s ║\n", inputDisplay))
	input.WriteString("║                                                                  ║\n")
	input.WriteString("║ Press [Enter] to confirm, [Esc] to cancel                       ║\n")
	input.WriteString("╚══════════════════════════════════════════════════════════════════╝")

	return input.String()
}

func (m model) renderMenu() string {
	var menu strings.Builder

	// Get selected device info
	selectedDeviceInfo := "None"

	if len(m.cfg.Devices) > 0 && m.selectedDeviceIdx >= 0 && m.selectedDeviceIdx < len(m.cfg.Devices) {
		device := m.cfg.Devices[m.selectedDeviceIdx]

		deviceIP := noIPPlaceholder
		if len(device.IPAddresses) > 0 {
			deviceIP = device.IPAddresses[0].String()
		}

		selectedDeviceInfo = fmt.Sprintf("%s (%s)", device.Name, deviceIP)
	}

	menu.WriteString("╔══════════════════════════════════════════════════════════════════╗\n")
	menu.WriteString("║         Interactive Error Injection Menu                        ║\n")
	menu.WriteString("╠══════════════════════════════════════════════════════════════════╣\n")
	menu.WriteString(fmt.Sprintf("║ Target Device: %-49s ║\n", selectedDeviceInfo))
	menu.WriteString("║ (Press Shift+D to change device)                                ║\n")
	menu.WriteString("╠══════════════════════════════════════════════════════════════════╣\n")

	for i, item := range m.menuItems {
		if i == m.selectedItem {
			menu.WriteString("║ " + selectedStyle.Render("→ "+item))
		} else {
			menu.WriteString("║   " + item)
		}
		// Pad to align the right border (66 chars wide)
		padding := 64 - len(item) - 3
		menu.WriteString(strings.Repeat(" ", padding))
		menu.WriteString("║\n")
	}

	menu.WriteString("╚══════════════════════════════════════════════════════════════════╝")

	return menu.String()
}

func formatDuration(d time.Duration) string {
	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60
	seconds := int(d.Seconds()) % 60

	return fmt.Sprintf("%02d:%02d:%02d", hours, minutes, seconds)
}

func padPanelLine(text string) string {
	const width = 64
	if len(text) > width {
		if width > 3 {
			text = text[:width-3] + "..."
		} else {
			text = text[:width]
		}
	}

	return fmt.Sprintf("║ %-64s ║\n", text)
}

func fitColumn(text string, width int) string {
	if width <= 0 {
		return ""
	}

	if len(text) > width {
		if width > 3 {
			text = text[:width-3] + "..."
		} else {
			text = text[:width]
		}
	}

	return fmt.Sprintf("%-*s", width, text)
}

func formatRelativeTime(ts time.Time) string {
	if ts.IsZero() {
		return "never"
	}

	elapsed := max(time.Since(ts), 0)
	if elapsed < time.Second {
		return "now"
	}

	if elapsed < time.Minute {
		return fmt.Sprintf("%ds ago", int(elapsed.Seconds()))
	}

	if elapsed < time.Hour {
		return fmt.Sprintf("%dm%ds ago", int(elapsed.Minutes()), int(elapsed.Seconds())%60)
	}

	hours := int(elapsed.Hours())
	minutes := int(elapsed.Minutes()) % 60

	if hours >= 100 {
		return fmt.Sprintf("%dh", hours)
	}

	return fmt.Sprintf("%dh%dm", hours, minutes)
}

func getDebugLevelName(level int) string {
	switch level {
	case 0:
		return "QUIET"
	case 1:
		return "NORMAL"
	case 2:
		return "VERBOSE"
	case 3:
		return "DEBUG"
	default:
		return "UNKNOWN"
	}
}

func (m *model) addDebugLog(message string) {
	timestamp := time.Now().Format("15:04:05")
	logEntry := fmt.Sprintf("[%s] %s", timestamp, message)
	m.debugLogs = append(m.debugLogs, logEntry)

	// Keep only last 100 log entries
	if len(m.debugLogs) > 100 {
		m.debugLogs = m.debugLogs[len(m.debugLogs)-100:]
	}
}

func (m model) renderHelp() string {
	var help strings.Builder

	help.WriteString("╔══════════════════════════════════════════════════════════════════╗\n")
	help.WriteString("║                         NIAC-Go Help                             ║\n")
	help.WriteString("╠══════════════════════════════════════════════════════════════════╣\n")
	help.WriteString("║ Keyboard Shortcuts:                                              ║\n")
	help.WriteString("║                                                                  ║\n")
	help.WriteString("║  [i]     Toggle interactive error injection menu                ║\n")
	help.WriteString("║  [D]     Cycle through devices (Shift+D)                        ║\n")
	help.WriteString("║  [d]     Cycle debug level (QUIET→NORMAL→VERBOSE→DEBUG)         ║\n")
	help.WriteString("║  [h][?]  Toggle this help screen                                ║\n")
	help.WriteString("║  [l]     Toggle debug log viewer                                ║\n")
	help.WriteString("║  [s]     Toggle statistics viewer                               ║\n")
	help.WriteString("║  [N]/[n] Toggle neighbor discovery table                        ║\n")
	help.WriteString("║  [x]     Toggle packet hex dump viewer                          ║\n")
	help.WriteString("║  [t]     Toggle template browser                                ║\n")
	help.WriteString("║  [v]     Validate configuration                                 ║\n")
	help.WriteString("║  [r]     Reload configuration from disk                         ║\n")
	help.WriteString("║                                                                  ║\n")
	help.WriteString("║ New Features:                                                    ║\n")
	help.WriteString("║  [C]     Config diff viewer (compare before/after reload)       ║\n")
	help.WriteString("║  [e]     Quick edit config in $EDITOR                           ║\n")
	help.WriteString("║  [E]     Export statistics to JSON/CSV file                     ║\n")
	help.WriteString("║  [T]     Network topology view (ASCII diagram)                  ║\n")
	help.WriteString("║  [/]     Search mode (filter devices/logs by pattern)           ║\n")
	help.WriteString("║                                                                  ║\n")
	help.WriteString("║ Navigation:                                                      ║\n")
	help.WriteString("║  [n]/[p] Navigate packets (next/previous) in hex viewer         ║\n")
	help.WriteString("║  [↑][↓]  Scroll / Navigate menu items                           ║\n")
	help.WriteString("║  [PgUp]  Page up in scrollable views                            ║\n")
	help.WriteString("║  [PgDn]  Page down in scrollable views                          ║\n")
	help.WriteString("║  [c]     Clear all error injections                             ║\n")
	help.WriteString("║  [1-7]   Quick error injection (FCS/Disc/If/Util/CPU/Mem/Disk) ║\n")
	help.WriteString("║  [q]     Quit application                                       ║\n")
	help.WriteString("║                                                                  ║\n")
	help.WriteString("║ Search Mode ([/]):                                               ║\n")
	help.WriteString("║  - Type to search devices, logs, and errors                     ║\n")
	help.WriteString("║  - [Tab] to cycle category (all/devices/logs)                   ║\n")
	help.WriteString("║  - [Enter] to select, [Esc] to cancel                           ║\n")
	help.WriteString("║                                                                  ║\n")
	help.WriteString("║ Export Mode ([E]):                                               ║\n")
	help.WriteString("║  - [j] for JSON format, [c] for CSV format                      ║\n")
	help.WriteString("║  - [Enter] to save file, [Esc] to cancel                        ║\n")
	help.WriteString("║                                                                  ║\n")
	help.WriteString("║ Debug Levels:                                                    ║\n")
	help.WriteString("║  0 - QUIET    Only critical errors                              ║\n")
	help.WriteString("║  1 - NORMAL   Status messages (default)                         ║\n")
	help.WriteString("║  2 - VERBOSE  Protocol details                                   ║\n")
	help.WriteString("║  3 - DEBUG    Full packet details                               ║\n")
	help.WriteString("║                                                                  ║\n")
	help.WriteString("║ Error Injection Types:                                           ║\n")
	help.WriteString("║  • FCS Errors        - Frame Check Sequence errors (0-100)      ║\n")
	help.WriteString("║  • Packet Discards   - Dropped packets rate (0-100)             ║\n")
	help.WriteString("║  • Interface Errors  - General interface errors (0-100)         ║\n")
	help.WriteString("║  • High Utilization  - Link utilization percentage (0-100)      ║\n")
	help.WriteString("║  • High CPU          - CPU usage percentage (0-100)             ║\n")
	help.WriteString("║  • High Memory       - Memory usage percentage (0-100)          ║\n")
	help.WriteString("║  • High Disk         - Disk usage percentage (0-100)            ║\n")
	help.WriteString("╚══════════════════════════════════════════════════════════════════╝")

	return help.String()
}

func (m model) renderLogs() string {
	var logs strings.Builder

	logs.WriteString("╔══════════════════════════════════════════════════════════════════╗\n")
	logs.WriteString("║                      Debug Log Viewer                           ║\n")
	logs.WriteString("╠══════════════════════════════════════════════════════════════════╣\n")

	if len(m.debugLogs) == 0 {
		logs.WriteString("║ No debug logs yet                                                ║\n")
	} else {
		// Show last 10 logs
		start := 0
		if len(m.debugLogs) > 10 {
			start = len(m.debugLogs) - 10
		}

		for _, log := range m.debugLogs[start:] {
			// Pad to 66 characters for alignment
			padded := log
			if len(log) > 64 {
				padded = log[:64]
			} else {
				padded = log + strings.Repeat(" ", 64-len(log))
			}

			logs.WriteString(fmt.Sprintf("║ %s ║\n", padded))
		}
	}

	logs.WriteString("╚══════════════════════════════════════════════════════════════════╝")

	return logs.String()
}

func (m model) renderStatistics() string {
	var stats strings.Builder

	totalPackets := m.stackStats.PacketsReceived + m.stackStats.PacketsSent

	stats.WriteString("╔══════════════════════════════════════════════════════════════════╗\n")
	stats.WriteString("║                     Detailed Statistics                          ║\n")
	stats.WriteString("╠══════════════════════════════════════════════════════════════════╣\n")
	stats.WriteString(
		fmt.Sprintf("║ Uptime:              %s                                    ║\n", formatDuration(m.uptime)),
	)
	stats.WriteString(
		fmt.Sprintf(
			"║ Debug Level:         %d (%s)                              ║\n",
			m.debugLevel,
			getDebugLevelName(m.debugLevel),
		),
	)
	stats.WriteString(fmt.Sprintf("║ Interface:           %-40s ║\n", m.interfaceName))
	stats.WriteString("║                                                                  ║\n")
	stats.WriteString(fmt.Sprintf("║ Total Packets:       %-10d                                    ║\n", totalPackets))
	stats.WriteString(
		fmt.Sprintf(
			"║ RX / TX Packets:     %-10d / %-10d                       ║\n",
			m.stackStats.PacketsReceived,
			m.stackStats.PacketsSent,
		),
	)
	stats.WriteString(
		fmt.Sprintf(
			"║ ARP Req / Rep:       %-10d / %-10d                       ║\n",
			m.stackStats.ARPRequests,
			m.stackStats.ARPReplies,
		),
	)
	stats.WriteString(
		fmt.Sprintf(
			"║ ICMP Req / Rep:      %-10d / %-10d                       ║\n",
			m.stackStats.ICMPRequests,
			m.stackStats.ICMPReplies,
		),
	)
	stats.WriteString(
		fmt.Sprintf("║ DNS Queries:         %-10d                                    ║\n", m.stackStats.DNSQueries),
	)
	stats.WriteString(
		fmt.Sprintf("║ DHCP Requests:       %-10d                                    ║\n", m.stackStats.DHCPRequests),
	)
	stats.WriteString(
		fmt.Sprintf("║ Packets Injected:    %-10d                                    ║\n", m.packetsInjected),
	)
	stats.WriteString(
		fmt.Sprintf("║ Active Errors:       %-10d                                    ║\n", m.errorsActive),
	)
	stats.WriteString("║                                                                  ║\n")
	stats.WriteString(
		fmt.Sprintf("║ Devices Simulated:   %-10d                                    ║\n", len(m.cfg.Devices)),
	)

	snmpCount := 0

	for _, dev := range m.cfg.Devices {
		if dev.SNMPConfig.Community != "" || dev.SNMPConfig.WalkFile != "" {
			snmpCount++
		}
	}

	stats.WriteString(fmt.Sprintf("║ SNMP Devices:        %-10d                                    ║\n", snmpCount))
	stats.WriteString("║                                                                  ║\n")
	stats.WriteString(
		fmt.Sprintf("║ Start Time:          %s                                    ║\n", m.startTime.Format("15:04:05")),
	)
	stats.WriteString("╚══════════════════════════════════════════════════════════════════╝")

	return stats.String()
}

func (m model) renderNeighbors() string {
	var panel strings.Builder

	panel.WriteString("╔══════════════════════════════════════════════════════════════════╗\n")
	panel.WriteString("║                    Neighbor Discovery Table                      ║\n")
	panel.WriteString("╠══════════════════════════════════════════════════════════════════╣\n")

	if len(m.neighbors) == 0 {
		panel.WriteString(padPanelLine("No neighbors discovered yet"))
		panel.WriteString(padPanelLine("Advertise LLDP/CDP/EDP/FDP to populate this view"))
		panel.WriteString("╚══════════════════════════════════════════════════════════════════╝")

		return panel.String()
	}

	rows := make([]protocols.NeighborRecord, len(m.neighbors))
	copy(rows, m.neighbors)
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].LocalDevice != rows[j].LocalDevice {
			return rows[i].LocalDevice < rows[j].LocalDevice
		}

		if rows[i].Protocol != rows[j].Protocol {
			return rows[i].Protocol < rows[j].Protocol
		}

		return rows[i].RemoteDevice < rows[j].RemoteDevice
	})

	header := fmt.Sprintf("%s %s %s %s %s",
		fitColumn("Proto", 5),
		fitColumn("Local Device", 14),
		fitColumn("Remote (Port)", 18),
		fitColumn("Mgmt Address", 15),
		fitColumn("Seen", 8),
	)
	panel.WriteString(padPanelLine(header))
	panel.WriteString("╠══════════════════════════════════════════════════════════════════╣\n")

	for _, entry := range rows {
		remote := entry.RemoteDevice
		if entry.RemotePort != "" {
			if remote == "" {
				remote = entry.RemotePort
			} else {
				remote = fmt.Sprintf("%s/%s", remote, entry.RemotePort)
			}
		}

		mgmt := entry.ManagementAddress
		if mgmt == "" {
			mgmt = "-"
		}

		line := fmt.Sprintf("%s %s %s %s %s",
			fitColumn(strings.ToUpper(entry.Protocol), 5),
			fitColumn(entry.LocalDevice, 14),
			fitColumn(remote, 18),
			fitColumn(mgmt, 15),
			fitColumn(formatRelativeTime(entry.LastSeen), 8),
		)
		panel.WriteString(padPanelLine(line))
	}

	panel.WriteString("╠══════════════════════════════════════════════════════════════════╣\n")

	summary := fmt.Sprintf("Total neighbors: %d  •  TTL refresh every 30s", len(rows))
	panel.WriteString(padPanelLine(summary))
	panel.WriteString(padPanelLine("Press [N]/[n] to close this view"))
	panel.WriteString("╚══════════════════════════════════════════════════════════════════╝")

	return panel.String()
}

func (m model) renderHexDump() string {
	var dump strings.Builder

	dump.WriteString("╔══════════════════════════════════════════════════════════════════╗\n")
	dump.WriteString("║                    Packet Hex Dump Viewer                        ║\n")
	dump.WriteString("╠══════════════════════════════════════════════════════════════════╣\n")

	if len(m.packetBuffer) == 0 {
		dump.WriteString("║ No packets captured yet                                          ║\n")
		dump.WriteString("║ Packets will appear here as they are received                    ║\n")
		dump.WriteString("╚══════════════════════════════════════════════════════════════════╝")

		return dump.String()
	}

	// Get current packet
	if m.hexDumpPacketIndex >= len(m.packetBuffer) {
		m.hexDumpPacketIndex = len(m.packetBuffer) - 1
	}

	pkt := m.packetBuffer[m.hexDumpPacketIndex]

	// Packet metadata
	dump.WriteString(fmt.Sprintf("║ Packet: %d/%d                                                    ║\n",
		m.hexDumpPacketIndex+1, len(m.packetBuffer)))
	dump.WriteString(fmt.Sprintf("║ Time:     %-54s ║\n", pkt.Timestamp.Format("15:04:05.000000")))
	dump.WriteString(fmt.Sprintf("║ Protocol: %-54s ║\n", pkt.Protocol))
	dump.WriteString(fmt.Sprintf("║ Source:   %-54s ║\n", pkt.SrcAddr))
	dump.WriteString(fmt.Sprintf("║ Dest:     %-54s ║\n", pkt.DstAddr))
	dump.WriteString(fmt.Sprintf("║ Length:   %-54d ║\n", pkt.Length))
	dump.WriteString("╠══════════════════════════════════════════════════════════════════╣\n")
	dump.WriteString("║ Offset   Hex                                      ASCII          ║\n")
	dump.WriteString("╠══════════════════════════════════════════════════════════════════╣\n")

	// Calculate number of lines to display (16 bytes per line)
	maxLines := 15 // Display max 15 lines
	totalLines := (len(pkt.Data) + 15) / 16

	startLine := m.hexDumpScrollY
	if startLine >= totalLines {
		startLine = max(totalLines-1, 0)
	}

	endLine := min(startLine+maxLines, totalLines)

	// Render hex dump lines
	for line := startLine; line < endLine; line++ {
		offset := line * 16
		end := min(offset+16, len(pkt.Data))

		// Offset
		lineStr := fmt.Sprintf("║ %04x   ", offset)

		// Hex bytes
		var hexBuilder, asciiBuilder strings.Builder

		for i := offset; i < end; i++ {
			b := pkt.Data[i]
			fmt.Fprintf(&hexBuilder, "%02x ", b)

			if b >= 32 && b <= 126 {
				asciiBuilder.WriteByte(b)
			} else {
				asciiBuilder.WriteByte('.')
			}
		}

		// Pad hex to align ASCII column (48 chars for 16 bytes)
		hexStr := fmt.Sprintf("%-48s", hexBuilder.String())
		asciiStr := fmt.Sprintf("%-16s", asciiBuilder.String())

		lineStr += hexStr + " " + asciiStr + " ║\n"
		dump.WriteString(lineStr)
	}

	// Show scroll indicator if needed
	if totalLines > maxLines {
		dump.WriteString("╠══════════════════════════════════════════════════════════════════╣\n")
		dump.WriteString(fmt.Sprintf("║ Showing lines %d-%d of %d (use ↑/↓/PgUp/PgDn to scroll)        ║\n",
			startLine+1, endLine, totalLines))
	}

	dump.WriteString("╠══════════════════════════════════════════════════════════════════╣\n")
	dump.WriteString("║ Press [n] next packet  [p] previous packet  [x] close           ║\n")
	dump.WriteString("╚══════════════════════════════════════════════════════════════════╝")

	return dump.String()
}

func (m model) renderTemplateBrowser() string {
	var panel strings.Builder

	if m.showTemplatePreview {
		// Show template content preview
		panel.WriteString("╔══════════════════════════════════════════════════════════════════╗\n")
		panel.WriteString("║                    Template Preview                              ║\n")
		panel.WriteString("╠══════════════════════════════════════════════════════════════════╣\n")

		// Show template name
		if m.selectedTemplate >= 0 && m.selectedTemplate < len(m.templateList) {
			tmpl := m.templateList[m.selectedTemplate]
			panel.WriteString(padPanelLine("Name: " + tmpl.Name))
			panel.WriteString(padPanelLine("Description: " + tmpl.Description))
			panel.WriteString("╠══════════════════════════════════════════════════════════════════╣\n")
		}

		// Show first 15 lines of content
		lines := strings.Split(m.templatePreviewContent, "\n")

		maxPreviewLines := 15
		if len(lines) > maxPreviewLines {
			lines = lines[:maxPreviewLines]
		}

		for _, line := range lines {
			panel.WriteString(padPanelLine(line))
		}

		if len(strings.Split(m.templatePreviewContent, "\n")) > maxPreviewLines {
			panel.WriteString(padPanelLine("... (content truncated)"))
		}

		panel.WriteString("╠══════════════════════════════════════════════════════════════════╣\n")
		panel.WriteString(padPanelLine("[ESC] Back to list  [t] Close browser"))
		panel.WriteString("╚══════════════════════════════════════════════════════════════════╝")

		return panel.String()
	}

	// Template list view
	panel.WriteString("╔══════════════════════════════════════════════════════════════════╗\n")
	panel.WriteString("║                    Template Browser                              ║\n")
	panel.WriteString("╠══════════════════════════════════════════════════════════════════╣\n")

	if len(m.templateList) == 0 {
		panel.WriteString(padPanelLine("No templates available"))
		panel.WriteString("╚══════════════════════════════════════════════════════════════════╝")

		return panel.String()
	}

	for i, tmpl := range m.templateList {
		prefix := "  "
		if i == m.selectedTemplate {
			prefix = selectedStyle.Render("->")
		}

		// Format: name (description)
		line := fmt.Sprintf("%s %-18s (%s)", prefix, tmpl.Name, tmpl.Description)
		if len(line) > 64 {
			line = line[:61] + "..."
		}

		panel.WriteString(padPanelLine(line))
	}

	panel.WriteString("╠══════════════════════════════════════════════════════════════════╣\n")
	panel.WriteString(padPanelLine("[up/down] Navigate  [Enter] Preview  [c] Copy  [ESC] Close"))
	panel.WriteString("╚══════════════════════════════════════════════════════════════════╝")

	return panel.String()
}

func (m model) renderValidation() string {
	var panel strings.Builder

	panel.WriteString("╔══════════════════════════════════════════════════════════════════╗\n")
	panel.WriteString("║                   Configuration Validation                       ║\n")
	panel.WriteString("╠══════════════════════════════════════════════════════════════════╣\n")

	if m.validationResults == nil {
		panel.WriteString(padPanelLine("No validation results available"))
		panel.WriteString("╚══════════════════════════════════════════════════════════════════╝")

		return panel.String()
	}

	// Summary line
	errorCount := len(m.validationResults.Errors)
	warningCount := len(m.validationResults.Warnings)

	if errorCount == 0 && warningCount == 0 {
		successLine := validationSuccessStyle.Render("Configuration is valid - no errors or warnings")
		panel.WriteString(padPanelLine(successLine))
		panel.WriteString(padPanelLine(""))
		panel.WriteString(padPanelLine(fmt.Sprintf("Devices configured: %d", len(m.cfg.Devices))))
	} else {
		// Show summary
		summaryLine := fmt.Sprintf("Found: %s, %s",
			validationErrorStyle.Render(fmt.Sprintf("%d error(s)", errorCount)),
			validationWarningStyle.Render(fmt.Sprintf("%d warning(s)", warningCount)))
		panel.WriteString(padPanelLine(summaryLine))
		panel.WriteString("╠══════════════════════════════════════════════════════════════════╣\n")

		// Show errors first (red)
		if errorCount > 0 {
			panel.WriteString(padPanelLine(validationErrorStyle.Render("ERRORS:")))

			for i, err := range m.validationResults.Errors {
				if i >= 8 { // Limit display to prevent overflow
					remaining := errorCount - i
					panel.WriteString(
						padPanelLine(validationErrorStyle.Render(fmt.Sprintf("  ... and %d more error(s)", remaining))),
					)

					break
				}

				errLine := fmt.Sprintf("  [%s] %s", err.Field, err.Message)
				if len(errLine) > 62 {
					errLine = errLine[:59] + "..."
				}

				panel.WriteString(padPanelLine(validationErrorStyle.Render(errLine)))
			}

			panel.WriteString(padPanelLine(""))
		}

		// Show warnings (yellow)
		if warningCount > 0 {
			panel.WriteString(padPanelLine(validationWarningStyle.Render("WARNINGS:")))

			for i, warn := range m.validationResults.Warnings {
				if i >= 8 { // Limit display to prevent overflow
					remaining := warningCount - i
					panel.WriteString(
						padPanelLine(
							validationWarningStyle.Render(fmt.Sprintf("  ... and %d more warning(s)", remaining)),
						),
					)

					break
				}

				warnLine := fmt.Sprintf("  [%s] %s", warn.Field, warn.Message)
				if len(warnLine) > 62 {
					warnLine = warnLine[:59] + "..."
				}

				panel.WriteString(padPanelLine(validationWarningStyle.Render(warnLine)))
			}
		}
	}

	panel.WriteString("╠══════════════════════════════════════════════════════════════════╣\n")
	panel.WriteString(padPanelLine(validationInfoStyle.Render("Press [v] or [Esc] to close this view")))
	panel.WriteString("╚══════════════════════════════════════════════════════════════════╝")

	return panel.String()
}

// AddPacket adds a packet to the capture buffer.
func (m *model) AddPacket(protocol, srcAddr, dstAddr string, data []byte) {
	pkt := CapturedPacket{
		Timestamp: time.Now(),
		Protocol:  protocol,
		SrcAddr:   srcAddr,
		DstAddr:   dstAddr,
		Length:    len(data),
		Data:      make([]byte, len(data)),
	}
	copy(pkt.Data, data)

	m.packetBuffer = append(m.packetBuffer, pkt)

	// Keep only last maxPacketBuffer packets
	if len(m.packetBuffer) > maxPacketBuffer {
		m.packetBuffer = m.packetBuffer[len(m.packetBuffer)-maxPacketBuffer:]
		// Adjust index if needed
		if m.hexDumpPacketIndex >= len(m.packetBuffer) {
			m.hexDumpPacketIndex = len(m.packetBuffer) - 1
		}
	}
}

// Run starts the interactive mode.
func Run(
	interfaceName string,
	cfg *config.Config,
	debugConfig *logging.DebugConfig,
	stack *protocols.Stack,
	startTime time.Time,
	reloadFunc func() (*config.Config, error),
) error {
	return RunWithConfigPath(interfaceName, cfg, debugConfig, stack, startTime, reloadFunc, "")
}

// RunWithConfigPath starts the interactive mode with a config file path for editing.
func RunWithConfigPath(
	interfaceName string,
	cfg *config.Config,
	debugConfig *logging.DebugConfig,
	stack *protocols.Stack,
	startTime time.Time,
	reloadFunc func() (*config.Config, error),
	configFilePath string,
) error {
	if debugConfig == nil {
		debugConfig = logging.NewDebugConfig(1)
	}

	debugLevel := debugConfig.GetGlobal()

	if startTime.IsZero() {
		startTime = time.Now()
	}

	// Initialize state manager
	stateManager := errors.NewStateManager()

	// Create menu items
	menuItems := []string{
		"1. Inject FCS Errors (custom value)",
		"2. Inject Packet Discards (custom value)",
		"3. Inject Interface Errors (custom value)",
		"4. Inject High Utilization (custom value)",
		"5. Inject High CPU (custom value)",
		"6. Inject High Memory (custom value)",
		"7. Inject High Disk (custom value)",
		"8. Clear All Errors",
		"9. Exit Menu",
	}

	// Create model
	m := model{
		cfg:            cfg,
		stateManager:   stateManager,
		interfaceName:  interfaceName,
		debugLevel:     debugLevel,
		stack:          stack,
		reloadFunc:     reloadFunc,
		menuItems:      menuItems,
		startTime:      startTime,
		configFilePath: configFilePath,
		exportFormat:   formatJSON,
		searchCategory: searchCategoryAll,
		statusMessage:  "Press 'i' for menu, 'r' to reload config, 'h' for help",
		debugLogs:      make([]string, 0, 100),
	}

	if stack != nil {
		m.refreshStats()
	}

	// Add initial log entry
	m.addDebugLog("Started NIAC-Go interactive mode on " + interfaceName)
	m.addDebugLog(fmt.Sprintf("Debug level: %d (%s)", debugLevel, getDebugLevelName(debugLevel)))
	m.addDebugLog(fmt.Sprintf("Simulating %d device(s)", len(cfg.Devices)))

	// Start TUI
	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("error running program: %w", err)
	}

	return nil
}

// closeAllOverlays closes all overlay panels.
func (m *model) closeAllOverlays() {
	m.showHelp = false
	m.showLogs = false
	m.showStats = false
	m.showNeighbors = false
	m.showHexDump = false
	m.showValidation = false
	m.showTemplates = false
	m.showTemplatePreview = false
	m.menuVisible = false
	m.valueInputMode = false
}

// handleSearchInput handles keyboard input during search mode.
func (m model) handleSearchInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case keyEsc:
		m.searchMode = false
		m.showSearch = false
		m.searchQuery = ""
		m.searchResults = nil
		m.statusMessage = "Search cancelled"
		m.statusIsError = false

		return m, nil

	case keyEnter:
		// Select current search result
		if len(m.searchResults) > 0 && m.selectedResult >= 0 && m.selectedResult < len(m.searchResults) {
			result := m.searchResults[m.selectedResult]
			m.searchMode = false
			m.showSearch = false

			// Navigate to the selected item
			switch result.Category {
			case "device":
				if result.Index >= 0 && result.Index < len(m.cfg.Devices) {
					m.selectedDeviceIdx = result.Index
					m.statusMessage = successStyle.Render("Selected device: " + result.Title)
				}
			case "log":
				m.showLogs = true
				m.statusMessage = successStyle.Render("Opened log viewer")
			}

			m.statusIsError = false
		}

		return m, nil

	case "up":
		if m.selectedResult > 0 {
			m.selectedResult--
		}

		return m, nil

	case keyDown:
		if m.selectedResult < len(m.searchResults)-1 {
			m.selectedResult++
		}

		return m, nil

	case "tab":
		// Cycle search category
		switch m.searchCategory {
		case searchCategoryAll:
			m.searchCategory = searchCategoryDevices
		case searchCategoryDevices:
			m.searchCategory = searchCategoryLogs
		case searchCategoryLogs:
			m.searchCategory = searchCategoryAll
		}

		m.performSearch()

		return m, nil

	case "backspace":
		if len(m.searchQuery) > 0 {
			m.searchQuery = m.searchQuery[:len(m.searchQuery)-1]
			m.performSearch()
		}

		return m, nil

	default:
		// Only accept printable characters
		if len(msg.String()) == 1 {
			char := msg.String()[0]
			if char >= 32 && char <= 126 {
				m.searchQuery += msg.String()
				m.performSearch()
			}
		}

		return m, nil
	}
}

// performSearch executes the search and populates results.
func (m *model) performSearch() {
	m.searchResults = nil
	m.selectedResult = 0

	if m.searchQuery == "" {
		return
	}

	query := strings.ToLower(m.searchQuery)

	// Search devices
	if m.searchCategory == searchCategoryAll || m.searchCategory == searchCategoryDevices {
		for i, device := range m.cfg.Devices {
			if strings.Contains(strings.ToLower(device.Name), query) ||
				strings.Contains(strings.ToLower(device.Type), query) {
				ip := noIPPlaceholder
				if len(device.IPAddresses) > 0 {
					ip = device.IPAddresses[0].String()
				}

				m.searchResults = append(m.searchResults, searchResult{
					Category: "device",
					Title:    device.Name,
					Detail:   fmt.Sprintf("%s - %s - %s", device.Type, ip, device.MACAddress.String()),
					Index:    i,
				})
			}
			// Also search by IP
			for _, ip := range device.IPAddresses {
				if strings.Contains(ip.String(), query) {
					m.searchResults = append(m.searchResults, searchResult{
						Category: "device",
						Title:    device.Name,
						Detail:   fmt.Sprintf("%s - %s", device.Type, ip.String()),
						Index:    i,
					})

					break
				}
			}
		}
	}

	// Search logs
	if m.searchCategory == searchCategoryAll || m.searchCategory == searchCategoryLogs {
		for i, log := range m.debugLogs {
			if strings.Contains(strings.ToLower(log), query) {
				m.searchResults = append(m.searchResults, searchResult{
					Category: "log",
					Title:    log,
					Detail:   fmt.Sprintf("Log entry #%d", i+1),
					Index:    i,
				})
			}
		}
	}

	// Search active errors
	if m.searchCategory == searchCategoryAll {
		for _, state := range m.stateManager.GetAllStates() {
			if strings.Contains(strings.ToLower(state.DeviceIP), query) ||
				strings.Contains(strings.ToLower(string(state.ErrorType)), query) {
				m.searchResults = append(m.searchResults, searchResult{
					Category: "error",
					Title:    fmt.Sprintf("%s on %s", state.ErrorType, state.DeviceIP),
					Detail:   fmt.Sprintf("Interface: %s, Value: %d%%", state.Interface, state.Value),
					Index:    -1,
				})
			}
		}
	}
}

// handleExportInput handles keyboard input during export panel.
func (m model) handleExportInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case keyEsc:
		m.showExport = false
		m.statusMessage = "Export cancelled"
		m.statusIsError = false

		return m, nil

	case "j":
		m.exportFormat = formatJSON
		m.statusMessage = "Export format: JSON"
		m.statusIsError = false

		return m, nil

	case "c":
		m.exportFormat = "csv"
		m.statusMessage = "Export format: CSV"
		m.statusIsError = false

		return m, nil

	case keyEnter:
		// Perform export
		err := m.exportStats()
		if err != nil {
			m.statusMessage = errorStyle.Render(fmt.Sprintf("Export failed: %v", err))
			m.statusIsError = true
		} else {
			m.statusMessage = successStyle.Render("Exported to: " + m.lastExportPath)
			m.statusIsError = false
			m.showExport = false
		}

		return m, nil
	}

	return m, nil
}

// exportStats exports current statistics to a file.
func (m *model) exportStats() error {
	timestamp := time.Now().Format("20060102-150405")

	var (
		filename string
		err      error
	)

	if m.exportFormat == formatJSON {
		filename = fmt.Sprintf("niac-stats-%s.json", timestamp)
		err = m.exportStatsJSON(filename)
	} else {
		filename = fmt.Sprintf("niac-stats-%s.csv", timestamp)
		err = m.exportStatsCSV(filename)
	}

	if err != nil {
		return err
	}

	m.lastExportPath = filename
	m.lastExportTime = time.Now()
	m.addDebugLog("Exported stats to " + filename)

	return nil
}

// exportStatsJSON exports statistics to JSON format.
func (m *model) exportStatsJSON(filename string) error {
	stats := map[string]any{
		"timestamp":        time.Now().Format(time.RFC3339),
		"interface":        m.interfaceName,
		"uptime_seconds":   m.uptime.Seconds(),
		"debug_level":      m.debugLevel,
		"device_count":     len(m.cfg.Devices),
		"errors_active":    m.errorsActive,
		"packets_injected": m.packetsInjected,
		"stack_stats": map[string]uint64{
			"packets_received": m.stackStats.PacketsReceived,
			"packets_sent":     m.stackStats.PacketsSent,
			"arp_requests":     m.stackStats.ARPRequests,
			"arp_replies":      m.stackStats.ARPReplies,
			"icmp_requests":    m.stackStats.ICMPRequests,
			"icmp_replies":     m.stackStats.ICMPReplies,
			"dns_queries":      m.stackStats.DNSQueries,
			"dhcp_requests":    m.stackStats.DHCPRequests,
		},
		"devices": m.getDevicesSummary(),
	}

	data, err := json.MarshalIndent(stats, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal stats JSON: %w", err)
	}

	if err := os.WriteFile(filename, data, 0o600); err != nil {
		return fmt.Errorf("failed to write stats file: %w", err)
	}
	return nil
}

// exportStatsCSV exports statistics to CSV format.
func (m *model) exportStatsCSV(filename string) error {
	// SECURITY FIX #163: Create file with restricted permissions (owner-only)
	file, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600) // #nosec G304 -- user-initiated
	if err != nil {
		return fmt.Errorf("failed to create CSV file: %w", err)
	}

	defer func() { _ = file.Close() }()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header
	_ = writer.Write([]string{"Metric", "Value", "Category"}) // CSV write errors handled by writer.Error()

	// General stats
	_ = writer.Write(
		[]string{"Timestamp", time.Now().Format(time.RFC3339), "General"},
	) // CSV write errors handled by writer.Error()
	_ = writer.Write(
		[]string{"Interface", m.interfaceName, "General"},
	) // CSV write errors handled by writer.Error()
	_ = writer.Write(
		[]string{"Uptime (seconds)", fmt.Sprintf("%.0f", m.uptime.Seconds()), "General"},
	) // CSV write errors handled by writer.Error()
	_ = writer.Write(
		[]string{"Debug Level", strconv.Itoa(m.debugLevel), "General"},
	) // CSV write errors handled by writer.Error()
	_ = writer.Write(
		[]string{"Device Count", strconv.Itoa(len(m.cfg.Devices)), "General"},
	) // CSV write errors handled by writer.Error()
	_ = writer.Write(
		[]string{"Errors Active", strconv.Itoa(m.errorsActive), "General"},
	) // CSV write errors handled by writer.Error()
	_ = writer.Write(
		[]string{"Packets Injected", strconv.Itoa(m.packetsInjected), "General"},
	) // CSV write errors handled by writer.Error()

	// Stack stats
	_ = writer.Write(
		[]string{"Packets Received", strconv.FormatUint(m.stackStats.PacketsReceived, 10), "Network"},
	) // CSV write errors handled by writer.Error()
	_ = writer.Write(
		[]string{"Packets Sent", strconv.FormatUint(m.stackStats.PacketsSent, 10), "Network"},
	) // CSV write errors handled by writer.Error()
	_ = writer.Write(
		[]string{"ARP Requests", strconv.FormatUint(m.stackStats.ARPRequests, 10), "Protocol"},
	) // CSV write errors handled by writer.Error()
	_ = writer.Write(
		[]string{"ARP Replies", strconv.FormatUint(m.stackStats.ARPReplies, 10), "Protocol"},
	) // CSV write errors handled by writer.Error()
	_ = writer.Write(
		[]string{"ICMP Requests", strconv.FormatUint(m.stackStats.ICMPRequests, 10), "Protocol"},
	) // CSV write errors handled by writer.Error()
	_ = writer.Write(
		[]string{"ICMP Replies", strconv.FormatUint(m.stackStats.ICMPReplies, 10), "Protocol"},
	) // CSV write errors handled by writer.Error()
	_ = writer.Write(
		[]string{"DNS Queries", strconv.FormatUint(m.stackStats.DNSQueries, 10), "Protocol"},
	) // CSV write errors handled by writer.Error()
	_ = writer.Write(
		[]string{"DHCP Requests", strconv.FormatUint(m.stackStats.DHCPRequests, 10), "Protocol"},
	) // CSV write errors handled by writer.Error()

	// Devices
	for _, device := range m.cfg.Devices {
		ip := noIPPlaceholder
		if len(device.IPAddresses) > 0 {
			ip = device.IPAddresses[0].String()
		}

		_ = writer.Write(
			[]string{device.Name, fmt.Sprintf("%s,%s,%s", device.Type, ip, device.MACAddress.String()), "Device"},
		) // CSV write errors handled by writer.Error()
	}

	return nil
}

// getDevicesSummary returns a summary of devices for export.
func (m *model) getDevicesSummary() []map[string]string {
	devices := make([]map[string]string, 0, len(m.cfg.Devices))
	for _, device := range m.cfg.Devices {
		ip := noIPPlaceholder
		if len(device.IPAddresses) > 0 {
			ip = device.IPAddresses[0].String()
		}

		devices = append(devices, map[string]string{
			"name": device.Name,
			"type": device.Type,
			"ip":   ip,
			"mac":  device.MACAddress.String(),
		})
	}

	return devices
}

// openEditor opens the config file in the user's preferred editor.
func (m *model) openEditor() tea.Cmd {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = os.Getenv("VISUAL")
	}

	if editor == "" {
		editor = "vi" // Default fallback
	}

	c := exec.CommandContext(context.Background(), editor, m.configFilePath) // #nosec G204 -- user-specified editor

	return tea.ExecProcess(c, func(err error) tea.Msg {
		return editorFinishedMsg{err: err}
	})
}

// generateConfigDiff generates a diff between current and previous config.
func (m *model) generateConfigDiff() {
	m.configDiffContent = nil

	if m.previousConfig == nil {
		m.configDiffContent = append(m.configDiffContent, "No previous configuration to compare.")
		m.configDiffContent = append(m.configDiffContent, "Reload config with [r] to create a comparison baseline.")

		return
	}

	// Compare device counts
	prevCount := len(m.previousConfig.Devices)
	currCount := len(m.cfg.Devices)

	m.configDiffContent = append(m.configDiffContent, diffHeaderStyle.Render("=== Configuration Diff ==="))
	m.configDiffContent = append(m.configDiffContent, "")

	if prevCount != currCount {
		if currCount > prevCount {
			m.configDiffContent = append(
				m.configDiffContent,
				diffAddedStyle.Render(
					fmt.Sprintf("+ Device count: %d -> %d (+%d)", prevCount, currCount, currCount-prevCount),
				),
			)
		} else {
			m.configDiffContent = append(
				m.configDiffContent,
				diffRemovedStyle.Render(
					fmt.Sprintf("- Device count: %d -> %d (-%d)", prevCount, currCount, prevCount-currCount),
				),
			)
		}
	} else {
		m.configDiffContent = append(m.configDiffContent, fmt.Sprintf("  Device count: %d (unchanged)", currCount))
	}

	m.configDiffContent = append(m.configDiffContent, "")

	// Build device maps for comparison
	prevDevices := make(map[string]*config.Device)
	for i := range m.previousConfig.Devices {
		prevDevices[m.previousConfig.Devices[i].Name] = &m.previousConfig.Devices[i]
	}

	currDevices := make(map[string]*config.Device)
	for i := range m.cfg.Devices {
		currDevices[m.cfg.Devices[i].Name] = &m.cfg.Devices[i]
	}

	// Find added devices
	m.configDiffContent = append(m.configDiffContent, diffHeaderStyle.Render("--- Devices ---"))

	for name, device := range currDevices {
		if _, exists := prevDevices[name]; !exists {
			ip := noIPPlaceholder
			if len(device.IPAddresses) > 0 {
				ip = device.IPAddresses[0].String()
			}

			m.configDiffContent = append(
				m.configDiffContent,
				diffAddedStyle.Render(fmt.Sprintf("+ %s (%s, %s)", name, device.Type, ip)),
			)
		}
	}

	// Find removed devices
	for name, device := range prevDevices {
		if _, exists := currDevices[name]; !exists {
			ip := noIPPlaceholder
			if len(device.IPAddresses) > 0 {
				ip = device.IPAddresses[0].String()
			}

			m.configDiffContent = append(
				m.configDiffContent,
				diffRemovedStyle.Render(fmt.Sprintf("- %s (%s, %s)", name, device.Type, ip)),
			)
		}
	}

	// Find modified devices
	for name, currDev := range currDevices {
		if prevDev, exists := prevDevices[name]; exists {
			changes := m.compareDevices(prevDev, currDev)
			for _, change := range changes {
				m.configDiffContent = append(m.configDiffContent, fmt.Sprintf("  %s: %s", name, change))
			}
		}
	}

	if len(m.configDiffContent) <= 4 {
		m.configDiffContent = append(m.configDiffContent, "  No device changes detected")
	}
}

// compareDevices compares two devices and returns a list of changes.
func (m *model) compareDevices(prev, curr *config.Device) []string {
	var changes []string

	// Compare type
	if prev.Type != curr.Type {
		changes = append(changes, fmt.Sprintf("type: %s -> %s", prev.Type, curr.Type))
	}

	// Compare MAC
	if prev.MACAddress.String() != curr.MACAddress.String() {
		changes = append(changes, fmt.Sprintf("MAC: %s -> %s", prev.MACAddress, curr.MACAddress))
	}

	// Compare IPs
	prevIPs := make(map[string]bool)
	for _, ip := range prev.IPAddresses {
		prevIPs[ip.String()] = true
	}

	currIPs := make(map[string]bool)
	for _, ip := range curr.IPAddresses {
		currIPs[ip.String()] = true
	}

	for ip := range currIPs {
		if !prevIPs[ip] {
			changes = append(changes, diffAddedStyle.Render("+ IP: "+ip))
		}
	}

	for ip := range prevIPs {
		if !currIPs[ip] {
			changes = append(changes, diffRemovedStyle.Render("- IP: "+ip))
		}
	}

	return changes
}

// generateTopologyView generates an ASCII network topology diagram.
func (m *model) generateTopologyView() {
	var sb strings.Builder

	// Build topology using the api package
	topology := api.BuildTopology(m.cfg)

	sb.WriteString(diffHeaderStyle.Render("=== Network Topology ==="))
	sb.WriteString("\n\n")

	// Display nodes
	sb.WriteString(topologyNodeStyle.Render("Nodes:"))
	sb.WriteString("\n")

	// Group nodes by type
	nodesByType := make(map[string][]api.TopologyNode)
	for _, node := range topology.Nodes {
		nodesByType[node.Type] = append(nodesByType[node.Type], node)
	}

	for nodeType, nodes := range nodesByType {
		sb.WriteString(fmt.Sprintf("  [%s]\n", nodeType))

		for _, node := range nodes {
			sb.WriteString(fmt.Sprintf("    +-- %s\n", node.Name))
		}
	}

	sb.WriteString("\n")
	sb.WriteString(topologyLinkStyle.Render("Links:"))
	sb.WriteString("\n")

	if len(topology.Links) == 0 {
		sb.WriteString("  (no trunk connections defined)\n")
	} else {
		for _, link := range topology.Links {
			linkInfo := fmt.Sprintf("  %s [%s] <---> [%s] %s",
				link.Source,
				link.SourceInterface,
				link.TargetInterface,
				link.Target)

			if link.LinkType != "" {
				linkInfo += fmt.Sprintf(" (%s)", link.LinkType)
			}

			if len(link.VLANs) > 0 {
				linkInfo += fmt.Sprintf(" VLANs: %v", link.VLANs)
			}

			sb.WriteString(linkInfo + "\n")
		}
	}

	sb.WriteString("\n")
	sb.WriteString(diffHeaderStyle.Render("=== ASCII Diagram ==="))
	sb.WriteString("\n\n")

	// Generate simple ASCII diagram
	sb.WriteString(m.generateASCIIDiagram(topology))

	m.topologyContent = sb.String()
}

// generateASCIIDiagram creates a simple ASCII network diagram.
func (m *model) generateASCIIDiagram(topology api.Topology) string {
	var sb strings.Builder

	if len(topology.Nodes) == 0 {
		return "  (no devices configured)\n"
	}

	// Simple layout: core devices in center, edges around
	routers := make([]api.TopologyNode, 0)
	switches := make([]api.TopologyNode, 0)
	others := make([]api.TopologyNode, 0)

	for _, node := range topology.Nodes {
		switch node.Type {
		case "router":
			routers = append(routers, node)
		case "switch":
			switches = append(switches, node)
		default:
			others = append(others, node)
		}
	}

	// Draw routers at top
	if len(routers) > 0 {
		sb.WriteString("                        ROUTERS\n")
		sb.WriteString("  ")

		for i, r := range routers {
			if i > 0 {
				sb.WriteString("    ")
			}

			sb.WriteString(fmt.Sprintf("[%s]", truncateName(r.Name, 12)))
		}

		sb.WriteString("\n")

		if len(switches) > 0 || len(others) > 0 {
			sb.WriteString("      ")

			for range routers {
				sb.WriteString("    |       ")
			}

			sb.WriteString("\n")
		}
	}

	// Draw switches in middle
	if len(switches) > 0 {
		sb.WriteString("                        SWITCHES\n")
		sb.WriteString("  ")

		for i, s := range switches {
			if i > 0 {
				sb.WriteString("    ")
			}

			sb.WriteString(fmt.Sprintf("[%s]", truncateName(s.Name, 12)))
		}

		sb.WriteString("\n")

		if len(others) > 0 {
			sb.WriteString("      ")

			for range switches {
				sb.WriteString("    |       ")
			}

			sb.WriteString("\n")
		}
	}

	// Draw other devices at bottom
	if len(others) > 0 {
		sb.WriteString("                        DEVICES\n")
		sb.WriteString("  ")

		for i, o := range others {
			if i > 0 {
				sb.WriteString("    ")
			}

			sb.WriteString(fmt.Sprintf("[%s]", truncateName(o.Name, 12)))
		}

		sb.WriteString("\n")
	}

	return sb.String()
}

// truncateName truncates a name to fit in the diagram.
func truncateName(name string, maxLen int) string {
	if len(name) <= maxLen {
		return name
	}

	if maxLen <= 3 {
		return name[:maxLen]
	}

	return name[:maxLen-3] + "..."
}

// renderConfigDiff renders the config diff view.
func (m model) renderConfigDiff() string {
	var panel strings.Builder

	panel.WriteString("╔══════════════════════════════════════════════════════════════════╗\n")
	panel.WriteString("║                    Configuration Diff                            ║\n")
	panel.WriteString("╠══════════════════════════════════════════════════════════════════╣\n")

	if len(m.configDiffContent) == 0 {
		panel.WriteString(padPanelLine("No diff content available"))
		panel.WriteString("╚══════════════════════════════════════════════════════════════════╝")

		return panel.String()
	}

	// Calculate visible lines
	maxLines := 15
	totalLines := len(m.configDiffContent)

	startLine := m.configDiffScrollY
	if startLine >= totalLines {
		startLine = max(totalLines-1, 0)
	}

	endLine := min(startLine+maxLines, totalLines)

	for i := startLine; i < endLine; i++ {
		panel.WriteString(padPanelLine(m.configDiffContent[i]))
	}

	if totalLines > maxLines {
		panel.WriteString("╠══════════════════════════════════════════════════════════════════╣\n")
		panel.WriteString(
			padPanelLine(
				fmt.Sprintf("Lines %d-%d of %d (use arrows/PgUp/PgDn to scroll)", startLine+1, endLine, totalLines),
			),
		)
	}

	panel.WriteString("╠══════════════════════════════════════════════════════════════════╣\n")
	panel.WriteString(padPanelLine("[C] or [ESC] Close  [r] Reload config to update baseline"))
	panel.WriteString("╚══════════════════════════════════════════════════════════════════╝")

	return panel.String()
}

// renderSearch renders the search panel.
func (m model) renderSearch() string {
	var panel strings.Builder

	panel.WriteString("╔══════════════════════════════════════════════════════════════════╗\n")
	panel.WriteString("║                         Search Mode                              ║\n")
	panel.WriteString("╠══════════════════════════════════════════════════════════════════╣\n")

	// Search input
	queryDisplay := m.searchQuery
	if queryDisplay == "" {
		queryDisplay = "_"
	}

	panel.WriteString(padPanelLine("Query: " + queryDisplay))
	panel.WriteString(padPanelLine("Category: [" + m.searchCategory + "] (Tab to cycle)"))
	panel.WriteString("╠══════════════════════════════════════════════════════════════════╣\n")

	if len(m.searchResults) == 0 {
		if m.searchQuery == "" {
			panel.WriteString(padPanelLine("Type to search devices, logs, and errors"))
		} else {
			panel.WriteString(padPanelLine("No results found"))
		}
	} else {
		panel.WriteString(padPanelLine(fmt.Sprintf("Results: %d found", len(m.searchResults))))
		panel.WriteString("╠══════════════════════════════════════════════════════════════════╣\n")

		// Show up to 10 results
		maxResults := 10

		start := 0
		if m.selectedResult >= maxResults {
			start = m.selectedResult - maxResults + 1
		}

		end := min(start+maxResults, len(m.searchResults))

		for i := start; i < end; i++ {
			result := m.searchResults[i]

			prefix := "  "
			if i == m.selectedResult {
				prefix = selectedStyle.Render("->")
			}

			categoryTag := "[" + result.Category + "]"

			line := prefix + " " + categoryTag + " " + result.Title
			if len(line) > 64 {
				line = line[:61] + "..."
			}

			panel.WriteString(padPanelLine(line))

			// Show detail for selected item
			if i == m.selectedResult && result.Detail != "" {
				detailLine := "     " + result.Detail
				if len(detailLine) > 64 {
					detailLine = detailLine[:61] + "..."
				}

				panel.WriteString(padPanelLine(statsStyle.Render(detailLine)))
			}
		}
	}

	panel.WriteString("╠══════════════════════════════════════════════════════════════════╣\n")
	panel.WriteString(padPanelLine("[Enter] Select  [Tab] Category  [ESC] Close"))
	panel.WriteString("╚══════════════════════════════════════════════════════════════════╝")

	return panel.String()
}

// renderExport renders the export panel.
func (m model) renderExport() string {
	var panel strings.Builder

	panel.WriteString("╔══════════════════════════════════════════════════════════════════╗\n")
	panel.WriteString("║                      Export Statistics                           ║\n")
	panel.WriteString("╠══════════════════════════════════════════════════════════════════╣\n")

	panel.WriteString(padPanelLine("Export current simulation statistics to a file."))
	panel.WriteString(padPanelLine(""))

	// Format selection
	jsonSelected := "  "
	csvSelected := "  "

	if m.exportFormat == formatJSON {
		jsonSelected = selectedStyle.Render("->")
	} else {
		csvSelected = selectedStyle.Render("->")
	}

	panel.WriteString(padPanelLine(jsonSelected + " [j] JSON - Structured data format"))
	panel.WriteString(padPanelLine(csvSelected + " [c] CSV  - Spreadsheet compatible"))
	panel.WriteString(padPanelLine(""))

	// Stats preview
	panel.WriteString("╠══════════════════════════════════════════════════════════════════╣\n")
	panel.WriteString(padPanelLine(diffHeaderStyle.Render("Data to export:")))
	panel.WriteString(padPanelLine("  Devices:         " + strconv.Itoa(len(m.cfg.Devices))))
	panel.WriteString(padPanelLine("  Active Errors:   " + strconv.Itoa(m.errorsActive)))
	panel.WriteString(
		padPanelLine(fmt.Sprintf("  Packets RX/TX:   %d / %d", m.stackStats.PacketsReceived, m.stackStats.PacketsSent)),
	)
	panel.WriteString(padPanelLine("  Uptime:          " + formatDuration(m.uptime)))

	// Last export info
	if !m.lastExportTime.IsZero() {
		panel.WriteString(padPanelLine(""))
		panel.WriteString(padPanelLine(successStyle.Render("Last export: " + m.lastExportPath)))
	}

	panel.WriteString("╠══════════════════════════════════════════════════════════════════╣\n")
	panel.WriteString(padPanelLine("[Enter] Export  [j] JSON  [c] CSV  [ESC] Cancel"))
	panel.WriteString("╚══════════════════════════════════════════════════════════════════╝")

	return panel.String()
}

// renderTopology renders the topology view.
func (m model) renderTopology() string {
	var panel strings.Builder

	panel.WriteString("╔══════════════════════════════════════════════════════════════════╗\n")
	panel.WriteString("║                      Network Topology                            ║\n")
	panel.WriteString("╠══════════════════════════════════════════════════════════════════╣\n")

	if m.topologyContent == "" {
		panel.WriteString(padPanelLine("No topology data available"))
		panel.WriteString("╚══════════════════════════════════════════════════════════════════╝")

		return panel.String()
	}

	lines := strings.Split(m.topologyContent, "\n")

	// Calculate visible lines
	maxLines := 18
	totalLines := len(lines)

	startLine := m.topologyScrollY
	if startLine >= totalLines {
		startLine = max(totalLines-1, 0)
	}

	endLine := min(startLine+maxLines, totalLines)

	for i := startLine; i < endLine; i++ {
		panel.WriteString(padPanelLine(lines[i]))
	}

	if totalLines > maxLines {
		panel.WriteString("╠══════════════════════════════════════════════════════════════════╣\n")
		panel.WriteString(
			padPanelLine(fmt.Sprintf("Lines %d-%d of %d (use arrows/PgUp/PgDn)", startLine+1, endLine, totalLines)),
		)
	}

	panel.WriteString("╠══════════════════════════════════════════════════════════════════╣\n")
	panel.WriteString(padPanelLine("[T] or [ESC] Close"))
	panel.WriteString("╚══════════════════════════════════════════════════════════════════╝")

	return panel.String()
}

// handleAlertConfigInput handles keyboard input in alert config panel.
func (m model) handleAlertConfigInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	alertTypes := []string{"CPU", "Memory", "Disk", "PacketLoss", "Latency"}

	switch msg.String() {
	case keyEsc:
		m.showAlertConfig = false
		m.statusMessage = "Alert configuration saved"
		m.statusIsError = false

		return m, nil

	case "up":
		if m.selectedAlertType > 0 {
			m.selectedAlertType--
		}

		return m, nil

	case keyDown:
		if m.selectedAlertType < len(alertTypes)-1 {
			m.selectedAlertType++
		}

		return m, nil

	case "left":
		// Decrease threshold by 5%
		alertType := alertTypes[m.selectedAlertType]
		if m.alertThresholds == nil {
			m.alertThresholds = make(map[string]int)
		}

		if m.alertThresholds[alertType] > 0 {
			m.alertThresholds[alertType] -= 5
		}

		return m, nil

	case "right":
		// Increase threshold by 5%
		alertType := alertTypes[m.selectedAlertType]
		if m.alertThresholds == nil {
			m.alertThresholds = make(map[string]int)
		}

		if m.alertThresholds[alertType] < 100 {
			m.alertThresholds[alertType] += 5
		}

		return m, nil

	case keyEnter:
		// Toggle individual alert
		m.alertsEnabled = !m.alertsEnabled

		return m, nil

	case " ":
		// Toggle all alerts
		m.alertsEnabled = !m.alertsEnabled

		return m, nil
	}

	return m, nil
}

// renderAlertConfig renders the alert configuration panel.
func (m model) renderAlertConfig() string {
	var panel strings.Builder

	panel.WriteString("╔══════════════════════════════════════════════════════════════════╗\n")
	panel.WriteString(padPanelLine(diffHeaderStyle.Render("Alert Configuration")))
	panel.WriteString("╠══════════════════════════════════════════════════════════════════╣\n")

	alertTypes := []string{"CPU", "Memory", "Disk", "PacketLoss", "Latency"}
	alertDescriptions := map[string]string{
		"CPU":        "High CPU usage alert",
		"Memory":     "High memory usage alert",
		"Disk":       "High disk usage alert",
		"PacketLoss": "Packet loss threshold",
		"Latency":    "High latency alert (ms)",
	}

	// Initialize thresholds if nil
	if m.alertThresholds == nil {
		// These would normally be initialized elsewhere
		panel.WriteString(padPanelLine("No alert thresholds configured"))
	} else {
		for i, alertType := range alertTypes {
			threshold := m.alertThresholds[alertType]
			if threshold == 0 {
				threshold = 80 // Default threshold
			}

			// Create visual threshold bar
			barWidth := 20
			filledWidth := (threshold * barWidth) / 100
			bar := strings.Repeat("█", filledWidth) + strings.Repeat("░", barWidth-filledWidth)

			// Highlight selected row
			prefix := "  "
			if i == m.selectedAlertType {
				prefix = selectedStyle.Render("→ ")
			}

			line := fmt.Sprintf("%s%-12s [%s] %3d%% - %s",
				prefix, alertType, bar, threshold, alertDescriptions[alertType])
			panel.WriteString(padPanelLine(line))
		}
	}

	panel.WriteString("╠══════════════════════════════════════════════════════════════════╣\n")

	enabledStatus := "DISABLED"
	if m.alertsEnabled {
		enabledStatus = successStyle.Render("ENABLED")
	}

	panel.WriteString(padPanelLine(fmt.Sprintf("Alerts: %s  [Space] Toggle All", enabledStatus)))
	panel.WriteString("╠══════════════════════════════════════════════════════════════════╣\n")
	panel.WriteString(padPanelLine("[↑↓] Navigate  [←→] Adjust  [Enter] Toggle  [ESC] Save & Close"))
	panel.WriteString("╚══════════════════════════════════════════════════════════════════╝")

	return panel.String()
}

// handlePcapReplayInput handles keyboard input in PCAP replay panel.
func (m model) handlePcapReplayInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case keyEsc:
		m.showPcapReplay = false
		m.pcapPlaying = false
		m.statusMessage = "PCAP replay closed"
		m.statusIsError = false

		return m, nil

	case " ":
		// Toggle play/pause
		m.pcapPlaying = !m.pcapPlaying
		if m.pcapPlaying {
			m.statusMessage = "PCAP playback started"
		} else {
			m.statusMessage = "PCAP playback paused"
		}

		m.statusIsError = false

		return m, nil

	case "left":
		// Step backward one packet
		if m.pcapPlaybackIndex > 0 {
			m.pcapPlaybackIndex--
		}

		return m, nil

	case "right":
		// Step forward one packet
		if m.pcapPlaybackIndex < len(m.pcapPackets)-1 {
			m.pcapPlaybackIndex++
		}

		return m, nil

	case "+", "=":
		// Increase playback speed
		if m.pcapPlaybackSpeed < 4.0 {
			m.pcapPlaybackSpeed *= 2
		}

		return m, nil

	case "-", "_":
		// Decrease playback speed
		if m.pcapPlaybackSpeed > 0.25 {
			m.pcapPlaybackSpeed /= 2
		}

		return m, nil

	case "r":
		// Restart from beginning
		m.pcapPlaybackIndex = 0
		m.statusMessage = "PCAP replay restarted"
		m.statusIsError = false

		return m, nil
	}

	return m, nil
}

// renderPcapReplay renders the PCAP replay control panel.
func (m model) renderPcapReplay() string {
	var panel strings.Builder

	panel.WriteString("╔══════════════════════════════════════════════════════════════════╗\n")
	panel.WriteString(padPanelLine(diffHeaderStyle.Render("PCAP Replay Control")))
	panel.WriteString("╠══════════════════════════════════════════════════════════════════╣\n")

	if len(m.pcapPackets) == 0 {
		panel.WriteString(padPanelLine("No PCAP file loaded"))
		panel.WriteString(padPanelLine("Use the hex dump viewer to capture packets"))
	} else {
		// Playback status
		status := "⏸ PAUSED"
		if m.pcapPlaying {
			status = successStyle.Render("▶ PLAYING")
		}

		panel.WriteString(padPanelLine(fmt.Sprintf("Status: %s  Speed: %.2fx", status, m.pcapPlaybackSpeed)))
		panel.WriteString(padPanelLine(fmt.Sprintf("Packet: %d / %d", m.pcapPlaybackIndex+1, len(m.pcapPackets))))

		// Progress bar
		barWidth := 50

		progress := 0
		if len(m.pcapPackets) > 0 {
			progress = (m.pcapPlaybackIndex * barWidth) / len(m.pcapPackets)
		}

		progressBar := "[" + strings.Repeat("=", progress) + strings.Repeat("-", barWidth-progress) + "]"
		panel.WriteString(padPanelLine(progressBar))

		panel.WriteString("╠══════════════════════════════════════════════════════════════════╣\n")

		// Current packet info
		if m.pcapPlaybackIndex >= 0 && m.pcapPlaybackIndex < len(m.pcapPackets) {
			pkt := m.pcapPackets[m.pcapPlaybackIndex]
			panel.WriteString(padPanelLine("Time: " + pkt.Timestamp.Format("15:04:05.000")))
			panel.WriteString(padPanelLine(fmt.Sprintf("Protocol: %s  Length: %d bytes", pkt.Protocol, pkt.Length)))
			panel.WriteString(padPanelLine("Src: " + pkt.SrcAddr + " → Dst: " + pkt.DstAddr))
		}
	}

	panel.WriteString("╠══════════════════════════════════════════════════════════════════╣\n")
	panel.WriteString(padPanelLine("[Space] Play/Pause  [←→] Step  [+/-] Speed  [r] Restart"))
	panel.WriteString(padPanelLine("[ESC] Close"))
	panel.WriteString("╚══════════════════════════════════════════════════════════════════╝")

	return panel.String()
}

// renderHistory renders the run history viewer panel.
func (m model) renderHistory() string {
	var panel strings.Builder

	panel.WriteString("╔══════════════════════════════════════════════════════════════════╗\n")
	panel.WriteString(padPanelLine(diffHeaderStyle.Render("Run History")))
	panel.WriteString("╠══════════════════════════════════════════════════════════════════╣\n")

	if len(m.historyEntries) == 0 {
		panel.WriteString(padPanelLine("No history entries recorded yet"))
		panel.WriteString(padPanelLine("History is recorded when simulations start/stop"))
	} else {
		// Header row
		panel.WriteString(
			padPanelLine(
				fmt.Sprintf(
					"%-20s %-10s %-8s %-12s %-12s",
					"Date/Time",
					"Duration",
					"Devices",
					"Packets RX",
					"Packets TX",
				),
			),
		)
		panel.WriteString(padPanelLine(strings.Repeat("-", 64)))

		// History entries
		startIdx := m.historyScrollY
		endIdx := min(startIdx+8, len(m.historyEntries))

		for i := startIdx; i < endIdx; i++ {
			entry := m.historyEntries[i]
			duration := entry.EndTime.Sub(entry.StartTime)

			prefix := "  "
			if i == m.selectedHistoryIdx {
				prefix = selectedStyle.Render("→ ")
			}

			line := fmt.Sprintf("%s%-20s %-10s %-8d %-12d %-12d",
				prefix,
				entry.StartTime.Format("2006-01-02 15:04"),
				formatDuration(duration),
				entry.DeviceCount,
				entry.PacketsReceived,
				entry.PacketsSent,
			)
			panel.WriteString(padPanelLine(line))
		}
	}

	panel.WriteString("╠══════════════════════════════════════════════════════════════════╣\n")
	panel.WriteString(padPanelLine(fmt.Sprintf("Total runs: %d", len(m.historyEntries))))
	panel.WriteString("╠══════════════════════════════════════════════════════════════════╣\n")
	panel.WriteString(padPanelLine("[↑↓] Navigate  [Enter] Details  [PgUp/PgDn] Scroll  [ESC] Close"))
	panel.WriteString("╚══════════════════════════════════════════════════════════════════╝")

	return panel.String()
}

// renderSnmpWalk renders the SNMP walk browser panel.
func (m model) renderSnmpWalk() string {
	var panel strings.Builder

	panel.WriteString("╔══════════════════════════════════════════════════════════════════╗\n")
	panel.WriteString(padPanelLine(diffHeaderStyle.Render("SNMP Walk Browser")))
	panel.WriteString("╠══════════════════════════════════════════════════════════════════╣\n")

	if len(m.snmpOidTree) == 0 {
		panel.WriteString(padPanelLine("No SNMP data available"))
		panel.WriteString(padPanelLine("SNMP must be enabled in the simulation"))
	} else {
		// Display OID tree
		startIdx := m.snmpScrollY
		endIdx := min(startIdx+12, len(m.snmpOidTree))

		for i := startIdx; i < endIdx; i++ {
			oid := m.snmpOidTree[i]

			prefix := "  "
			if i == m.selectedSnmpOid {
				prefix = selectedStyle.Render("→ ")
			}

			// Truncate long values
			value := oid.Value
			if len(value) > 30 {
				value = value[:27] + "..."
			}

			line := fmt.Sprintf("%s%-25s %-8s %s", prefix, oid.Name, oid.Type, value)
			panel.WriteString(padPanelLine(line))
		}
	}

	panel.WriteString("╠══════════════════════════════════════════════════════════════════╣\n")
	panel.WriteString(padPanelLine(fmt.Sprintf("OIDs: %d", len(m.snmpOidTree))))
	panel.WriteString("╠══════════════════════════════════════════════════════════════════╣\n")
	panel.WriteString(padPanelLine("[↑↓] Navigate  [Enter] Expand  [/] Search  [ESC] Close"))
	panel.WriteString("╚══════════════════════════════════════════════════════════════════╝")

	return panel.String()
}

// renderDeviceConfig renders the device configuration panel.
func (m model) renderDeviceConfig() string {
	var panel strings.Builder

	panel.WriteString("╔══════════════════════════════════════════════════════════════════╗\n")

	// Get current device
	var device *config.Device

	deviceName := "No Device Selected"

	if m.selectedDeviceIdx >= 0 && m.selectedDeviceIdx < len(m.cfg.Devices) {
		device = &m.cfg.Devices[m.selectedDeviceIdx]
		deviceName = device.Name
	}

	panel.WriteString(padPanelLine(diffHeaderStyle.Render("Device Configuration: " + deviceName)))
	panel.WriteString("╠══════════════════════════════════════════════════════════════════╣\n")

	// Tab bar
	tabs := []string{"General", "Interfaces", "Protocols", "SNMP"}

	var tabBar strings.Builder

	for i, tab := range tabs {
		if i == m.deviceConfigTab {
			tabBar.WriteString(selectedStyle.Render("[" + tab + "]"))
		} else {
			tabBar.WriteString(" " + tab + " ")
		}

		if i < len(tabs)-1 {
			tabBar.WriteString(" | ")
		}
	}

	panel.WriteString(padPanelLine(tabBar.String()))
	panel.WriteString("╠══════════════════════════════════════════════════════════════════╣\n")

	if device == nil {
		panel.WriteString(padPanelLine("Select a device with [D] to view configuration"))
	} else {
		switch m.deviceConfigTab {
		case 0: // General
			panel.WriteString(padPanelLine("Name:        " + device.Name))
			panel.WriteString(padPanelLine("Type:        " + device.Type))
			panel.WriteString(padPanelLine("MAC Address: " + device.MACAddress.String()))

			if len(device.IPAddresses) > 0 {
				panel.WriteString(padPanelLine("IP Address:  " + device.IPAddresses[0].String()))
			}

			if device.VLAN > 0 {
				panel.WriteString(padPanelLine(fmt.Sprintf("VLAN:        %d", device.VLAN)))
			}

			panel.WriteString(padPanelLine("Babble:      " + boolToEnabled(device.Babble)))

		case 1: // Interfaces
			if len(device.Interfaces) == 0 {
				panel.WriteString(padPanelLine("No interfaces configured"))
			} else {
				panel.WriteString(
					padPanelLine(fmt.Sprintf("%-15s %-10s %-10s %-8s", "Interface", "Speed", "Duplex", "Status")),
				)
				panel.WriteString(padPanelLine(strings.Repeat("-", 50)))

				for _, iface := range device.Interfaces {
					status := iface.AdminStatus
					if status == "" {
						status = "up"
					}

					panel.WriteString(padPanelLine(fmt.Sprintf("%-15s %-10d %-10s %-8s",
						iface.Name, iface.Speed, iface.Duplex, status)))
				}
			}

		case 2: // Protocols
			panel.WriteString(padPanelLine("LLDP:    " + boolToEnabled(device.LLDPConfig != nil)))
			panel.WriteString(padPanelLine("CDP:     " + boolToEnabled(device.CDPConfig != nil)))
			panel.WriteString(padPanelLine("STP:     " + boolToEnabled(device.STPConfig != nil)))
			panel.WriteString(padPanelLine("EDP:     " + boolToEnabled(device.EDPConfig != nil)))
			panel.WriteString(padPanelLine("FDP:     " + boolToEnabled(device.FDPConfig != nil)))

		case 3: // SNMP
			if device.SNMPConfig.Community != "" {
				panel.WriteString(padPanelLine("Community:  " + device.SNMPConfig.Community))
				panel.WriteString(padPanelLine("SysName:    " + device.SNMPConfig.SysName))
				panel.WriteString(padPanelLine("SysDescr:   " + device.SNMPConfig.SysDescr))
				panel.WriteString(padPanelLine("Contact:    " + device.SNMPConfig.SysContact))
				panel.WriteString(padPanelLine("Location:   " + device.SNMPConfig.SysLocation))
			} else {
				panel.WriteString(padPanelLine("SNMP not configured for this device"))
			}
		}
	}

	panel.WriteString("╠══════════════════════════════════════════════════════════════════╣\n")
	panel.WriteString(padPanelLine("[Tab] Switch Tab  [↑↓] Scroll  [ESC] Close"))
	panel.WriteString("╚══════════════════════════════════════════════════════════════════╝")

	return panel.String()
}

// boolToEnabled returns "Enabled" or "Disabled" based on the boolean.
func boolToEnabled(b bool) string {
	if b {
		return successStyle.Render("Enabled")
	}

	return "Disabled"
}

// handleScrollInput is a shared helper for handling scroll navigation in list views.
// It updates selectedIdx and scrollY based on the key pressed.
// Returns true if a navigation key was handled, false otherwise.
// visibleRows is the number of visible rows in the viewport.
func handleScrollInput(
	key string,
	selectedIdx, scrollY, listLen, visibleRows int,
) (newSelectedIdx, newScrollY int, handled bool) {
	switch key {
	case "up":
		if selectedIdx > 0 {
			selectedIdx--
			if selectedIdx < scrollY {
				scrollY = selectedIdx
			}
		}

		return selectedIdx, scrollY, true
	case keyDown:
		if selectedIdx < listLen-1 {
			selectedIdx++
			if selectedIdx >= scrollY+visibleRows {
				scrollY++
			}
		}

		return selectedIdx, scrollY, true
	case "pgup":
		scrollY -= visibleRows
		if scrollY < 0 {
			scrollY = 0
		}

		return selectedIdx, scrollY, true
	case "pgdown":
		scrollY += visibleRows
		maxScroll := max(listLen-visibleRows, 0)
		scrollY = min(scrollY, maxScroll)

		return selectedIdx, scrollY, true
	}

	return selectedIdx, scrollY, false
}

// handleHistoryInput handles keyboard input in history viewer.
func (m model) handleHistoryInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	if key == keyEsc {
		m.showHistory = false

		return m, nil
	}

	const visibleRows = 8

	newIdx, newScroll, handled := handleScrollInput(
		key,
		m.selectedHistoryIdx,
		m.historyScrollY,
		len(m.historyEntries),
		visibleRows,
	)
	if handled {
		m.selectedHistoryIdx = newIdx
		m.historyScrollY = newScroll
	}

	return m, nil
}

// handleSnmpWalkInput handles keyboard input in SNMP walk browser.
func (m model) handleSnmpWalkInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	if key == keyEsc {
		m.showSnmpWalk = false

		return m, nil
	}

	const visibleRows = 12

	newIdx, newScroll, handled := handleScrollInput(
		key,
		m.selectedSnmpOid,
		m.snmpScrollY,
		len(m.snmpOidTree),
		visibleRows,
	)
	if handled {
		m.selectedSnmpOid = newIdx
		m.snmpScrollY = newScroll
	}

	return m, nil
}

// handleDeviceConfigInput handles keyboard input in device config panel.
func (m model) handleDeviceConfigInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case keyEsc:
		m.showDeviceConfig = false

		return m, nil
	case "tab":
		m.deviceConfigTab = (m.deviceConfigTab + 1) % 4
		m.deviceConfigScrollY = 0

		return m, nil
	case "up":
		if m.deviceConfigScrollY > 0 {
			m.deviceConfigScrollY--
		}

		return m, nil
	case keyDown:
		m.deviceConfigScrollY++

		return m, nil
	}

	return m, nil
}
