// Package ui provides consistent terminal output formatting.
// Centralizes color usage and message prefixes for uniform UX.
package ui

import (
	"fmt"

	"github.com/fatih/color"
)

// =============================================================================
// Color Definitions (package-level for consistency)
// =============================================================================

var (
	cyan   = color.New(color.FgCyan)
	green  = color.New(color.FgGreen)
	yellow = color.New(color.FgYellow)
	red    = color.New(color.FgRed)
	white  = color.New(color.FgWhite)
	dim    = color.New(color.FgHiBlack)
)

// =============================================================================
// Status Messages
// =============================================================================

// Info prints an informational message with [*] prefix.
// Use for status updates and progress indicators.
func Info(format string, a ...interface{}) {
	cyan.Printf("[*] "+format+"\n", a...)
}

// Success prints a success message with [+] prefix.
// Use for completed operations and positive results.
func Success(format string, a ...interface{}) {
	green.Printf("[+] "+format+"\n", a...)
}

// Warning prints a warning message with [!] prefix.
// Use for non-critical issues and findings that need attention.
func Warning(format string, a ...interface{}) {
	yellow.Printf("[!] "+format+"\n", a...)
}

// Error prints an error message with [-] prefix.
// Use for failures and critical issues.
func Error(format string, a ...interface{}) {
	red.Printf("[-] "+format+"\n", a...)
}

// =============================================================================
// Headers and Sections
// =============================================================================

// Header prints a section header.
func Header(format string, a ...interface{}) {
	fmt.Println()
	cyan.Printf("=== "+format+" ===\n", a...)
	fmt.Println()
}

// Phase prints a phase indicator for multi-step operations.
func Phase(num int, format string, a ...interface{}) {
	fmt.Println()
	cyan.Printf("[Phase %d] "+format+"\n", append([]interface{}{num}, a...)...)
}

// =============================================================================
// Data Output
// =============================================================================

// Item prints a list item with indentation.
func Item(format string, a ...interface{}) {
	fmt.Printf("  "+format+"\n", a...)
}

// SubItem prints a sub-item with deeper indentation.
func SubItem(format string, a ...interface{}) {
	fmt.Printf("    "+format+"\n", a...)
}

// Detail prints secondary/dim information.
func Detail(format string, a ...interface{}) {
	dim.Printf("    "+format+"\n", a...)
}

// Finding prints a notable finding (like a secret or admin).
func Finding(format string, a ...interface{}) {
	yellow.Printf("  → "+format+"\n", a...)
}

// Critical prints critical findings (secrets, credentials).
func Critical(format string, a ...interface{}) {
	red.Printf("  ⚠ "+format+"\n", a...)
}

// =============================================================================
// Progress
// =============================================================================

// Progress prints a progress indicator without prefix.
func Progress(format string, a ...interface{}) {
	fmt.Printf("  "+format+"\n", a...)
}

// Result prints a result count.
func Result(format string, a ...interface{}) {
	green.Printf("  "+format+"\n", a...)
}

// =============================================================================
// Summary Statistics
// =============================================================================

// Stat prints a statistic line for summaries.
func Stat(label string, value interface{}) {
	fmt.Printf("  %-20s %v\n", label+":", value)
}

// StatHighlight prints a highlighted statistic (e.g., secrets found).
func StatHighlight(label string, value interface{}) {
	fmt.Printf("  %-20s ", label+":")
	red.Printf("%v\n", value)
}

// =============================================================================
// Inline Colors (for complex formatting)
// =============================================================================

// Dim returns a dimmed string.
func Dim(s string) string {
	return dim.Sprint(s)
}

// Highlight returns a yellow-highlighted string.
func Highlight(s string) string {
	return yellow.Sprint(s)
}
