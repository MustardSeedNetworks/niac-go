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

type model struct {
	cfg           *config.Config
	stateManager  *errors.StateManager
	interfaceName string
	debugLevel    int

	// Menu state
	menuVisible   bool
	menuItems     []string
	selectedItem  int

	// View state
	showHelp      bool
	showLogs      bool
	showStats     bool

	// Error injection state
	selectedDevice    int
	selectedInterface int
	selectedErrorType int
	errorValue        int

	// Stats
	packetsTotal   int
	packetsInjected int
	errorsActive   int
	uptime         time.Duration
	startTime      time.Time

	// Logs
	debugLogs []string

	// Status
	statusMessage string
	statusIsError bool
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
			m.menuVisible = false
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
			}
			return m, nil

		case "down":
			if m.menuVisible && m.selectedItem < len(m.menuItems)-1 {
				m.selectedItem++
			}
			return m, nil

		case "enter":
			if m.menuVisible {
				m.handleMenuSelection()
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

	// Handle menu selections
	switch {
	case strings.Contains(selection, "FCS Errors"):
		m.injectError(errors.ErrorTypeFCS, 50)
	case strings.Contains(selection, "Packet Discards"):
		m.injectError(errors.ErrorTypeDiscards, 25)
	case strings.Contains(selection, "Interface Errors"):
		m.injectError(errors.ErrorTypeInterface, 10)
	case strings.Contains(selection, "High Utilization"):
		m.injectError(errors.ErrorTypeUtilization, 95)
	case strings.Contains(selection, "High CPU"):
		m.injectError(errors.ErrorTypeCPU, 90)
	case strings.Contains(selection, "High Memory"):
		m.injectError(errors.ErrorTypeMemory, 85)
	case strings.Contains(selection, "High Disk"):
		m.injectError(errors.ErrorTypeDisk, 95)
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

func (m *model) injectError(errorType errors.ErrorType, value int) {
	// Inject error on first device, first interface
	if len(m.cfg.Devices) == 0 {
		m.statusMessage = errorStyle.Render("✗ No devices configured")
		m.statusIsError = true
		m.addDebugLog("ERROR: No devices configured for error injection")
		return
	}

	device := m.cfg.Devices[0]
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

	// Status bar
	stats := fmt.Sprintf("Uptime: %s  |  Debug: %d (%s)  |  Packets: %d  |  Errors Active: %d  |  Injected: %d",
		formatDuration(m.uptime),
		m.debugLevel,
		getDebugLevelName(m.debugLevel),
		m.packetsTotal,
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

		s.WriteString(fmt.Sprintf("  %d. %s (%s) - %s - %s\n",
			i+1,
			device.Name,
			device.Type,
			ip,
			device.MACAddress.String(),
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

	// Menu
	if m.menuVisible {
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

	// Controls
	s.WriteString("Controls: ")
	s.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("86")).Render("[i]"))
	s.WriteString(" Menu  ")
	s.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("86")).Render("[d]"))
	s.WriteString(" Debug  ")
	s.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("86")).Render("[h]"))
	s.WriteString(" Help  ")
	s.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("86")).Render("[l]"))
	s.WriteString(" Logs  ")
	s.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("86")).Render("[s]"))
	s.WriteString(" Stats  ")
	s.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("86")).Render("[c]"))
	s.WriteString(" Clear  ")
	s.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Render("[q]"))
	s.WriteString(" Quit")

	return s.String()
}

func (m model) renderMenu() string {
	var menu strings.Builder

	menu.WriteString("╔══════════════════════════════════════════════╗\n")
	menu.WriteString("║       Interactive Error Injection Menu      ║\n")
	menu.WriteString("╠══════════════════════════════════════════════╣\n")

	for i, item := range m.menuItems {
		if i == m.selectedItem {
			menu.WriteString("║ " + selectedStyle.Render("→ "+item))
		} else {
			menu.WriteString("║   " + item)
		}
		// Pad to align the right border
		padding := 44 - len(item) - 3
		menu.WriteString(strings.Repeat(" ", padding))
		menu.WriteString("║\n")
	}

	menu.WriteString("╚══════════════════════════════════════════════╝")

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
	help.WriteString("║  [d]     Cycle debug level (QUIET→NORMAL→VERBOSE→DEBUG)         ║\n")
	help.WriteString("║  [h][?]  Toggle this help screen                                ║\n")
	help.WriteString("║  [l]     Toggle debug log viewer                                ║\n")
	help.WriteString("║  [s]     Toggle statistics viewer                               ║\n")
	help.WriteString("║  [c]     Clear all error injections                             ║\n")
	help.WriteString("║  [q]     Quit application                                       ║\n")
	help.WriteString("║                                                                  ║\n")
	help.WriteString("║ Debug Levels:                                                    ║\n")
	help.WriteString("║  0 - QUIET    Only critical errors                              ║\n")
	help.WriteString("║  1 - NORMAL   Status messages (default)                         ║\n")
	help.WriteString("║  2 - VERBOSE  Protocol details                                   ║\n")
	help.WriteString("║  3 - DEBUG    Full packet details                               ║\n")
	help.WriteString("║                                                                  ║\n")
	help.WriteString("║ Error Injection Types:                                           ║\n")
	help.WriteString("║  • FCS Errors        - Frame Check Sequence errors              ║\n")
	help.WriteString("║  • Packet Discards   - Dropped packets                          ║\n")
	help.WriteString("║  • Interface Errors  - General interface errors                 ║\n")
	help.WriteString("║  • High Utilization  - Link utilization > 95%                   ║\n")
	help.WriteString("║  • High CPU          - CPU usage > 90%                          ║\n")
	help.WriteString("║  • High Memory       - Memory usage > 85%                       ║\n")
	help.WriteString("║  • High Disk         - Disk usage > 95%                         ║\n")
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
	stats.WriteString(fmt.Sprintf("║ Memory Usage:        ~15 MB (estimated)                      ║\n"))
	stats.WriteString(fmt.Sprintf("║ Start Time:          %s                                    ║\n", m.startTime.Format("15:04:05")))
	stats.WriteString("╚══════════════════════════════════════════════════════════════════╝")

	return stats.String()
}

// Run starts the interactive mode
func Run(interfaceName string, cfg *config.Config, debugConfig *logging.DebugConfig) error {
	// Extract global debug level for now (interactive mode doesn't yet support per-protocol debug)
	debugLevel := debugConfig.GetGlobal()

	// Initialize state manager
	stateManager := errors.NewStateManager()

	// Create menu items
	menuItems := []string{
		"1. Inject FCS Errors (50%)",
		"2. Inject Packet Discards (25%)",
		"3. Inject Interface Errors (10%)",
		"4. Inject High Utilization (95%)",
		"5. Inject High CPU (90%)",
		"6. Inject High Memory (85%)",
		"7. Inject High Disk (95%)",
		"8. Configure Interface",
		"9. Clear All Errors",
		"0. Exit Menu",
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
