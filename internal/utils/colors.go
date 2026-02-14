package utils

import (
	"fmt"

	"github.com/fatih/color"
)

// Color functions for consistent output across the application
var (
	// Success - Green color for successful operations
	Success = color.New(color.FgGreen).SprintFunc()

	// Error - Red color for critical errors
	Error = color.New(color.FgRed).SprintFunc()

	// Warning - Yellow color for important warnings
	Warning = color.New(color.FgYellow).SprintFunc()

	// Info - Blue color for important information and headers
	Info = color.New(color.FgBlue).SprintFunc()
)

// PrintSuccess prints a success message in green
func PrintSuccess(format string, a ...interface{}) {
	fmt.Printf("%s\n", Success(fmt.Sprintf(format, a...)))
}

// PrintError prints an error message in red
func PrintError(format string, a ...interface{}) {
	fmt.Printf("%s\n", Error(fmt.Sprintf(format, a...)))
}

// PrintWarning prints a warning message in yellow
func PrintWarning(format string, a ...interface{}) {
	fmt.Printf("%s\n", Warning(fmt.Sprintf(format, a...)))
}

// PrintInfo prints an info message in blue
func PrintInfo(format string, a ...interface{}) {
	fmt.Printf("%s\n", Info(fmt.Sprintf(format, a...)))
}

// PrintText prints normal text without color
func PrintText(format string, a ...interface{}) {
	fmt.Printf(format, a...)
}

// DisableColors disables color output (useful for testing or CI environments)
func DisableColors() {
	color.NoColor = true
}

// ColorsEnabled returns true if colors are enabled
func ColorsEnabled() bool {
	return !color.NoColor
}
