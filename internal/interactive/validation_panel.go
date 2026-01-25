package interactive

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/krisarmstrong/niac-go/internal/config"
)

// renderValidationErrors renders the validation errors section.
func (m *model) renderValidationErrors(panel *strings.Builder) {
	errorCount := len(m.validationResults.Errors)
	if errorCount == 0 {
		return
	}

	panel.WriteString(padPanelLine(validationErrorStyle.Render("ERRORS:")))

	for i, err := range m.validationResults.Errors {
		if i >= maxValidationDisplay {
			remaining := errorCount - i
			panel.WriteString(
				padPanelLine(
					validationErrorStyle.Render(
						fmt.Sprintf("  ... and %d more error(s)", remaining),
					),
				),
			)
			break
		}

		errLine := fmt.Sprintf("  [%s] %s", err.Field, err.Message)
		if len(errLine) > validationTruncate {
			errLine = errLine[:validationTruncate-minEllipsisWidth] + "..."
		}

		panel.WriteString(padPanelLine(validationErrorStyle.Render(errLine)))
	}

	panel.WriteString(padPanelLine(""))
}

// renderValidationWarnings renders the validation warnings section.
func (m *model) renderValidationWarnings(panel *strings.Builder) {
	warningCount := len(m.validationResults.Warnings)
	if warningCount == 0 {
		return
	}

	panel.WriteString(padPanelLine(validationWarningStyle.Render("WARNINGS:")))

	for i, warn := range m.validationResults.Warnings {
		if i >= maxValidationDisplay {
			remaining := warningCount - i
			panel.WriteString(
				padPanelLine(
					validationWarningStyle.Render(
						fmt.Sprintf("  ... and %d more warning(s)", remaining),
					),
				),
			)
			break
		}

		warnLine := fmt.Sprintf("  [%s] %s", warn.Field, warn.Message)
		if len(warnLine) > validationTruncate {
			warnLine = warnLine[:validationTruncate-minEllipsisWidth] + "..."
		}

		panel.WriteString(padPanelLine(validationWarningStyle.Render(warnLine)))
	}
}

// renderValidationSuccess renders the success message when no validation issues exist.
func (m *model) renderValidationSuccess(panel *strings.Builder) {
	successLine := validationSuccessStyle.Render("Configuration is valid - no errors or warnings")
	panel.WriteString(padPanelLine(successLine))
	panel.WriteString(padPanelLine(""))
	panel.WriteString(padPanelLine(fmt.Sprintf("Devices configured: %d", len(m.cfg.Devices))))
}

// renderValidationIssues renders errors and warnings when present.
func (m *model) renderValidationIssues(panel *strings.Builder) {
	errorCount := len(m.validationResults.Errors)
	warningCount := len(m.validationResults.Warnings)

	summaryLine := fmt.Sprintf("Found: %s, %s",
		validationErrorStyle.Render(fmt.Sprintf("%d error(s)", errorCount)),
		validationWarningStyle.Render(fmt.Sprintf("%d warning(s)", warningCount)))
	panel.WriteString(padPanelLine(summaryLine))
	panel.WriteString("╠══════════════════════════════════════════════════════════════════╣\n")

	m.renderValidationErrors(panel)
	m.renderValidationWarnings(panel)
}

func (m *model) renderValidation() string {
	var panel strings.Builder

	panel.WriteString("╔══════════════════════════════════════════════════════════════════╗\n")
	panel.WriteString("║                   Configuration Validation                       ║\n")
	panel.WriteString("╠══════════════════════════════════════════════════════════════════╣\n")

	if m.validationResults == nil {
		panel.WriteString(padPanelLine("No validation results available"))
		panel.WriteString("╚══════════════════════════════════════════════════════════════════╝")
		return panel.String()
	}

	errorCount := len(m.validationResults.Errors)
	warningCount := len(m.validationResults.Warnings)

	if errorCount == 0 && warningCount == 0 {
		m.renderValidationSuccess(&panel)
	} else {
		m.renderValidationIssues(&panel)
	}

	panel.WriteString("╠══════════════════════════════════════════════════════════════════╣\n")
	panel.WriteString(
		padPanelLine(validationInfoStyle.Render("Press [v] or [Esc] to close this view")),
	)
	panel.WriteString("╚══════════════════════════════════════════════════════════════════╝")

	return panel.String()
}

// handleValidationToggle toggles the validation view.
func (m *model) handleValidationToggle() (tea.Model, tea.Cmd) {
	if m.showValidation {
		m.showValidation = false

		return m, nil
	}

	validator := config.NewValidator("current config")
	m.validationResults = validator.Validate(m.cfg)
	m.showValidation = true
	m.showHelp = false
	m.showLogs = false
	m.showStats = false
	m.showNeighbors = false
	m.showHexDump = false
	m.menuVisible = false

	m.setValidationStatus()
	m.addDebugLog(fmt.Sprintf("Config validation: %d errors, %d warnings",
		len(m.validationResults.Errors), len(m.validationResults.Warnings)))

	return m, nil
}

// setValidationStatus sets the status message based on validation results.
func (m *model) setValidationStatus() {
	switch {
	case m.validationResults.HasErrors():
		m.statusMessage = errorStyle.Render(
			fmt.Sprintf("Validation found %d error(s), %d warning(s)",
				len(m.validationResults.Errors), len(m.validationResults.Warnings)),
		)
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
}
