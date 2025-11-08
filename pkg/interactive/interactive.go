// Package interactive provides a terminal user interface for network simulation control and monitoring
package interactive

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/krisarmstrong/niac-go/pkg/config"
	"github.com/krisarmstrong/niac-go/pkg/errors"
	"github.com/krisarmstrong/niac-go/pkg/logging"
)

// Styles
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

	menuStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("62")).
			Padding(1, 2)

	selectedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("170")).
			Bold(true)

	statsStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("246"))
)

// CapturedPacket stores packet data for hex dump viewer
type CapturedPacket struct {
	Timestamp time.Time
	Protocol  string
	SrcAddr   string
	DstAddr   string
	Length    int
	Data      []byte
}

const maxPacketBuffer = 20 // Keep last 20 packets

type model struct {
	cfg           *config.Config
	stateManager  *errors.StateManager
	interfaceName string
	debugLevel    int

	// Menu state
	menuVisible      bool
	menuItems        []string
	selectedItem     int
	valueInputMode   bool
	valueInputPrompt string
	valueInputBuffer string

	// View state
	showHelp    bool
	showLogs    bool
	showStats   bool
	showHexDump bool

	// Error injection state
	selectedDeviceIdx int
	selectedInterface int
	selectedErrorType int
	errorValue        int

	// Stats
	packetsTotal    int
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
}

type tickMsg time.Time

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

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Handle value input mode
		if m.valueInputMode {
			return m.handleValueInput(msg)
		}

		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit

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
				deviceIP := "no-ip"
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

		case "h", "?":
			m.showHelp = !m.showHelp
			m.showLogs = false
			m.showStats = false
			m.menuVisible = false
			return m, nil

		case "l":
			m.showLogs = !m.showLogs
			m.showHelp = false
			m.showStats = false
			m.menuVisible = false
			return m, nil

		case "s":
			m.showStats = !m.showStats
			m.showHelp = false
			m.showLogs = false
			m.showHexDump = false
			m.menuVisible = false
			return m, nil

		case "x":
			m.showHexDump = !m.showHexDump
			m.showHelp = false
			m.showLogs = false
			m.showStats = false
			m.menuVisible = false
			if m.showHexDump {
				m.hexDumpScrollY = 0
				m.statusMessage = "Hex dump viewer opened - use arrow keys to navigate, [n]/[p] for next/prev packet"
			}
			return m, nil

		case "n":
			if m.showHexDump && len(m.packetBuffer) > 0 {
				m.hexDumpPacketIndex = (m.hexDumpPacketIndex + 1) % len(m.packetBuffer)
				m.hexDumpScrollY = 0
				m.statusMessage = successStyle.Render(fmt.Sprintf("✓ Packet %d/%d", m.hexDumpPacketIndex+1, len(m.packetBuffer)))
			}
			return m, nil

		case "p":
			if m.showHexDump && len(m.packetBuffer) > 0 {
				m.hexDumpPacketIndex--
				if m.hexDumpPacketIndex < 0 {
					m.hexDumpPacketIndex = len(m.packetBuffer) - 1
				}
				m.hexDumpScrollY = 0
				m.statusMessage = successStyle.Render(fmt.Sprintf("✓ Packet %d/%d", m.hexDumpPacketIndex+1, len(m.packetBuffer)))
			}
			return m, nil

		case "c":
			m.stateManager.ClearAll()
			m.statusMessage = successStyle.Render("✓ All error injections cleared")
			m.statusIsError = false
			m.errorsActive = 0
			m.addDebugLog("All error injections cleared")
			return m, nil

		case "up":
			if m.menuVisible && m.selectedItem > 0 {
				m.selectedItem--
			} else if m.showHexDump && m.hexDumpScrollY > 0 {
				m.hexDumpScrollY--
			}
			return m, nil

		case "down":
			if m.menuVisible && m.selectedItem < len(m.menuItems)-1 {
				m.selectedItem++
			} else if m.showHexDump {
				m.hexDumpScrollY++
			}
			return m, nil

		case "pgup":
			if m.showHexDump {
				m.hexDumpScrollY -= 10
				if m.hexDumpScrollY < 0 {
					m.hexDumpScrollY = 0
				}
			}
			return m, nil

		case "pgdown":
			if m.showHexDump {
				m.hexDumpScrollY += 10
			}
			return m, nil

		case "enter":
			if m.menuVisible {
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
		return m, tickCmd()
	}

	return m, nil
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
	case "enter":
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

	case "esc":
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
		ip := "no-ip"
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

	// Hex dump viewer
	if m.showHexDump {
		s.WriteString(m.renderHexDump())
		s.WriteString("\n")
	}

	// Controls
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
	s.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("86")).Render("[x]"))
	s.WriteString(" Hex  ")
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
		deviceIP := "no-ip"
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
	help.WriteString("║  [x]     Toggle packet hex dump viewer                          ║\n")
	help.WriteString("║  [n]/[p] Navigate packets (next/previous) in hex viewer         ║\n")
	help.WriteString("║  [↑][↓]  Scroll hex dump / Navigate menu items                  ║\n")
	help.WriteString("║  [PgUp]  Page up in hex dump                                    ║\n")
	help.WriteString("║  [PgDn]  Page down in hex dump                                  ║\n")
	help.WriteString("║  [c]     Clear all error injections                             ║\n")
	help.WriteString("║  [1-7]   Quick error injection (FCS/Disc/If/Util/CPU/Mem/Disk) ║\n")
	help.WriteString("║  [q]     Quit application                                       ║\n")
	help.WriteString("║                                                                  ║\n")
	help.WriteString("║ Error Injection Workflow:                                        ║\n")
	help.WriteString("║  Method 1 (Quick Access):                                       ║\n")
	help.WriteString("║    1. Press [D] to select target device                         ║\n")
	help.WriteString("║    2. Press number key [1-7] for error type                     ║\n")
	help.WriteString("║    3. Enter value (0-100) and press [Enter]                     ║\n")
	help.WriteString("║  Method 2 (Menu):                                                ║\n")
	help.WriteString("║    1. Press [D] to select target device                         ║\n")
	help.WriteString("║    2. Press [i] to open error injection menu                    ║\n")
	help.WriteString("║    3. Use arrow keys, [Enter], type value, [Enter]              ║\n")
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

	stats.WriteString("╔══════════════════════════════════════════════════════════════════╗\n")
	stats.WriteString("║                     Detailed Statistics                          ║\n")
	stats.WriteString("╠══════════════════════════════════════════════════════════════════╣\n")
	stats.WriteString(fmt.Sprintf("║ Uptime:              %s                                    ║\n", formatDuration(m.uptime)))
	stats.WriteString(fmt.Sprintf("║ Debug Level:         %d (%s)                              ║\n", m.debugLevel, getDebugLevelName(m.debugLevel)))
	stats.WriteString(fmt.Sprintf("║ Interface:           %-40s ║\n", m.interfaceName))
	stats.WriteString("║                                                                  ║\n")
	stats.WriteString(fmt.Sprintf("║ Total Packets:       %-10d                                    ║\n", m.packetsTotal))
	stats.WriteString(fmt.Sprintf("║ Packets Injected:    %-10d                                    ║\n", m.packetsInjected))
	stats.WriteString(fmt.Sprintf("║ Active Errors:       %-10d                                    ║\n", m.errorsActive))
	stats.WriteString("║                                                                  ║\n")
	stats.WriteString(fmt.Sprintf("║ Devices Simulated:   %-10d                                    ║\n", len(m.cfg.Devices)))

	// Count SNMP-enabled devices
	snmpCount := 0
	for _, dev := range m.cfg.Devices {
		if dev.SNMPConfig.Community != "" || dev.SNMPConfig.WalkFile != "" {
			snmpCount++
		}
	}
	stats.WriteString(fmt.Sprintf("║ SNMP Devices:        %-10d                                    ║\n", snmpCount))
	stats.WriteString("║                                                                  ║\n")
	stats.WriteString("║ Memory Usage:        ~15 MB (estimated)                      ║\n")
	stats.WriteString(fmt.Sprintf("║ Start Time:          %s                                    ║\n", m.startTime.Format("15:04:05")))
	stats.WriteString("╚══════════════════════════════════════════════════════════════════╝")

	return stats.String()
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
		startLine = totalLines - 1
		if startLine < 0 {
			startLine = 0
		}
	}
	endLine := startLine + maxLines
	if endLine > totalLines {
		endLine = totalLines
	}

	// Render hex dump lines
	for line := startLine; line < endLine; line++ {
		offset := line * 16
		end := offset + 16
		if end > len(pkt.Data) {
			end = len(pkt.Data)
		}

		// Offset
		lineStr := fmt.Sprintf("║ %04x   ", offset)

		// Hex bytes
		hexStr := ""
		asciiStr := ""
		for i := offset; i < end; i++ {
			b := pkt.Data[i]
			hexStr += fmt.Sprintf("%02x ", b)
			if b >= 32 && b <= 126 {
				asciiStr += string(b)
			} else {
				asciiStr += "."
			}
		}

		// Pad hex to align ASCII column (48 chars for 16 bytes)
		hexStr = fmt.Sprintf("%-48s", hexStr)
		asciiStr = fmt.Sprintf("%-16s", asciiStr)

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

// AddPacket adds a packet to the capture buffer
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

	m.packetsTotal++
}

// Run starts the interactive mode
func Run(interfaceName string, cfg *config.Config, debugConfig *logging.DebugConfig) error {
	// Extract global debug level for now (interactive mode doesn't yet support per-protocol debug)
	debugLevel := debugConfig.GetGlobal()

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
		cfg:           cfg,
		stateManager:  stateManager,
		interfaceName: interfaceName,
		debugLevel:    debugLevel,
		menuItems:     menuItems,
		startTime:     time.Now(),
		statusMessage: "Press 'i' to open interactive menu, 'h' for help",
		debugLogs:     make([]string, 0, 100),
	}

	// Add initial log entry
	m.addDebugLog(fmt.Sprintf("Started NIAC-Go interactive mode on %s", interfaceName))
	m.addDebugLog(fmt.Sprintf("Debug level: %d (%s)", debugLevel, getDebugLevelName(debugLevel)))
	m.addDebugLog(fmt.Sprintf("Simulating %d device(s)", len(cfg.Devices)))

	// Start TUI
	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("error running program: %w", err)
	}

	return nil
}
