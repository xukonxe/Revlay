package color

import (
	"fmt"
	"runtime"
)

// ANSI color codes
const (
	ColorRed    = "\033[31m"
	ColorGreen  = "\033[32m"
	ColorYellow = "\033[33m"
	ColorCyan   = "\033[36m"
	ColorReset  = "\033[0m"
)

// isWindows checks if the OS is Windows, which might not support ANSI codes.
var isWindows = runtime.GOOS == "windows"

func colorize(colorCode, format string, a ...interface{}) string {
	if isWindows {
		return fmt.Sprintf(format, a...)
	}
	return fmt.Sprintf(colorCode+format+ColorReset, a...)
}

// Red formats a string in red.
func Red(format string, a ...interface{}) string {
	return colorize(ColorRed, format, a...)
}

// Green formats a string in green.
func Green(format string, a ...interface{}) string {
	return colorize(ColorGreen, format, a...)
}

// Yellow formats a string in yellow.
func Yellow(format string, a ...interface{}) string {
	return colorize(ColorYellow, format, a...)
}

// Cyan formats a string in cyan.
func Cyan(format string, a ...interface{}) string {
	return colorize(ColorCyan, format, a...)
} 