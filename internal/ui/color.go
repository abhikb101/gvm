package ui

import (
	"fmt"
	"os"

	"golang.org/x/term"
)

var colorEnabled = true

func init() {
	if !term.IsTerminal(int(os.Stdout.Fd())) {
		colorEnabled = false
	}
	if os.Getenv("NO_COLOR") != "" {
		colorEnabled = false
	}
}

const (
	reset   = "\033[0m"
	bold    = "\033[1m"
	dim     = "\033[2m"
	red     = "\033[31m"
	green   = "\033[32m"
	yellow  = "\033[33m"
	blue    = "\033[34m"
	magenta = "\033[35m"
	cyan    = "\033[36m"
	white   = "\033[37m"
)

func styled(style, s string) string {
	if !colorEnabled {
		return s
	}
	return style + s + reset
}

// Bold returns bold text.
func Bold(s string) string { return styled(bold, s) }

// Dim returns dimmed text.
func Dim(s string) string { return styled(dim, s) }

// Red returns red text.
func Red(s string) string { return styled(red, s) }

// Green returns green text.
func Green(s string) string { return styled(green, s) }

// Yellow returns yellow text.
func Yellow(s string) string { return styled(yellow, s) }

// Blue returns blue text.
func Blue(s string) string { return styled(blue, s) }

// Magenta returns magenta text.
func Magenta(s string) string { return styled(magenta, s) }

// Cyan returns cyan text.
func Cyan(s string) string { return styled(cyan, s) }

// Success prints a green check mark with a message.
func Success(format string, args ...any) {
	fmt.Printf("%s %s\n", Green("✓"), fmt.Sprintf(format, args...))
}

// Fail prints a red cross with a message.
func Fail(format string, args ...any) {
	fmt.Printf("%s %s\n", Red("✗"), fmt.Sprintf(format, args...))
}

// Warn prints a yellow warning with a message.
func Warn(format string, args ...any) {
	fmt.Printf("%s %s\n", Yellow("!"), fmt.Sprintf(format, args...))
}

// Info prints a blue info marker with a message.
func Info(format string, args ...any) {
	fmt.Printf("%s %s\n", Blue("→"), fmt.Sprintf(format, args...))
}

// Errorf prints a red error message to stderr.
func Errorf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "%s %s\n", Red("error:"), fmt.Sprintf(format, args...))
}
