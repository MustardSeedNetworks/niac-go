package interactive

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/krisarmstrong/niac-go/pkg/config"
	"github.com/krisarmstrong/niac-go/pkg/errors"
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

		case "c":
			m.stateManager.ClearAll()
			m.statusMessage = successStyle.Render("✓ All error injections cleared")
			m.statusIsError = false
			m.errorsActive = 0
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
				m = m.handleMenuSelection()
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

func (m model) handleMenuSelection() model {
	if m.selectedItem < 0 || m.selectedItem >= len(m.menuItems) {
		return m
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
	case strings.Contains(selection, "Exit"):
		m.menuVisible = false
	}

	return m
}

func (m model) injectError(errorType errors.ErrorType, value int) {
	// Inject error on first device, first interface
	if len(m.cfg.Devices) == 0 {
		m.statusMessage = errorStyle.Render("✗ No devices configured")
		m.statusIsError = true
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
}

func (m model) View() string {
	var s strings.Builder

	// Title
	s.WriteString(titleStyle.Render(fmt.Sprintf(" NIAC-Go Interactive Mode - %s ", m.interfaceName)))
	s.WriteString("\n\n")

	// Status bar
	stats := fmt.Sprintf("Uptime: %s  |  Packets: %d  |  Errors Active: %d  |  Injected: %d",
		formatDuration(m.uptime),
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

	// Controls
	s.WriteString("Controls: ")
	s.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("86")).Render("[i]"))
	s.WriteString(" Menu  ")
	s.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("86")).Render("[c]"))
	s.WriteString(" Clear All  ")
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

// Run starts the interactive mode
func Run(interfaceName string, cfg *config.Config, debugLevel int) error {
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
		statusMessage: "Press 'i' to open interactive menu",
	}

	// Start TUI
	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("error running program: %w", err)
	}

	return nil
}
