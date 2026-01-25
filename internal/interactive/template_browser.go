package interactive

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/krisarmstrong/niac-go/internal/templates"
)

func (m *model) renderTemplateBrowser() string {
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
			panel.WriteString(
				"╠══════════════════════════════════════════════════════════════════╣\n",
			)
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
		if len(line) > panelContentWidth {
			line = line[:panelContentWidth-minEllipsisWidth] + "..."
		}

		panel.WriteString(padPanelLine(line))
	}

	panel.WriteString("╠══════════════════════════════════════════════════════════════════╣\n")
	panel.WriteString(padPanelLine("[up/down] Navigate  [Enter] Preview  [c] Copy  [ESC] Close"))
	panel.WriteString("╚══════════════════════════════════════════════════════════════════╝")

	return panel.String()
}

// handleTemplateToggle toggles the template browser.
func (m *model) handleTemplateToggle() (tea.Model, tea.Cmd) {
	if m.showTemplates {
		m.showTemplates = false
		m.showTemplatePreview = false
		m.templatePreviewContent = ""

		return m, nil
	}

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

	return m, nil
}

// handleTemplateAction handles 'c' key in template browser.
func (m *model) handleTemplateAction() (tea.Model, tea.Cmd) {
	if !m.showTemplates || m.showTemplatePreview {
		return m, nil
	}

	if m.selectedTemplate < 0 || m.selectedTemplate >= len(m.templateList) {
		return m, nil
	}

	templateName := m.templateList[m.selectedTemplate].Name
	m.statusMessage = successStyle.Render(
		fmt.Sprintf(
			"Template: %s - Use: niac template use %s <output.yaml>",
			templateName,
			templateName,
		),
	)
	m.statusIsError = false
	m.addDebugLog("Template path shown: " + templateName)

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
