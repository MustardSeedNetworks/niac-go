// Package interactive provides a terminal user interface for network simulation control and monitoring
package interactive

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/krisarmstrong/niac-go/internal/apperr"
	"github.com/krisarmstrong/niac-go/internal/config"
	"github.com/krisarmstrong/niac-go/internal/logging"
	"github.com/krisarmstrong/niac-go/internal/protocols"
)

// Init initializes the interactive TUI model.
func (m *model) Init() tea.Cmd {
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
func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case reloadMsg:
		return m.handleReloadMsg(msg)
	case editorFinishedMsg:
		return m.handleEditorFinishedMsg(msg)
	case tea.KeyMsg:
		return m.handleKeyMsg(msg)
	case tickMsg:
		m.uptime = time.Since(m.startTime)
		m.errorsActive = len(m.stateManager.GetAllStates())
		m.refreshStats()

		return m, tickCmd()
	}

	return m, nil
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
	stateManager := apperr.NewStateManager()

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
		debugLogs:      make([]string, 0, maxDebugLogs),
	}

	if stack != nil {
		m.refreshStats()
	}

	// Add initial log entry
	m.addDebugLog("Started NIAC-Go interactive mode on " + interfaceName)
	m.addDebugLog(fmt.Sprintf("Debug level: %d (%s)", debugLevel, getDebugLevelName(debugLevel)))
	m.addDebugLog(fmt.Sprintf("Simulating %d device(s)", len(cfg.Devices)))

	// Start TUI
	p := tea.NewProgram(&m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("error running program: %w", err)
	}

	return nil
}
