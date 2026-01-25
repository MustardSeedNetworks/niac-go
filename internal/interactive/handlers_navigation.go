package interactive

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/krisarmstrong/niac-go/internal/templates"
)

// handleEscapeKey handles escape key presses in main view.
func (m *model) handleEscapeKey() (tea.Model, tea.Cmd) {
	if m.showValidation {
		m.showValidation = false

		return m, nil
	}

	if m.showTemplatePreview {
		m.showTemplatePreview = false
		m.templatePreviewContent = ""

		return m, nil
	}

	if m.showTemplates {
		m.showTemplates = false

		return m, nil
	}

	if m.showConfigDiff {
		m.showConfigDiff = false

		return m, nil
	}

	if m.searchMode || m.showSearch {
		m.searchMode = false
		m.showSearch = false
		m.searchQuery = ""
		m.searchResults = nil
		m.statusMessage = "Search cancelled"
		m.statusIsError = false

		return m, nil
	}

	if m.showExport {
		m.showExport = false

		return m, nil
	}

	if m.showTopology {
		m.showTopology = false

		return m, nil
	}

	if m.showAlertConfig {
		m.showAlertConfig = false
		m.statusMessage = successStyle.Render("Alert configuration saved")
		m.statusIsError = false

		return m, nil
	}

	return m, nil
}

// handleUpKey handles up arrow key navigation.
func (m *model) handleUpKey() (tea.Model, tea.Cmd) {
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
}

// handleDownKey handles down arrow key navigation.
func (m *model) handleDownKey() (tea.Model, tea.Cmd) {
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
}

// handlePageUp handles page up key for scrollable views.
func (m *model) handlePageUp() (tea.Model, tea.Cmd) {
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
}

// handlePageDown handles page down key for scrollable views.
func (m *model) handlePageDown() (tea.Model, tea.Cmd) {
	switch {
	case m.showConfigDiff:
		m.configDiffScrollY += 10
	case m.showTopology:
		m.topologyScrollY += 10
	case m.showHexDump:
		m.hexDumpScrollY += 10
	}

	return m, nil
}

// handleEnterKey handles enter key in various contexts.
func (m *model) handleEnterKey() (tea.Model, tea.Cmd) {
	if m.showTemplates && !m.showTemplatePreview {
		return m.showTemplatePreviewAction()
	}

	if m.menuVisible {
		m.handleMenuSelection()
	}

	return m, nil
}

// showTemplatePreviewAction shows the preview for selected template.
func (m *model) showTemplatePreviewAction() (tea.Model, tea.Cmd) {
	if m.selectedTemplate < 0 || m.selectedTemplate >= len(m.templateList) {
		return m, nil
	}

	tmpl, err := templates.Get(m.templateList[m.selectedTemplate].Name)
	if err != nil {
		m.statusMessage = errorStyle.Render(fmt.Sprintf("Error loading template: %v", err))
		m.statusIsError = true

		return m, nil
	}

	m.templatePreviewContent = tmpl.Content
	m.showTemplatePreview = true
	m.statusMessage = successStyle.Render(
		fmt.Sprintf("Previewing: %s - Press ESC to go back", tmpl.Name),
	)
	m.statusIsError = false

	return m, nil
}

// handleHexDumpNextPacket navigates to the next packet in hex dump viewer.
func (m *model) handleHexDumpNextPacket() (tea.Model, tea.Cmd) {
	if !m.showHexDump || len(m.packetBuffer) == 0 {
		return m, nil
	}

	m.hexDumpPacketIndex = (m.hexDumpPacketIndex + 1) % len(m.packetBuffer)
	m.hexDumpScrollY = 0
	m.statusMessage = successStyle.Render(
		fmt.Sprintf("Packet %d/%d", m.hexDumpPacketIndex+1, len(m.packetBuffer)),
	)

	return m, nil
}

// handleHexDumpPrevPacket navigates to the previous packet in hex dump viewer.
func (m *model) handleHexDumpPrevPacket() (tea.Model, tea.Cmd) {
	if !m.showHexDump || len(m.packetBuffer) == 0 {
		return m, nil
	}

	m.hexDumpPacketIndex--
	if m.hexDumpPacketIndex < 0 {
		m.hexDumpPacketIndex = len(m.packetBuffer) - 1
	}

	m.hexDumpScrollY = 0
	m.statusMessage = successStyle.Render(
		fmt.Sprintf("Packet %d/%d", m.hexDumpPacketIndex+1, len(m.packetBuffer)),
	)

	return m, nil
}
