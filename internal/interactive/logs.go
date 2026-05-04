package interactive

import (
	"fmt"
	"strings"
	"time"
)

func (m *model) addDebugLog(message string) {
	timestamp := time.Now().Format("15:04:05")
	logEntry := fmt.Sprintf("[%s] %s", timestamp, message)
	m.debugLogs = append(m.debugLogs, logEntry)

	// Keep only last maxDebugLogs log entries
	if len(m.debugLogs) > maxDebugLogs {
		m.debugLogs = m.debugLogs[len(m.debugLogs)-maxDebugLogs:]
	}
}

func (m *model) renderHelp() string {
	return `╔══════════════════════════════════════════════════════════════════╗
║                         NIAC-Go Help                             ║
╠══════════════════════════════════════════════════════════════════╣
║ Keyboard Shortcuts:                                              ║
║                                                                  ║
║  [i]     Toggle interactive error injection menu                ║
║  [D]     Cycle through devices (Shift+D)                        ║
║  [d]     Cycle debug level (QUIET→NORMAL→VERBOSE→DEBUG)         ║
║  [h][?]  Toggle this help screen                                ║
║  [l]     Toggle debug log viewer                                ║
║  [s]     Toggle statistics viewer                               ║
║  [N]/[n] Toggle neighbor discovery table                        ║
║  [x]     Toggle packet hex dump viewer                          ║
║  [t]     Toggle template browser                                ║
║  [v]     Validate configuration                                 ║
║  [r]     Reload configuration from disk                         ║
║                                                                  ║
║ New Features:                                                    ║
║  [C]     Config diff viewer (compare before/after reload)       ║
║  [e]     Quick edit config in $EDITOR                           ║
║  [E]     Export statistics to JSON/CSV file                     ║
║  [T]     Network topology view (ASCII diagram)                  ║
║  [/]     Search mode (filter devices/logs by pattern)           ║
║                                                                  ║
║ Navigation:                                                      ║
║  [n]/[p] Navigate packets (next/previous) in hex viewer         ║
║  [↑][↓]  Scroll / Navigate menu items                           ║
║  [PgUp]  Page up in scrollable views                            ║
║  [PgDn]  Page down in scrollable views                          ║
║  [c]     Clear all error injections                             ║
║  [1-7]   Quick error injection (FCS/Disc/If/Util/CPU/Mem/Disk) ║
║  [q]     Quit application                                       ║
║                                                                  ║
║ Search Mode ([/]):                                               ║
║  - Type to search devices, logs, and errors                     ║
║  - [Tab] to cycle category (all/devices/logs)                   ║
║  - [Enter] to select, [Esc] to cancel                           ║
║                                                                  ║
║ Export Mode ([E]):                                               ║
║  - [j] for JSON format, [c] for CSV format                      ║
║  - [Enter] to save file, [Esc] to cancel                        ║
║                                                                  ║
║ Debug Levels:                                                    ║
║  0 - QUIET    Only critical errors                              ║
║  1 - NORMAL   Status messages (default)                         ║
║  2 - VERBOSE  Protocol details                                   ║
║  3 - DEBUG    Full packet details                               ║
║                                                                  ║
║ Error Injection Types:                                           ║
║  • FCS Errors        - Frame Check Sequence errors (0-100)      ║
║  • Packet Discards   - Dropped packets rate (0-100)             ║
║  • Interface Errors  - General interface errors (0-100)         ║
║  • High Utilization  - Link utilization percentage (0-100)      ║
║  • High CPU          - CPU usage percentage (0-100)             ║
║  • High Memory       - Memory usage percentage (0-100)          ║
║  • High Disk         - Disk usage percentage (0-100)            ║
╚══════════════════════════════════════════════════════════════════╝`
}

func (m *model) renderLogs() string {
	var logs strings.Builder

	logs.WriteString("╔══════════════════════════════════════════════════════════════════╗\n")
	logs.WriteString("║                      Debug Log Viewer                           ║\n")
	logs.WriteString("╠══════════════════════════════════════════════════════════════════╣\n")

	if len(m.debugLogs) == 0 {
		logs.WriteString("║ No debug logs yet                                                ║\n")
	} else {
		// Show last displayedLogCount logs
		start := 0
		if len(m.debugLogs) > displayedLogCount {
			start = len(m.debugLogs) - displayedLogCount
		}

		for _, log := range m.debugLogs[start:] {
			// Pad to panelContentWidth characters for alignment
			var padded string
			if len(log) > panelContentWidth {
				padded = log[:panelContentWidth]
			} else {
				padded = log + strings.Repeat(" ", panelContentWidth-len(log))
			}

			fmt.Fprintf(&logs, "║ %s ║\n", padded)
		}
	}

	logs.WriteString("╚══════════════════════════════════════════════════════════════════╝")

	return logs.String()
}
