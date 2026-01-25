package interactive

import "github.com/charmbracelet/lipgloss"

// Styles for the interactive TUI.
//
//nolint:gochecknoglobals // Lipgloss styles are idiomatic as package-level vars
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
