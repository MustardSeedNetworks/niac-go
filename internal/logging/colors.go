// Package logging provides colored console output and debug logging utilities
package logging

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"

	"github.com/fatih/color"
)

var (
	// Color functions.
	errorColor    = color.New(color.FgRed, color.Bold)
	warningColor  = color.New(color.FgYellow)
	successColor  = color.New(color.FgGreen)
	infoColor     = color.New(color.FgBlue)
	protocolColor = color.New(color.FgCyan, color.Bold)
	deviceColor   = color.New(color.FgMagenta)
	debugColor    = color.New(color.FgWhite, color.Faint)

	// Control flags - protected by colorsMu.
	colorsEnabled           = true
	output        io.Writer = os.Stdout
	colorsMu      sync.Mutex
)

// InitColors initializes the color system.
func InitColors(enabled bool) {
	colorsMu.Lock()
	defer colorsMu.Unlock()

	colorsEnabled = enabled

	// Respect NO_COLOR environment variable (https://no-color.org/)
	if os.Getenv("NO_COLOR") != "" {
		colorsEnabled = false
	}

	setColorEnabled(colorsEnabled)
}

// SetOutput sets the output writer for log messages.
// Passing nil resets output to [os.Stdout].
func SetOutput(w io.Writer) {
	colorsMu.Lock()
	defer colorsMu.Unlock()

	if w == nil {
		output = os.Stdout
	} else {
		output = w
	}
}

func setColorEnabled(enabled bool) {
	if enabled {
		errorColor.EnableColor()
		warningColor.EnableColor()
		successColor.EnableColor()
		infoColor.EnableColor()
		protocolColor.EnableColor()
		deviceColor.EnableColor()
		debugColor.EnableColor()
		return
	}

	errorColor.DisableColor()
	warningColor.DisableColor()
	successColor.DisableColor()
	infoColor.DisableColor()
	protocolColor.DisableColor()
	deviceColor.DisableColor()
	debugColor.DisableColor()
}

// AreColorsEnabled returns whether colors are currently enabled.
func AreColorsEnabled() bool {
	colorsMu.Lock()
	defer colorsMu.Unlock()

	return colorsEnabled
}

// stripControlChars removes CR, LF, and ESC from log format arguments to prevent log injection.
func stripControlChars(args []any) []any {
	cleaned := make([]any, len(args))
	for i, arg := range args {
		if s, ok := arg.(string); ok {
			cleaned[i] = strings.NewReplacer("\r", "", "\n", "", "\x1b", "").Replace(s)
		} else {
			cleaned[i] = arg
		}
	}
	return cleaned
}

// Errorf prints an error message in red.
func Errorf(format string, args ...any) {
	colorsMu.Lock()
	defer colorsMu.Unlock()

	args = stripControlChars(args)
	if colorsEnabled {
		_, _ = fmt.Fprintln(output, errorColor.Sprintf("ERROR: "+format, args...))
	} else {
		_, _ = fmt.Fprintf(output, "ERROR: "+format+"\n", args...)
	}
}

// Warningf prints a warning message in yellow.
func Warningf(format string, args ...any) {
	colorsMu.Lock()
	defer colorsMu.Unlock()

	args = stripControlChars(args)
	if colorsEnabled {
		_, _ = fmt.Fprintln(output, warningColor.Sprintf("WARN: "+format, args...))
	} else {
		_, _ = fmt.Fprintf(output, "WARN: "+format+"\n", args...)
	}
}

// Successf prints a success message in green.
func Successf(format string, args ...any) {
	colorsMu.Lock()
	defer colorsMu.Unlock()

	args = stripControlChars(args)
	if colorsEnabled {
		_, _ = fmt.Fprintln(output, successColor.Sprintf("✓ "+format, args...))
	} else {
		_, _ = fmt.Fprintf(output, "✓ "+format+"\n", args...)
	}
}

// Infof prints an info message in blue.
func Infof(format string, args ...any) {
	colorsMu.Lock()
	defer colorsMu.Unlock()

	args = stripControlChars(args)
	if colorsEnabled {
		_, _ = fmt.Fprintln(output, infoColor.Sprintf(format, args...))
	} else {
		_, _ = fmt.Fprintf(output, format+"\n", args...)
	}
}

// Debugf prints a debug message in faint white.
func Debugf(format string, args ...any) {
	colorsMu.Lock()
	defer colorsMu.Unlock()

	args = stripControlChars(args)
	if colorsEnabled {
		_, _ = fmt.Fprintln(output, debugColor.Sprintf(format, args...))
	} else {
		_, _ = fmt.Fprintf(output, format+"\n", args...)
	}
}

// Protocolf prints a protocol-specific message with the protocol name in cyan.
func Protocolf(protocol string, format string, args ...any) {
	if colorsEnabled {
		prefix := protocolColor.Sprintf("[%s] ", protocol)
		_, _ = fmt.Fprintf(output, prefix+format+"\n", args...)
	} else {
		_, _ = fmt.Fprintf(output, "[%s] "+format+"\n", append([]any{protocol}, args...)...)
	}
}

// Devicef prints a device-specific message with the device name in magenta.
func Devicef(device string, format string, args ...any) {
	if colorsEnabled {
		prefix := deviceColor.Sprintf("[%s] ", device)
		_, _ = fmt.Fprintf(output, prefix+format+"\n", args...)
	} else {
		_, _ = fmt.Fprintf(output, "[%s] "+format+"\n", append([]any{device}, args...)...)
	}
}

// ProtocolDebugf prints a debug message for a specific protocol.
func ProtocolDebugf(protocol string, debugLevel int, minLevel int, format string, args ...any) {
	if debugLevel >= minLevel {
		Protocolf(protocol, format, args...)
	}
}

// DeviceDebugf prints a debug message for a specific device.
func DeviceDebugf(device string, debugLevel int, minLevel int, format string, args ...any) {
	if debugLevel >= minLevel {
		Devicef(device, format, args...)
	}
}

// Sprintf returns a colored string without printing (useful for building messages).
func Sprintf(colorType string, format string, args ...any) string {
	var c *color.Color

	switch colorType {
	case "error":
		c = errorColor
	case "warning":
		c = warningColor
	case "success":
		c = successColor
	case "info":
		c = infoColor
	case "protocol":
		c = protocolColor
	case "device":
		c = deviceColor
	case "debug":
		c = debugColor
	default:
		return fmt.Sprintf(format, args...)
	}

	if colorsEnabled {
		return c.Sprintf(format, args...)
	}

	return fmt.Sprintf(format, args...)
}

// ErrorString returns a colored error string.
func ErrorString(s string) string {
	if colorsEnabled {
		return errorColor.Sprint(s)
	}

	return s
}

// WarningString returns a colored warning string.
func WarningString(s string) string {
	if colorsEnabled {
		return warningColor.Sprint(s)
	}

	return s
}

// SuccessString returns a colored success string.
func SuccessString(s string) string {
	if colorsEnabled {
		return successColor.Sprint(s)
	}

	return s
}

// InfoString returns a colored info string.
func InfoString(s string) string {
	if colorsEnabled {
		return infoColor.Sprint(s)
	}

	return s
}

// ProtocolString returns a colored protocol string.
func ProtocolString(s string) string {
	if colorsEnabled {
		return protocolColor.Sprint(s)
	}

	return s
}

// DeviceString returns a colored device string.
func DeviceString(s string) string {
	if colorsEnabled {
		return deviceColor.Sprint(s)
	}

	return s
}
