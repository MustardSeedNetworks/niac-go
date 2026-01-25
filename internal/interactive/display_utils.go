package interactive

import (
	"fmt"
	"time"
)

func formatDuration(d time.Duration) string {
	hours := int(d.Hours())
	minutes := int(d.Minutes()) % minutesPerHour
	seconds := int(d.Seconds()) % secondsPerMinute

	return fmt.Sprintf("%02d:%02d:%02d", hours, minutes, seconds)
}

func padPanelLine(text string) string {
	if len(text) > panelContentWidth {
		if panelContentWidth > minEllipsisWidth {
			text = text[:panelContentWidth-minEllipsisWidth] + "..."
		} else {
			text = text[:panelContentWidth]
		}
	}

	return fmt.Sprintf("║ %-64s ║\n", text)
}

func fitColumn(text string, width int) string {
	if width <= 0 {
		return ""
	}

	if len(text) > width {
		if width > minEllipsisWidth {
			text = text[:width-minEllipsisWidth] + "..."
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
		return fmt.Sprintf(
			"%dm%ds ago",
			int(elapsed.Minutes()),
			int(elapsed.Seconds())%secondsPerMinute,
		)
	}

	hours := int(elapsed.Hours())
	minutes := int(elapsed.Minutes()) % minutesPerHour

	if hours >= maxHoursNoMinutes {
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
	case debugLevelVerbose:
		return "VERBOSE"
	case debugLevelDebug:
		return "DEBUG"
	default:
		return "UNKNOWN"
	}
}

// boolToEnabled returns "Enabled" or "Disabled" based on the boolean.
func boolToEnabled(b bool) string {
	if b {
		return successStyle.Render("Enabled")
	}

	return "Disabled"
}

// truncateName truncates a name to fit in the diagram.
func truncateName(name string, maxLen int) string {
	if len(name) <= maxLen {
		return name
	}

	if maxLen <= minEllipsisWidth {
		return name[:maxLen]
	}

	return name[:maxLen-minEllipsisWidth] + "..."
}

// handleScrollInput is a shared helper for handling scroll navigation in list views.
// It updates selectedIdx and scrollY based on the key pressed.
// Returns true if a navigation key was handled, false otherwise.
// visibleRows is the number of visible rows in the viewport.
func handleScrollInput(
	key string,
	selectedIdx, scrollY, listLen, visibleRows int,
) (int, int, bool) {
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
