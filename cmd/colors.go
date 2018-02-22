// +build !windows

package cmd

import (
	"fmt"

	"github.com/fatih/color"
)

// CError a Red-colored Error string (with Ansi escape codes, supports Windows)
func CError(format string, a ...interface{}) error {
	// Append a new line for a better error readibility.
	return fmt.Errorf(color.HiRedString(format+"\n", a...))
}

// PrintRed prints a red text into the terminal
func PrintRed(format string, a ...interface{}) {
	color.Red(format, a...)
}

// PrintGreen prints a green text into the terminal
func PrintGreen(format string, a ...interface{}) {
	color.Red(format, a...)
}

// PrintYellow prints a yellow text into the terminal
func PrintYellow(format string, a ...interface{}) {
	color.Yellow(format, a...)
}

// PrintBlue prints a blue text into the terminal
func PrintBlue(format string, a ...interface{}) {
	color.Blue(format, a...)
}

// StringYellow format a text with yellow ansi escape codes
func StringYellow(text string) string {
	return color.YellowString(text)
}

// StringBlue format a text with blue ansi escape codes
func StringBlue(text string) string {
	return color.BlueString(text)
}

// StringGreen format a text with string ansi escape codes
func StringGreen(text string) string {
	return color.GreenString(text)
}

// StringRed return a Red string
func StringRed(text string) string {
	return color.RedString(text)
}

// fStringYellow forcibly return a Green string, even under Windwos too.
// should be called when a colored string is needed for "later" simple prints
// (e.g. usage text, but not in uilive updates)
func fStringGreen(text string) string {
	return color.GreenString(text)
}

// fStringYellow forcibly return a Yellow string, even under Windwos too.
// should be called when a colored string is needed for "later" simple prints
// (e.g. usage text, but not in uilive updates)
func fStringYellow(text string) string {
	return color.YellowString(text)
}

// fStringRed forcibly return a Red string, even under Windwos.
// should be called when a colored string is needed for "later" simple prints
// (e.g. usage text, but not in uilive updates)
func fStringRed(text string) string {
	return color.RedString(text)
}
