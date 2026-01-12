// Package logging provides colored console output and debug logging utilities
package logging

import (
	"fmt"
	"os"

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

	// Control flags.
	colorsEnabled = true
)

// InitColors initializes the color system.
func InitColors(enabled bool) {
	colorsEnabled = enabled

	// Respect NO_COLOR environment variable (https://no-color.org/)
	if os.Getenv("NO_COLOR") != "" {
		colorsEnabled = false
	}

	// Disable colors if output is not a terminal
	color.NoColor = !colorsEnabled
}

// AreColorsEnabled returns whether colors are currently enabled.
func AreColorsEnabled() bool {
	return colorsEnabled
}

// Error prints an error message in red.
func Error(format string, args ...any) {
	if colorsEnabled {
		_, _ = errorColor.Printf("ERROR: "+format+"\n", args...)
	} else {
		_, _ = fmt.Fprintf(os.Stdout, "ERROR: "+format+"\n", args...)
	}
}

// Warning prints a warning message in yellow.
func Warning(format string, args ...any) {
	if colorsEnabled {
		_, _ = warningColor.Printf("WARN: "+format+"\n", args...)
	} else {
		_, _ = fmt.Fprintf(os.Stdout, "WARN: "+format+"\n", args...)
	}
}

// Success prints a success message in green.
func Success(format string, args ...any) {
	if colorsEnabled {
		_, _ = successColor.Printf("✓ "+format+"\n", args...)
	} else {
		_, _ = fmt.Fprintf(os.Stdout, "✓ "+format+"\n", args...)
	}
}

// Info prints an info message in blue.
func Info(format string, args ...any) {
	if colorsEnabled {
		_, _ = infoColor.Printf(format+"\n", args...)
	} else {
		_, _ = fmt.Fprintf(os.Stdout, format+"\n", args...)
	}
}

// Debug prints a debug message in faint white.
func Debug(format string, args ...any) {
	if colorsEnabled {
		_, _ = debugColor.Printf(format+"\n", args...)
	} else {
		_, _ = fmt.Fprintf(os.Stdout, format+"\n", args...)
	}
}

// Protocol prints a protocol-specific message with the protocol name in cyan.
func Protocol(protocol string, format string, args ...any) {
	if colorsEnabled {
		_, _ = protocolColor.Printf("[%s] ", protocol)
		_, _ = fmt.Fprintf(os.Stdout, format+"\n", args...)
	} else {
		_, _ = fmt.Fprintf(os.Stdout, "[%s] "+format+"\n", append([]any{protocol}, args...)...)
	}
}

// Device prints a device-specific message with the device name in magenta.
func Device(device string, format string, args ...any) {
	if colorsEnabled {
		_, _ = deviceColor.Printf("[%s] ", device)
		_, _ = fmt.Fprintf(os.Stdout, format+"\n", args...)
	} else {
		_, _ = fmt.Fprintf(os.Stdout, "[%s] "+format+"\n", append([]any{device}, args...)...)
	}
}

// ProtocolDebug prints a debug message for a specific protocol.
func ProtocolDebug(protocol string, debugLevel int, minLevel int, format string, args ...any) {
	if debugLevel >= minLevel {
		Protocol(protocol, format, args...)
	}
}

// DeviceDebug prints a debug message for a specific device.
func DeviceDebug(device string, debugLevel int, minLevel int, format string, args ...any) {
	if debugLevel >= minLevel {
		Device(device, format, args...)
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
