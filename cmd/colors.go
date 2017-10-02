package cmd

import (
	"fmt"

	"github.com/fatih/color"
)

// CError a Red-colored Error string (with Ansi escape codes, supports Windows)
func CError(format string, a ...interface{}) error {
	red := color.New(color.FgHiRed).SprintfFunc()
	// Append a new line for a better error readibility.
	return fmt.Errorf(red(format+"\n"), a...)
}

// PrintRed prints a red text into the terminal
func PrintRed(format string, a ...interface{}) {
	color.Red(format, a...)
}

// PrintYellow prints a yellow text into the terminal
func PrintYellow(format string, a ...interface{}) {
	color.Yellow(format, a...)
}

// StringYellow format a text with yellow ansi escape codes
func StringYellow(text string) string {
	yellow := color.New(color.FgHiYellow).SprintfFunc()
	return yellow(text)
}
